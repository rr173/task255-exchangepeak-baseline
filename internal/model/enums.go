package model

// 谱图批次状态
const (
	BatchReceiving   = "receiving"
	BatchPendingLink = "pending_link"
	BatchNeedsReview = "needs_review"
	BatchPublished   = "published"
	BatchSealed      = "sealed"
)

// 批次状态合法流转
var batchTransitions = map[string][]string{
	BatchReceiving:   {BatchPendingLink},
	BatchPendingLink: {BatchNeedsReview, BatchReceiving},
	BatchNeedsReview: {BatchPublished, BatchPendingLink},
	BatchPublished:   {BatchSealed},
	BatchSealed:      {},
}

// 峰记录状态
const (
	PeakRaw       = "raw"
	PeakCorrected = "corrected"
	PeakImpurity  = "impurity"
	PeakDuplicate = "duplicate"
	PeakExcluded  = "excluded"
)

// 交换候选状态
const (
	ExGenerated     = "generated"
	ExContinuous    = "continuous"
	ExSplitConflict = "split_conflict"
	ExConfirmed     = "confirmed"
	ExRejected      = "rejected"
)

// 交换候选类型
const (
	ExchangeMerge = "merge" // 合并：随升温两峰靠近/融合
	ExchangeSplit = "split" // 分裂：单峰随温度变化裂为两峰
)

// 归属快照状态
const (
	SnapDraft      = "draft"
	SnapPublished  = "published"
	SnapSuperseded = "superseded"
)

// 化学位移单位
const (
	UnitPPM = "ppm"
	UnitHz  = "hz"
)

// CanTransitionBatch 校验批次状态流转是否合法。
func CanTransitionBatch(from, to string) bool {
	allowed, ok := batchTransitions[from]
	if !ok {
		return false
	}
	for _, a := range allowed {
		if a == to {
			return true
		}
	}
	return false
}

// IsActivePeakState reports whether a peak participates in analysis.
func IsActivePeakState(state string) bool {
	return state == PeakRaw || state == PeakCorrected
}
