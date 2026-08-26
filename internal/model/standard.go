package model

import "time"

// InternalStandard 是谱图批次使用的内标（参比物）。
// 内标锁定（Locked=true）后，其真值点与参比峰均不可再修改。
type InternalStandard struct {
	ID        string    `json:"id"`
	BatchID   string    `json:"batch_id"`
	Name      string    `json:"name"`
	Locked    bool      `json:"locked"`
	CreatedAt time.Time `json:"created_at"`
}

// InternalStandardPoint 给出内标在某一温度下的已知真值化学位移（ppm）。
// 校正偏移 offset(T) = 参比峰在 T 的观测值 − TrueShift(T)。
type InternalStandardPoint struct {
	ID         string  `json:"id"`
	StandardID string  `json:"standard_id"`
	TempC      float64 `json:"temp_c"`
	TrueShift  float64 `json:"true_shift"`
}
