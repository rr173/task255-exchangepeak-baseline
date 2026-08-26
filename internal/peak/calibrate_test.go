package peak

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// seedCalibrateFixtures builds a batch with, per temperature:
//   - a ppm reference (is_standard) peak that calibration should use,
//   - a ppm analyte peak that should be corrected,
//   - an excluded/duplicate/impurity peak recorded in Hz (incompatible unit).
//
// Returning the IDs lets the test assert the outcome of Calibrate.
func seedCalibrateFixtures(t *testing.T) (*store.Store, string, map[string]string) {
	t.Helper()
	st, err := store.Open(t.TempDir() + "/calibrate.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}

	smp := &model.Sample{
		ID: uuid.NewString(), Name: "s", Compound: "C", Solvent: "CDCl3",
		Concentration: 1, CreatedAt: time.Now(),
	}
	if err := st.CreateSample(smp); err != nil {
		t.Fatalf("CreateSample: %v", err)
	}
	tp := &model.TemperaturePoint{
		ID: uuid.NewString(), SampleID: smp.ID, TempC: 25, SortOrder: 0,
	}
	if err := st.AddTemperaturePoint(tp); err != nil {
		t.Fatalf("AddTemperaturePoint: %v", err)
	}
	batch := &model.SpectrumBatch{
		ID: uuid.NewString(), SampleID: smp.ID, Label: "b",
		State: model.BatchReceiving, CreatedAt: time.Now(),
	}
	if err := st.CreateBatch(batch); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	std := &model.InternalStandard{
		ID: uuid.NewString(), BatchID: batch.ID, Name: "TMS",
		Locked: true, CreatedAt: time.Now(),
	}
	if err := st.CreateStandard(std); err != nil {
		t.Fatalf("CreateStandard: %v", err)
	}
	if err := st.AddStandardPoint(&model.InternalStandardPoint{
		ID: uuid.NewString(), StandardID: std.ID, TempC: 25, TrueShift: 0.0,
	}); err != nil {
		t.Fatalf("AddStandardPoint: %v", err)
	}

	ids := map[string]string{}
	mk := func(unit, state string, isStd bool, obs float64) string {
		id := uuid.NewString()
		if err := st.CreatePeak(&model.Peak{
			ID: id, BatchID: batch.ID, TemperatureID: tp.ID, TempC: 25,
			Unit: unit, ObservedShift: obs, CorrectedShift: obs,
			IsStandard: isStd, State: state,
		}); err != nil {
			t.Fatalf("CreatePeak: %v", err)
		}
		return id
	}
	// 有效参比峰（ppm，0.05）：校正确应选用此峰。
	ids["ref"] = mk(model.UnitPPM, model.PeakRaw, true, 0.05)
	// 有效分析峰（ppm，1.50）：应被校正为 1.50 − 0.05 = 1.45。
	ids["analyte"] = mk(model.UnitPPM, model.PeakRaw, false, 1.50)
	// 被排除峰，单位为 Hz（不兼容）：应保留记录但不参与校正。
	ids["excluded"] = mk(model.UnitHz, model.PeakExcluded, false, 300.0)
	// 杂质峰，单位为 Hz（不兼容）：应保留记录但不参与校正。
	ids["impurity"] = mk(model.UnitHz, model.PeakImpurity, false, 600.0)
	// 重复峰，单位为 Hz（不兼容）：应保留记录但不参与校正。
	ids["duplicate"] = mk(model.UnitHz, model.PeakDuplicate, false, 900.0)
	return st, batch.ID, ids
}

// TestCalibrateIgnoresExcludedImpurityPeaksWithIncompatibleUnit asserts that
// excluded/impurity/duplicate peaks recorded in Hz no longer break calibration
// of an otherwise-valid batch, and are neither selected as the reference nor
// written a corrected value.
func TestCalibrateIgnoresExcludedImpurityPeaksWithIncompatibleUnit(t *testing.T) {
	st, batchID, ids := seedCalibrateFixtures(t)
	defer st.Close()

	svc := New(st)
	offsets, err := svc.Calibrate(batchID)
	if err != nil {
		t.Fatalf("Calibrate returned error %v, want nil (excluded/impurity peaks with incompatible unit must not break calibration)", err)
	}
	if got, want := offsets[25], 0.05; got != want {
		t.Fatalf("offset[25] = %v, want %v", got, want)
	}

	// 有效分析峰应被校正为 observed − offset = 1.50 − 0.05 = 1.45。
	analyte, err := st.GetPeak(ids["analyte"])
	if err != nil {
		t.Fatalf("GetPeak analyte: %v", err)
	}
	if got, want := analyte.CorrectedShift, 1.45; got != want {
		t.Fatalf("analyte CorrectedShift = %v, want %v", got, want)
	}
	if analyte.State != model.PeakCorrected {
		t.Fatalf("analyte State = %q, want %q", analyte.State, model.PeakCorrected)
	}

	// 被排除/杂质/重复峰不应被写入校正值，状态不变。
	for label, wantState := range map[string]string{
		"excluded":  model.PeakExcluded,
		"impurity":  model.PeakImpurity,
		"duplicate": model.PeakDuplicate,
	} {
		p, err := st.GetPeak(ids[label])
		if err != nil {
			t.Fatalf("GetPeak %s: %v", label, err)
		}
		if p.State != wantState {
			t.Fatalf("%s State = %q, want %q (must not become corrected)", label, p.State, wantState)
		}
		// CorrectedShift 应保持录入时的原始值，未被校正逻辑覆盖。
		if p.CorrectedShift != p.ObservedShift {
			t.Fatalf("%s CorrectedShift = %v, want %v (must not be recalibrated)", label, p.CorrectedShift, p.ObservedShift)
		}
	}
}

// TestCalibrateFailsWhenStandardPeakIsExcluded asserts that an excluded
// reference peak is not eligible for selection, even if it is_standard.
func TestCalibrateFailsWhenStandardPeakIsExcluded(t *testing.T) {
	st, batchID, _ := seedCalibrateFixtures(t)
	defer st.Close()

	// 将唯一的有效参比峰排除，使该温度档无有效参比峰可选。
	excluded, err := st.StandardPeakAtTemp(batchID, 25)
	if err != nil {
		t.Fatalf("StandardPeakAtTemp: %v", err)
	}
	if err := st.SetPeakState(excluded.ID, model.PeakExcluded); err != nil {
		t.Fatalf("SetPeakState: %v", err)
	}

	svc := New(st)
	if _, err := svc.Calibrate(batchID); err != model.ErrNoStandardPeak {
		t.Fatalf("Calibrate error = %v, want ErrNoStandardPeak (excluded reference must not be selected)", err)
	}
}
