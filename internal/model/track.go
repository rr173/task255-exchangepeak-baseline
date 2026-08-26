package model

import "time"

// PeakTrack 是把同一共振在不同温度下的峰关联起来形成的轨迹。
// 成员按温度升序排列，体现化学位移随温度的演变。
type PeakTrack struct {
	ID        string    `json:"id"`
	BatchID   string    `json:"batch_id"`
	Label     string    `json:"label"`
	CreatedAt time.Time `json:"created_at"`
}

// PeakTrackMember 是轨迹在某温度档上关联到的具体峰。
type PeakTrackMember struct {
	ID     string  `json:"id"`
	TrackID string `json:"track_id"`
	PeakID string  `json:"peak_id"`
	TempC  float64 `json:"temp_c"`
}
