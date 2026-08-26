package exchange

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/model"
)

const (
	// minCommonTemps 是判定趋势所需的最少共同温度档。
	minCommonTemps = 3
	// mergeClosePPM 是融合判定：高温档两峰位移差需收敛到此值以内。
	mergeClosePPM = 0.20
	// splitClosePPM 是裂分判定：低温档两峰需已接近到此值以内。
	splitClosePPM = 0.20
	// splitFarPPM 是裂分判定：高温档两峰需分离到此值以上。
	splitFarPPM = 0.40
	// trendSlopeEps 是线性趋势斜率的最小显著值（ppm/℃）。
	trendSlopeEps = 1e-4
)

// pt 是轨迹在某温度档上的校正位移。
type pt struct {
	tempC float64
	delta float64
}

// pair 是一条轨迹对在某共同温度档上的位移。
type pair struct {
	t, da, db float64
}

// trackSeries 是一条轨迹在各温度档上的校正位移序列。
type trackSeries struct {
	track model.PeakTrack
	pts   []pt
}

// score 重新计算批次内全部轨迹对的化学交换候选（先清空再写入）。
func (svc *Service) score(batchID string) ([]model.ExchangeCandidate, error) {
	b, err := svc.Store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.State == model.BatchSealed {
		return nil, model.ErrSealedBatch
	}
	tracks, err := svc.Store.ListTracksByBatch(batchID)
	if err != nil {
		return nil, err
	}
	series, err := svc.loadSeries(tracks)
	if err != nil {
		return nil, err
	}

	cands := make([]model.ExchangeCandidate, 0, len(series))
	for i := 0; i < len(series); i++ {
		for j := i + 1; j < len(series); j++ {
			kind, sc, reason, ok := classify(series[i].pts, series[j].pts)
			if !ok {
				continue
			}
			cands = append(cands, model.ExchangeCandidate{
				ID:        uuid.NewString(),
				BatchID:   batchID,
				TrackAID:  series[i].track.ID,
				TrackBID:  series[j].track.ID,
				Kind:      kind,
				Score:     sc,
				State:     model.ExGenerated,
				Reason:    reason,
				CreatedAt: time.Now(),
			})
		}
	}

	if err := svc.Store.DeleteCandidatesByBatch(batchID); err != nil {
		return nil, err
	}
	for k := range cands {
		if err := svc.Store.CreateCandidate(&cands[k]); err != nil {
			return nil, err
		}
	}
	return cands, nil
}

// loadSeries 把每条轨迹展开成 (温度, 校正位移) 序列，并排除含参比峰的轨迹。
func (svc *Service) loadSeries(tracks []model.PeakTrack) ([]trackSeries, error) {
	out := make([]trackSeries, 0, len(tracks))
	for _, t := range tracks {
		members, err := svc.Store.ListTrackMembers(t.ID)
		if err != nil {
			return nil, err
		}
		pts := make([]pt, 0, len(members))
		skip := false
		for _, m := range members {
			p, err := svc.Store.GetPeak(m.PeakID)
			if err != nil {
				return nil, err
			}
			if p.IsStandard {
				skip = true
				break
			}
			pts = append(pts, pt{tempC: m.TempC, delta: p.CorrectedShift})
		}
		if skip || len(pts) < minCommonTemps {
			continue
		}
		sort.Slice(pts, func(a, b int) bool { return pts[a].tempC < pts[b].tempC })
		out = append(out, trackSeries{track: t, pts: pts})
	}
	return out, nil
}

// classify 判断两条轨迹的位移差随温度的趋势，给出 merge/split 判定与评分。
func classify(a, b []pt) (string, float64, string, bool) {
	// 对齐共同温度档
	idxB := 0
	common := make([]pair, 0)
	for _, pa := range a {
		for idxB < len(b) && b[idxB].tempC < pa.tempC {
			idxB++
		}
		if idxB < len(b) && math.Abs(b[idxB].tempC-pa.tempC) < 1e-9 {
			common = append(common, pair{t: pa.tempC, da: pa.delta, db: b[idxB].delta})
			idxB++
		}
	}
	if len(common) < minCommonTemps {
		return "", 0, "", false
	}
	ds := make([]float64, len(common))
	for i := range common {
		ds[i] = math.Abs(common[i].da - common[i].db)
	}
	slope := linSlope(commonTemps(common), ds)
	lo, hi := ds[0], ds[len(ds)-1]

	switch {
	case slope < -trendSlopeEps: // 随升温差值缩小 → 融合
		if hi > mergeClosePPM || !model.ValidExchangeTrend(model.ExchangeMerge, ds) {
			return "", 0, "", false
		}
		sc := (lo - hi) / math.Max(lo, 0.05)
		if sc > 1 {
			sc = 1
		}
		reason := fmt.Sprintf("随升温两峰化学位移差由 %.3f 收敛至 %.3f ppm，呈快交换融合特征", lo, hi)
		return model.ExchangeMerge, sc, reason, true
	case slope > trendSlopeEps: // 随升温差值扩大 → 裂分
		if lo > splitClosePPM || hi < splitFarPPM || !model.ValidExchangeTrend(model.ExchangeSplit, ds) {
			return "", 0, "", false
		}
		sc := (hi - lo) / math.Max(hi, 0.05)
		if sc > 1 {
			sc = 1
		}
		reason := fmt.Sprintf("低温两峰接近(Δδ=%.3f)，升温后分离至 %.3f ppm，呈慢交换裂分特征", lo, hi)
		return model.ExchangeSplit, sc, reason, true
	}
	return "", 0, "", false
}


func commonTemps(c []pair) []float64 {
	out := make([]float64, len(c))
	for i := range c {
		out[i] = c[i].t
	}
	return out
}

// linSlope 计算 (x,y) 的最小二乘斜率。
func linSlope(xs, ys []float64) float64 {
	n := float64(len(xs))
	var sx, sy, sxx, sxy float64
	for i := range xs {
		x, y := xs[i], ys[i]
		sx += x
		sy += y
		sxx += x * x
		sxy += x * y
	}
	denom := n*sxx - sx*sx
	if math.Abs(denom) < 1e-12 {
		return 0
	}
	return (n*sxy - sx*sy) / denom
}
