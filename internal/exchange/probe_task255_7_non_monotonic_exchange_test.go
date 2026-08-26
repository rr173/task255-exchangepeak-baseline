package exchange

import (
	"testing"
)

func TestBug07NonMonotonicMergeIsNotCandidate(t *testing.T) {
	a := []pt{{25, 1.0}, {50, 1.4}, {75, 1.6}}
	b := []pt{{25, 1.6}, {50, 1.5}, {75, 1.8}}
	if kind, _, _, ok := classify(a, b); ok {
		t.Fatalf("non-monotonic trend classified as %q", kind)
	}
}
