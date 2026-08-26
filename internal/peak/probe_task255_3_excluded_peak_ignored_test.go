package peak_test

import (
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
	"task255-exchangepeak/internal/store"
)

func TestBug03ExcludedPeakDoesNotBlockCalibration(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	smp, err := svc.Sample.Create(sample.CreateInput{Name: "s", Compound: "c", Solvent: "d", Concentration: 1})
	if err != nil {
		t.Fatal(err)
	}
	temps, err := svc.Sample.AddTemperatures(smp.ID, []float64{25})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := svc.CreateBatch(smp.ID, "batch")
	if err != nil {
		t.Fatal(err)
	}
	std, err := svc.CreateStandard(batch.ID, "TMS")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.AddStandardPoint(std.ID, 25, 0); err != nil {
		t.Fatal(err)
	}
	if err := svc.LockStandard(std.ID); err != nil {
		t.Fatal(err)
	}
	stdPeak, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temps[0].ID, TempC: 25, Unit: model.UnitPPM, ObservedShift: 0.05, Intensity: 1, IsStandard: true})
	if err != nil {
		t.Fatal(err)
	}
	analyte, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temps[0].ID, TempC: 25, Unit: model.UnitPPM, ObservedShift: 1.2, Intensity: 1})
	if err != nil {
		t.Fatal(err)
	}
	noise, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temps[0].ID, TempC: 25, Unit: model.UnitHz, ObservedShift: 500, Intensity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Peak.MarkExcluded(noise.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Peak.Calibrate(batch.ID); err != nil {
		t.Fatalf("Calibrate() error = %v", err)
	}
	got, err := svc.Store.GetPeak(analyte.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != model.PeakCorrected || got.CorrectedShift != 1.15 {
		t.Fatalf("analyte after calibration = %+v", got)
	}
	gotStd, err := svc.Store.GetPeak(stdPeak.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotStd.State != model.PeakCorrected {
		t.Fatalf("standard peak state = %q, want corrected", gotStd.State)
	}
}
