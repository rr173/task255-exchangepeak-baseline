package model

import "testing"

func TestCanTransitionBatchAcceptsOnlyAdjacentStates(t *testing.T) {
	for _, tc := range []struct {
		from string
		to   string
		want bool
	}{
		{BatchReceiving, BatchPendingLink, true},
		{BatchPendingLink, BatchNeedsReview, true},
		{BatchNeedsReview, BatchPublished, true},
		{BatchPublished, BatchSealed, true},
		{BatchReceiving, BatchPublished, false},
		{BatchSealed, BatchReceiving, false},
		{"unknown", BatchReceiving, false},
	} {
		if got := CanTransitionBatch(tc.from, tc.to); got != tc.want {
			t.Fatalf("CanTransitionBatch(%q, %q) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestValidExchangeTrendRejectsReversal(t *testing.T) {
	if ValidExchangeTrend(ExchangeMerge, []float64{0.6, 0.1, 0.2}) {
		t.Fatal("merge trend with a reversal was accepted")
	}
	if !ValidExchangeTrend(ExchangeSplit, []float64{0.1, 0.2, 0.4}) {
		t.Fatal("monotonic split trend was rejected")
	}
}
