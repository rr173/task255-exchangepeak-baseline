package model

import "time"

// SpectrumBatch 是一组在多个温度下采集的谱图批次。
// 状态：receiving → pending_link → needs_review → published → sealed。
type SpectrumBatch struct {
	ID        string    `json:"id"`
	SampleID  string    `json:"sample_id"`
	Label     string    `json:"label"`
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

// Peak 是某温度下的一条共振峰记录。
// ObservedShift 为仪器读出的原始化学位移，CorrectedShift 为经内标校正后的值。
// Unit 统一为 ppm；若以 Hz 录入必须显式转换，否则纠正阶段报错。
type Peak struct {
	ID            string  `json:"id"`
	BatchID       string  `json:"batch_id"`
	TemperatureID string  `json:"temperature_id"`
	TempC         float64 `json:"temp_c"`
	Unit          string  `json:"unit"`
	ObservedShift float64 `json:"observed_shift"`
	CorrectedShift float64 `json:"corrected_shift"`
	Intensity     float64 `json:"intensity"`
	WidthHz       float64 `json:"width_hz"`
	IsStandard    bool    `json:"is_standard"`
	State         string  `json:"state"`
	Note          string  `json:"note"`
}
