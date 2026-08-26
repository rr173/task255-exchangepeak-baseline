package model

import "time"

// AssignmentSnapshot 冻结某一时刻的峰归属证据：样品条件、温度序列、
// 内标版本、峰轨迹与交换候选结果。冻结后不可变，可被新快照替代。
type AssignmentSnapshot struct {
	ID       string    `json:"id"`
	BatchID  string    `json:"batch_id"`
	State    string    `json:"state"`
	FrozenAt time.Time `json:"frozen_at"`
	Payload  string    `json:"payload"`
}
