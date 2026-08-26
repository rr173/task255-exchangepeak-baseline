package exchange_test

import (
	"errors"
	"testing"
	"time"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
	"task255-exchangepeak/internal/exchange"
)

func TestBug08TerminalCandidateCannotBeRejudged(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := exchange.New(st)
	cand := &model.ExchangeCandidate{ID: "candidate-1", BatchID: "batch", TrackAID: "a", TrackBID: "b", Kind: model.ExchangeMerge, Score: .8, State: model.ExGenerated, CreatedAt: time.Unix(1, 0)}
	if err := st.CreateCandidate(cand); err != nil {
		t.Fatal(err)
	}
	if err := svc.Reject(cand.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Confirm(cand.ID); !errors.Is(err, model.ErrInvalidState) {
		t.Fatalf("Confirm() after reject = %v, want invalid state", err)
	}
	got, err := st.GetCandidate(cand.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != model.ExRejected {
		t.Fatalf("terminal candidate state = %q", got.State)
	}
}
