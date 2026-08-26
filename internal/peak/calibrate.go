package peak

import (
	"task255-exchangepeak/internal/model"
)

// Calibrate 以内标为基准，校正批次所有峰的化学位移。
//
// 对每个温度档 T：
//  1. 该档全部峰化学位移单位必须为 ppm，否则返回 ErrUnitMismatch；
//  2. 必须存在一条 is_standard 参比峰，否则返回 ErrNoStandardPeak；
//  3. 内标真值点必须覆盖 T，否则返回 ErrMissingStandard；
//  4. 偏移 offset(T) = 参比峰观测值 − 真值(T)；
//  5. 该档非参比峰校正为 corrected = observed − offset(T)。
//
// 返回每个温度档的偏移量，供轨迹关联复用。封存批次禁止校正。
func (svc *Service) Calibrate(batchID string) (map[float64]float64, error) {
	b, err := svc.Store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.State == model.BatchSealed {
		return nil, model.ErrSealedBatch
	}

	standards, err := svc.Store.ListStandardsByBatch(batchID)
	if err != nil {
		return nil, err
	}
	if len(standards) == 0 {
		return nil, model.ErrMissingStandard
	}
	std := standards[0]

	points, err := svc.Store.ListStandardPoints(std.ID)
	if err != nil {
		return nil, err
	}
	trueByTemp := make(map[float64]float64, len(points))
	for _, p := range points {
		trueByTemp[p.TempC] = p.TrueShift
	}

	peaks, err := svc.Store.ListPeaksByBatch(batchID)
	if err != nil {
		return nil, err
	}

	byTemp := make(map[float64][]model.Peak)
	for _, p := range peaks {
		byTemp[p.TempC] = append(byTemp[p.TempC], p)
	}

	offsets := make(map[float64]float64, len(byTemp))
	for tempC, group := range byTemp {
		for _, p := range group {
			if p.Unit != model.UnitPPM {
				return nil, model.ErrUnitMismatch
			}
		}
		var stdPeak *model.Peak
		for i := range group {
			if group[i].IsStandard {
				stdPeak = &group[i]
				break
			}
		}
		if stdPeak == nil {
			return nil, model.ErrNoStandardPeak
		}
		trueShift, ok := trueByTemp[tempC]
		if !ok {
			return nil, model.ErrMissingStandard
		}
		offsets[tempC] = stdPeak.ObservedShift - trueShift
	}

	for tempC, offset := range offsets {
		for _, p := range byTemp[tempC] {
			corrected := p.ObservedShift - offset
			if err := svc.Store.SetPeakCorrected(p.ID, corrected); err != nil {
				return nil, err
			}
		}
	}
	return offsets, nil
}
