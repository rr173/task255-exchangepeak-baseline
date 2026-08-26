package model

import "time"

// ExchangeCandidate 表示一对轨迹之间可能存在的化学交换关系。
// Kind 为 merge（随升温两峰融合）或 split（单峰裂为两峰）。
// 状态：generated → continuous / split_conflict → confirmed / rejected。
type ExchangeCandidate struct {
	ID        string    `json:"id"`
	BatchID   string    `json:"batch_id"`
	TrackAID  string    `json:"track_a_id"`
	TrackBID  string    `json:"track_b_id"`
	Kind      string    `json:"kind"`
	Score     float64   `json:"score"`
	State     string    `json:"state"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// ValidExchangeTrend checks the monotonic separation/convergence invariant
// used by the exchange classifier.
func ValidExchangeTrend(kind string, deltas []float64) bool {
	for i := 1; i < len(deltas); i++ {
		switch kind {
		case ExchangeMerge:
			if deltas[i] > deltas[i-1]+1e-12 {
				return false
			}
		case ExchangeSplit:
			if deltas[i] < deltas[i-1]-1e-12 {
				return false
			}
		default:
			return false
		}
	}
	return true
}
