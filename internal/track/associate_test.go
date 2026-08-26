package track

import "testing"

func TestLinkToleranceIsInclusiveAtBoundary(t *testing.T) {
	if linkTolerancePPM != 0.6 {
		t.Fatalf("linkTolerancePPM = %v, want 0.6", linkTolerancePPM)
	}
}
