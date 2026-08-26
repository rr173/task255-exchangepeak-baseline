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

// TestClassifyRejectsReboundInMergeTrend 覆盖用户报告的场景：两峰位移差总体
// 下降但中间反弹。ds=[0.60,0.20,0.25,0.15] 线性斜率为负、高温档收敛至
// mergeClosePPM 以内，按旧逻辑仅看总趋势会被误判为融合——此处必须因非单调而拒绝。
func TestClassifyRejectsReboundInMergeTrend(t *testing.T) {
	a := []pt{{25, 1.0}, {50, 1.0}, {75, 1.0}, {100, 1.0}}
	b := []pt{{25, 1.6}, {50, 1.2}, {75, 0.75}, {100, 0.85}}
	if _, _, _, ok := classify(a, b); ok {
		t.Fatal("classify accepted a merge trend with an intermediate rebound")
	}
}

// TestClassifyRejectsReboundInSplitTrend 覆盖裂分方向的对称情形：
// ds=[0.15,0.50,0.45,0.60] 斜率为正、低温接近且高温分离达标，但中途回落，
// 非单调，不得判为裂分。
func TestClassifyRejectsReboundInSplitTrend(t *testing.T) {
	a := []pt{{25, 1.0}, {50, 1.0}, {75, 1.0}, {100, 1.0}}
	b := []pt{{25, 1.15}, {50, 1.5}, {75, 0.55}, {100, 1.6}}
	if _, _, _, ok := classify(a, b); ok {
		t.Fatal("classify accepted a split trend with an intermediate rebound")
	}
}
