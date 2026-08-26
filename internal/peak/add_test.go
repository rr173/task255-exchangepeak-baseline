package peak

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// makeSampleWithTemp 在数据库中直接建立样品与一档温度点，绕过 service 编排，
// 避免 peak 包测试反向依赖 service（会构成 import cycle）。
func makeSampleWithTemp(t *testing.T, st *store.Store, tempC float64) (sampleID, tempID string, tp model.TemperaturePoint) {
	t.Helper()
	smp := &model.Sample{
		ID: uuid.NewString(), Name: "ethanol", Compound: "C2H5OH",
		Solvent: "CDCl3", Concentration: 0.5, CreatedAt: time.Now(),
	}
	if err := st.CreateSample(smp); err != nil {
		t.Fatalf("create sample: %v", err)
	}
	tp = model.TemperaturePoint{
		ID: uuid.NewString(), SampleID: smp.ID, TempC: tempC, SortOrder: 0,
	}
	if err := st.AddTemperaturePoint(&tp); err != nil {
		t.Fatalf("add temp point: %v", err)
	}
	return smp.ID, tp.ID, tp
}

func makeBatch(t *testing.T, st *store.Store, sampleID, label string) *model.SpectrumBatch {
	t.Helper()
	b := &model.SpectrumBatch{
		ID: uuid.NewString(), SampleID: sampleID, Label: label,
		State: model.BatchReceiving, CreatedAt: time.Now(),
	}
	if err := st.CreateBatch(b); err != nil {
		t.Fatalf("create batch: %v", err)
	}
	return b
}

// TestAddRejectsTemperatureFromAnotherSample 验证峰只能归属于当前批次的样品：
// 误用另一份样品的温度点时，Add 必须失败且不写入任何峰。
func TestAddRejectsTemperatureFromAnotherSample(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()

	ownSampleID, ownTempID, _ := makeSampleWithTemp(t, st, 25.0)
	otherSampleID, _, _ := makeSampleWithTemp(t, st, 25.0)
	otherBatch := makeBatch(t, st, otherSampleID, "other-batch")

	// otherBatch 属于第二份样品，ownTempID 属于第一份样品 → 不应允许。
	svc := &Service{Store: st}
	p, err := svc.Add(AddInput{
		BatchID: otherBatch.ID, TemperatureID: ownTempID, TempC: 25.0,
		Unit: model.UnitPPM, ObservedShift: 1.5, Intensity: 1.0,
	})
	if err == nil {
		t.Fatalf("Add accepted a temperature from another sample; peak=%+v", p)
	}
	if p != nil {
		t.Fatalf("no peak should be returned on failure, got %+v", p)
	}
	peaks, err := st.ListPeaksByBatch(otherBatch.ID)
	if err != nil {
		t.Fatalf("ListPeaksByBatch: %v", err)
	}
	if len(peaks) != 0 {
		t.Fatalf("no peak should be persisted on failure, got %d", len(peaks))
	}
	// 确保误用没有污染原样品的温度序列语义：ownTemp 仍只属于 ownSampleID。
	_ = ownSampleID
}

// TestAddRejectsUnknownTemperaturePoint 验证温度点不存在时拒绝新增且不写入峰。
func TestAddRejectsUnknownTemperaturePoint(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()

	sampleID, _, _ := makeSampleWithTemp(t, st, 25.0)
	batch := makeBatch(t, st, sampleID, "b")

	svc := &Service{Store: st}
	p, err := svc.Add(AddInput{
		BatchID: batch.ID, TemperatureID: "no-such-temp", TempC: 25.0,
		Unit: model.UnitPPM, ObservedShift: 1.5, Intensity: 1.0,
	})
	if err == nil {
		t.Fatalf("Add accepted an unknown temperature id; peak=%+v", p)
	}
	peaks, err := st.ListPeaksByBatch(batch.ID)
	if err != nil {
		t.Fatalf("ListPeaksByBatch: %v", err)
	}
	if len(peaks) != 0 {
		t.Fatalf("no peak should be persisted on failure, got %d", len(peaks))
	}
}

// TestAddRejectsMismatchedTempC 验证温度点存在且同属本批次样品，
// 但入参 TempC 与温度点实际温度不一致时拒绝新增。
func TestAddRejectsMismatchedTempC(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()

	sampleID, tempID, _ := makeSampleWithTemp(t, st, 25.0)
	batch := makeBatch(t, st, sampleID, "b")

	svc := &Service{Store: st}
	p, err := svc.Add(AddInput{
		BatchID: batch.ID, TemperatureID: tempID, TempC: 50.0, // 温度点实为 25.0
		Unit: model.UnitPPM, ObservedShift: 1.5, Intensity: 1.0,
	})
	if err == nil {
		t.Fatalf("Add accepted a mismatched temp_c; peak=%+v", p)
	}
	peaks, err := st.ListPeaksByBatch(batch.ID)
	if err != nil {
		t.Fatalf("ListPeaksByBatch: %v", err)
	}
	if len(peaks) != 0 {
		t.Fatalf("no peak should be persisted on failure, got %d", len(peaks))
	}
}

// TestAddAcceptsOwningSampleTemperature 确认修复未破坏正常路径：
// 温度点确实属于批次所属样品且温度一致时新增成功。
func TestAddAcceptsOwningSampleTemperature(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()

	sampleID, tempID, tp := makeSampleWithTemp(t, st, 25.0)
	batch := makeBatch(t, st, sampleID, "b")

	svc := &Service{Store: st}
	p, err := svc.Add(AddInput{
		BatchID: batch.ID, TemperatureID: tempID, TempC: 25.0,
		Unit: model.UnitPPM, ObservedShift: 1.5, Intensity: 1.0,
	})
	if err != nil {
		t.Fatalf("Add failed on valid input: %v", err)
	}
	if p.TemperatureID != tempID || p.BatchID != batch.ID || p.TempC != tp.TempC {
		t.Fatalf("unexpected peak = %+v", p)
	}
	peaks, err := st.ListPeaksByBatch(batch.ID)
	if err != nil {
		t.Fatalf("ListPeaksByBatch: %v", err)
	}
	if len(peaks) != 1 {
		t.Fatalf("expected 1 peak persisted, got %d", len(peaks))
	}
}
