package exchange_test

import (
	"errors"
	"testing"
	"time"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
	"task255-exchangepeak/internal/store"
)

func TestBug06SealedBatchRejectsRescore(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	smp, _ := svc.Sample.Create(sample.CreateInput{Name: "s", Compound: "c", Solvent: "d", Concentration: 1})
	batch, _ := svc.CreateBatch(smp.ID, "batch")
	cand := &model.ExchangeCandidate{ID: "candidate-1", BatchID: batch.ID, TrackAID: "a", TrackBID: "b", Kind: model.ExchangeMerge, Score: .8, State: model.ExGenerated, CreatedAt: time.Unix(1, 0)}
	if err := st.CreateCandidate(cand); err != nil {
		t.Fatal(err)
	}
	for _, state := range []string{model.BatchPendingLink, model.BatchNeedsReview, model.BatchPublished, model.BatchSealed} {
		if err := svc.SetBatchState(batch.ID, state); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := svc.Exchange.Score(batch.ID); !errors.Is(err, model.ErrSealedBatch) {
		t.Fatalf("Score() error = %v, want sealed batch", err)
	}
	got, err := st.GetCandidate(cand.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != model.ExGenerated {
		t.Fatalf("candidate state after rejected rescore = %q", got.State)
	}
}
