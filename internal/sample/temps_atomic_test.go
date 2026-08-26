package sample

import (
	"errors"
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// newTestService 打开一个临时 SQLite 存储并构造样品服务，用于端到端回归。
func newTestService(t *testing.T) (*Service, *model.Sample) {
	t.Helper()
	st, err := store.Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := New(st)
	smp, err := svc.Create(CreateInput{
		Name:          "ethanol",
		Compound:      "C2H5OH",
		Solvent:       "CDCl3",
		Concentration: 0.5,
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	return svc, smp
}

// TestAddTemperaturesRejectsDuplicateWithinBatchLeavesNoPartialRows 锁定：
// 一次追加的温度序列内含重复温度时，请求整体失败且不留任何部分新增温度。
func TestAddTemperaturesRejectsDuplicateWithinBatchLeavesNoPartialRows(t *testing.T) {
	svc, smp := newTestService(t)

	// 前半段合法温度先写入，建立基线序列。
	if _, err := svc.AddTemperatures(smp.ID, []float64{25, 50}); err != nil {
		t.Fatalf("initial add: %v", err)
	}
	before, err := svc.Temperatures(smp.ID)
	if err != nil {
		t.Fatalf("list before: %v", err)
	}

	// 这一批尾部带重复温度（75 出现两次），应整体失败。
	_, err = svc.AddTemperatures(smp.ID, []float64{75, 100, 75})
	if !errors.Is(err, model.ErrTempConflict) {
		t.Fatalf("got err = %v, want ErrTempConflict", err)
	}

	// 请求失败后，序列必须与失败前完全一致，没有半套新增温度。
	after, err := svc.Temperatures(smp.ID)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("temperature count changed after failed add: before=%d after=%d", len(before), len(after))
	}
	for i := range before {
		if after[i] != before[i] {
			t.Fatalf("temperature row drifted at index %d: before=%+v after=%+v", i, before[i], after[i])
		}
	}
}

// TestAddTemperaturesRejectsDuplicateAgainstExistingLeavesNoPartialRows 锁定：
// 追加的温度与已有序列重复时，同样整体失败且不留部分新增温度。
func TestAddTemperaturesRejectsDuplicateAgainstExistingLeavesNoPartialRows(t *testing.T) {
	svc, smp := newTestService(t)

	if _, err := svc.AddTemperatures(smp.ID, []float64{25, 50}); err != nil {
		t.Fatalf("initial add: %v", err)
	}
	before, err := svc.Temperatures(smp.ID)
	if err != nil {
		t.Fatalf("list before: %v", err)
	}

	// 第二个温度 50 与已有序列重复，应整体失败。
	_, err = svc.AddTemperatures(smp.ID, []float64{75, 50})
	if !errors.Is(err, model.ErrTempConflict) {
		t.Fatalf("got err = %v, want ErrTempConflict", err)
	}

	after, err := svc.Temperatures(smp.ID)
	if err != nil {
		t.Fatalf("list after: %v", err)
	}
	if len(after) != len(before) {
		t.Fatalf("temperature count changed after failed add: before=%d after=%d", len(before), len(after))
	}
}

// TestAddTemperaturesAppendsLegalSequenceInOrder 锁定：
// 完整合法的温度序列仍能按原顺序追加。
func TestAddTemperaturesAppendsLegalSequenceInOrder(t *testing.T) {
	svc, smp := newTestService(t)

	if _, err := svc.AddTemperatures(smp.ID, []float64{25, 50}); err != nil {
		t.Fatalf("initial add: %v", err)
	}
	out, err := svc.AddTemperatures(smp.ID, []float64{75, 100, 125})
	if err != nil {
		t.Fatalf("second add: %v", err)
	}

	wantTemps := []float64{75, 100, 125}
	for i, tp := range out {
		if tp.TempC != wantTemps[i] {
			t.Fatalf("out[%d].TempC = %v, want %v", i, tp.TempC, wantTemps[i])
		}
		if tp.SortOrder != 2+i {
			t.Fatalf("out[%d].SortOrder = %d, want %d", i, tp.SortOrder, 2+i)
		}
	}

	all, err := svc.Temperatures(smp.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	wantAll := []float64{25, 50, 75, 100, 125}
	if len(all) != len(wantAll) {
		t.Fatalf("len(all) = %d, want %d", len(all), len(wantAll))
	}
	for i, tp := range all {
		if tp.TempC != wantAll[i] {
			t.Fatalf("all[%d].TempC = %v, want %v", i, tp.TempC, wantAll[i])
		}
	}
}
