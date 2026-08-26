package peak

import (
	"task255-exchangepeak/internal/model"
)

// Calibrate 以内标为基准，校正批次所有峰的化学位移。
//
// 对每个温度档 T：
//  1. 被排除/重复/杂质峰保留审计记录但不参与校正——其单位不做校验、不参与参比峰
//     选择、不写入校正值；
//  2. 该档有效峰（raw/corrected）化学位移单位必须为 ppm，否则返回 ErrUnitMismatch；
//  3. 必须存在一条有效参比峰（is_standard 且状态有效），否则返回 ErrNoStandardPeak；
//  4. 内标真值点必须覆盖 T，否则返回 ErrMissingStandard；
//  5. 偏移 offset(T) = 参比峰观测值 − 真值(T)；
//  6. 该档有效峰校正为 corrected = observed − offset(T)。
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

	// 按温度分档；每档进一步区分参与校正的有效峰与仅保留审计记录的被排除/杂质峰。
	type peakGroup struct {
		active []model.Peak // raw/corrected：参与校正
		audit  []model.Peak // excluded/duplicate/impurity：保留记录但不参与校正
	}
	byTemp := make(map[float64]peakGroup)
	for _, p := range peaks {
		g := byTemp[p.TempC]
		if model.IsActivePeakState(p.State) {
			g.active = append(g.active, p)
		} else {
			g.audit = append(g.audit, p)
		}
		byTemp[p.TempC] = g
	}

	offsets := make(map[float64]float64, len(byTemp))
	for tempC, g := range byTemp {
		// 仅校验有效峰的单位；被排除/重复/杂质峰即使单位不兼容也不影响校正。
		for _, p := range g.active {
			if p.Unit != model.UnitPPM {
				return nil, model.ErrUnitMismatch
			}
		}
		// 参比峰只能从有效峰中选择；被排除/杂质的参比峰不参与校正。
		var stdPeak *model.Peak
		for i := range g.active {
			if g.active[i].IsStandard {
				stdPeak = &g.active[i]
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
		for _, p := range byTemp[tempC].active {
			corrected := p.ObservedShift - offset
			if err := svc.Store.SetPeakCorrected(p.ID, corrected); err != nil {
				return nil, err
			}
		}
	}
	return offsets, nil
}
