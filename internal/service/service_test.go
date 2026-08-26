package service

import (
	"encoding/json"
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/store"
)

// seedBatch 铺设一套最小可分析数据（3 温度、3 条校正峰，呈融合趋势），
// 返回已就绪的顶层编排服务与批次 ID。
func seedBatch(t *testing.T, st *store.Store) *Service {
	t.Helper()
	svc := New(st)
	smp, err := svc.Sample.Create(sample.CreateInput{
		Name: "ethanol", Compound: "C2H5OH", Solvent: "CDCl3", Concentration: 0.5,
	})
	if err != nil {
		t.Fatalf("create sample: %v", err)
	}
	temps, err := svc.Sample.AddTemperatures(smp.ID, []float64{25, 50, 75})
	if err != nil {
		t.Fatalf("add temps: %v", err)
	}
	tempID := map[float64]string{}
	for _, tp := range temps {
		tempID[tp.TempC] = tp.ID
	}
	batch, err := svc.CreateBatch(smp.ID, "b")
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	std, err := svc.CreateStandard(batch.ID, "TMS")
	if err != nil {
		t.Fatalf("create standard: %v", err)
	}
	for _, tp := range temps {
		if err := svc.AddStandardPoint(std.ID, tp.TempC, 0.00); err != nil {
			t.Fatalf("add std point: %v", err)
		}
	}
	if err := svc.LockStandard(std.ID); err != nil {
		t.Fatalf("lock standard: %v", err)
	}
	observedA := map[float64]float64{25: 1.50, 50: 1.40, 75: 1.30}
	observedB := map[float64]float64{25: 1.80, 50: 1.50, 75: 1.32}
	for _, tp := range temps {
		if _, err := svc.Peak.Add(peak.AddInput{
			BatchID: batch.ID, TemperatureID: tempID[tp.TempC], TempC: tp.TempC,
			Unit: "ppm", ObservedShift: 0.05, Intensity: 1.0, IsStandard: true,
		}); err != nil {
			t.Fatalf("std peak: %v", err)
		}
		if _, err := svc.Peak.Add(peak.AddInput{
			BatchID: batch.ID, TemperatureID: tempID[tp.TempC], TempC: tp.TempC,
			Unit: "ppm", ObservedShift: observedA[tp.TempC], Intensity: 0.8,
		}); err != nil {
			t.Fatalf("peak A: %v", err)
		}
		if _, err := svc.Peak.Add(peak.AddInput{
			BatchID: batch.ID, TemperatureID: tempID[tp.TempC], TempC: tp.TempC,
			Unit: "ppm", ObservedShift: observedB[tp.TempC], Intensity: 0.8,
		}); err != nil {
			t.Fatalf("peak B: %v", err)
		}
	}
	if _, err := svc.Peak.Calibrate(batch.ID); err != nil {
		t.Fatalf("calibrate: %v", err)
	}
	return svc
}

// TestReassociateReplacesPriorResult 覆盖三类回归：
//  1. 重复关联不叠加轨迹与成员；
//  2. 重算后候选证据不再引用已失效轨迹（TrackAID/TrackBID 全部可解析）；
//  3. 重新冻结的快照归属数量不随重算次数漂移。
func TestReassociateReplacesPriorResult(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()
	svc := seedBatch(t, st)

	// 取到刚创建的批次 ID
	batches, err := st.ListBatches("")
	if err != nil {
		t.Fatalf("list batches: %v", err)
	}
	if len(batches) != 1 {
		t.Fatalf("expected 1 batch, got %d", len(batches))
	}
	batchID := batches[0].ID

	// 首次关联 + 评分，确立基准轨迹集合与候选。
	first, err := svc.Track.Associate(batchID)
	if err != nil {
		t.Fatalf("associate #1: %v", err)
	}
	if _, err := svc.Exchange.Score(batchID); err != nil {
		t.Fatalf("score #1: %v", err)
	}

	// 重复关联多次，结果必须替换而非叠加。
	for i := 0; i < 3; i++ {
		if _, err := svc.Track.Associate(batchID); err != nil {
			t.Fatalf("associate #%d: %v", i+2, err)
		}
	}
	tracks, err := st.ListTracksByBatch(batchID)
	if err != nil {
		t.Fatalf("list tracks: %v", err)
	}
	if len(tracks) != len(first) {
		t.Fatalf("track count drifted after re-associate: got %d, want %d (first run)",
			len(tracks), len(first))
	}
	// 成员总数同样不可叠加：每条轨迹成员数应等于温度档数。
	for _, tr := range tracks {
		members, err := st.ListTrackMembers(tr.ID)
		if err != nil {
			t.Fatalf("list members: %v", err)
		}
		if len(members) != 3 {
			t.Fatalf("track %s member count drifted: got %d, want 3", tr.ID, len(members))
		}
	}

	// 重算后旧候选已被清空，新候选需由 score 重新生成；其引用的轨迹必须存在。
	if _, err := svc.Exchange.Score(batchID); err != nil {
		t.Fatalf("score #2: %v", err)
	}
	cands, err := st.ListCandidatesByBatch(batchID)
	if err != nil {
		t.Fatalf("list candidates: %v", err)
	}
	trackSet := map[string]bool{}
	for _, tr := range tracks {
		trackSet[tr.ID] = true
	}
	for _, c := range cands {
		if !trackSet[c.TrackAID] || !trackSet[c.TrackBID] {
			t.Fatalf("candidate %s references defunct track: a=%s b=%s", c.ID, c.TrackAID, c.TrackBID)
		}
	}

	// 快照归属数量须等于当前轨迹数，不随重算次数漂移。
	snap, err := svc.Snapshot.Freeze(batchID)
	if err != nil {
		t.Fatalf("freeze: %v", err)
	}
	var view struct {
		TrackCount     int                      `json:"track_count"`
		CandidateCount int                      `json:"candidate_count"`
		Tracks         []map[string]interface{} `json:"tracks"`
		Candidates     []model.ExchangeCandidate `json:"candidates"`
	}
	if err := json.Unmarshal([]byte(snap.Payload), &view); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if view.TrackCount != len(tracks) || len(view.Tracks) != len(tracks) {
		t.Fatalf("snapshot track attribution drifted: count=%d tracks=%d want=%d",
			view.TrackCount, len(view.Tracks), len(tracks))
	}
	if view.CandidateCount != len(cands) {
		t.Fatalf("snapshot candidate count drifted: got %d, want %d",
			view.CandidateCount, len(cands))
	}
}
