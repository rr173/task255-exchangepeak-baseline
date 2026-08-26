package exchange

import (
	"testing"

	"task255-exchangepeak/internal/model"
)

func TestClassifyDetectsMergeTrend(t *testing.T) {
	a := []pt{{25, 1.0}, {50, 1.4}, {75, 1.6}}
	b := []pt{{25, 1.6}, {50, 1.8}, {75, 1.75}}
	kind, score, _, ok := classify(a, b)
	if !ok || kind != model.ExchangeMerge || score <= 0 {
		t.Fatalf("classify merge = kind %q score %v ok %v", kind, score, ok)
	}
}

func TestClassifyRequiresCommonTemperatureCoverage(t *testing.T) {
	a := []pt{{25, 1.0}, {50, 1.2}}
	b := []pt{{25, 1.5}, {50, 1.6}}
	if _, _, _, ok := classify(a, b); ok {
		t.Fatal("classify accepted fewer than the minimum common temperatures")
	}
}
