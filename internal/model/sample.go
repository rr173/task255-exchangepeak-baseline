package model

import "time"

// Sample 描述一次 NMR 实验的样品与溶剂条件。
type Sample struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Compound      string    `json:"compound"`
	Solvent       string    `json:"solvent"`
	Concentration float64   `json:"concentration"`
	CreatedAt     time.Time `json:"created_at"`
}

// TemperaturePoint 是样品温度序列中的一档温度（摄氏度）。
// 同一样品的温度序列内 TempC 必须唯一。
type TemperaturePoint struct {
	ID        string  `json:"id"`
	SampleID  string  `json:"sample_id"`
	TempC     float64 `json:"temp_c"`
	SortOrder int     `json:"sort_order"`
}
