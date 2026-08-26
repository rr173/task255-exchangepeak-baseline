package exchange

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// TestScoreRejectsSealedBatchAndKeepsCandidates 守护不变量：批次封存后
// 再次评分必须被拒绝，且已发布的交换候选保持不变。
//
// 复现历史缺陷：score 先 DeleteCandidatesByBatch 再写入，未检查封存状态，
// 导致封存批次再评分会删掉旧候选并改写已发布分析结果。
func TestScoreRejectsSealedBatchAndKeepsCandidates(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()

	svc := New(st)

	// 建批次并推进至 published（为封存做合法流转铺垫）。
	const batchID = "batch-sealed"
	if err := seedSample(st, "sample-1", "S1"); err != nil {
		t.Fatalf("seedSample() error = %v", err)
	}
	if err := seedBatch(st, batchID, "sample-1", "B1"); err != nil {
		t.Fatalf("seedBatch() error = %v", err)
	}
	for _, to := range []string{
		model.BatchPendingLink,
		model.BatchNeedsReview,
		model.BatchPublished,
		model.BatchSealed,
	} {
		if err := st.SetBatchState(batchID, to); err != nil {
			t.Fatalf("SetBatchState(%s) error = %v", to, err)
		}
	}

	// 植入一个已发布分析阶段的既有候选，模拟封存前已冻结的结果。
	existing := &model.ExchangeCandidate{
		ID:        uuid.NewString(),
		BatchID:   batchID,
		TrackAID:  "track-a",
		TrackBID:  "track-b",
		Kind:      model.ExchangeMerge,
		Score:     0.5,
		State:     model.ExConfirmed,
		Reason:    "frozen-before-seal",
		CreatedAt: time.Now(),
	}
	if err := st.CreateCandidate(existing); err != nil {
		t.Fatalf("CreateCandidate() error = %v", err)
	}

	// 封存批次再次评分必须被拒绝。
	_, err = svc.Score(batchID)
	if !errors.Is(err, model.ErrSealedBatch) {
		t.Fatalf("Score(sealed) error = %v, want %v", err, model.ErrSealedBatch)
	}

	// 既有候选必须原样保留（未被清空、未被改写）。
	got, err := svc.List(batchID)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("after Score on sealed batch, candidates = %d, want 1 (existing preserved)", len(got))
	}
	if got[0].ID != existing.ID || got[0].State != existing.State || got[0].Score != existing.Score || got[0].Reason != existing.Reason {
		t.Fatalf("existing candidate mutated = %+v, want %+v", got[0], existing)
	}
}

func seedSample(st *store.Store, id, name string) error {
	return st.CreateSample(&model.Sample{ID: id, Name: name, CreatedAt: time.Now()})
}

func seedBatch(st *store.Store, id, sampleID, label string) error {
	return st.CreateBatch(&model.SpectrumBatch{
		ID:        id,
		SampleID:  sampleID,
		Label:     label,
		State:     model.BatchReceiving,
		CreatedAt: time.Now(),
	})
}
