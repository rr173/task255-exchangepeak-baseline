package track

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/model"
)

// linkTolerancePPM 是相邻温度档之间峰可关联的最大化学位移差。
const linkTolerancePPM = 0.6

// associate 采用贪心最近邻策略，把同一共振在不同温度下的峰串联成轨迹。
//
// 做法：
//  1. 仅保留未被标记杂质/排除的峰；
//  2. 按温度升序分层，每层峰按校正后化学位移排序；
//  3. 最低温度档每条峰各自起一条轨迹；
//  4. 对更高温度档，为每条已有轨迹在其前一温度位置附近（≤ linkTolerancePPM）
//     寻找最近的未分配峰并关联；剩余未分配峰新建轨迹；
//  5. 若批次处于 pending_link，关联完成后流转到 needs_review。
//
// 该启发式不保证全局最优，但对化学交换场景下"随温度升高两峰靠拢/融合"
// 的位移连续性刻画是稳定且可复现的。
func (svc *Service) associate(batchID string) ([]model.PeakTrack, error) {
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

	peaks, err := svc.Store.ListPeaksByBatch(batchID)
	if err != nil {
		return nil, err
	}
	active := make([]model.Peak, 0, len(peaks))
	for _, p := range peaks {
		if !model.IsActivePeakState(p.State) {
			continue
		}
		if p.State != model.PeakCorrected {
			return nil, model.ErrInvalidInput
		}
		active = append(active, p)
	}

	byTemp := make(map[float64][]model.Peak)
	for _, p := range active {
		byTemp[p.TempC] = append(byTemp[p.TempC], p)
	}
	temps := make([]float64, 0, len(byTemp))
	for t := range byTemp {
		temps = append(temps, t)
	}
	if len(temps) == 0 {
		return nil, model.ErrInvalidInput
	}
	sort.Float64s(temps)
	for _, t := range temps {
		sort.Slice(byTemp[t], func(i, j int) bool {
			return byTemp[t][i].CorrectedShift < byTemp[t][j].CorrectedShift
		})
	}

	type trackBuild struct {
		track   model.PeakTrack
		last    *model.Peak
		members []model.PeakTrackMember
	}
	var builds []trackBuild
	assigned := make(map[string]bool)

	// 最低温度档：每条峰起一条轨迹
	for i := range byTemp[temps[0]] {
		p := byTemp[temps[0]][i]
		tb := trackBuild{
			track: model.PeakTrack{
				ID:        uuid.NewString(),
				BatchID:   batchID,
				Label:     fmt.Sprintf("T%d", i+1),
				CreatedAt: time.Now(),
			},
			last: &p,
		}
		tb.members = append(tb.members, model.PeakTrackMember{
			ID: uuid.NewString(), TrackID: tb.track.ID, PeakID: p.ID, TempC: p.TempC,
		})
		assigned[p.ID] = true
		builds = append(builds, tb)
	}

	// 逐温度档向上关联
	for ti := 1; ti < len(temps); ti++ {
		t := temps[ti]
		candidates := byTemp[t]
		for bi := range builds {
			prev := builds[bi].last
			bestIdx := -1
			bestDist := linkTolerancePPM
			for ci := range candidates {
				if assigned[candidates[ci].ID] {
					continue
				}
				d := math.Abs(candidates[ci].CorrectedShift - prev.CorrectedShift)
				if d <= bestDist {
					bestDist = d
					bestIdx = ci
				}
			}
			if bestIdx >= 0 {
				p := candidates[bestIdx]
				assigned[p.ID] = true
				builds[bi].members = append(builds[bi].members, model.PeakTrackMember{
					ID: uuid.NewString(), TrackID: builds[bi].track.ID, PeakID: p.ID, TempC: p.TempC,
				})
				builds[bi].last = &p
			}
		}
		// 剩余未分配峰：新建轨迹（从此温度档起）
		for ci := range candidates {
			if assigned[candidates[ci].ID] {
				continue
			}
			p := candidates[ci]
			tb := trackBuild{
				track: model.PeakTrack{
					ID:        uuid.NewString(),
					BatchID:   batchID,
					Label:     fmt.Sprintf("T%d", len(builds)+1),
					CreatedAt: time.Now(),
				},
				last: &p,
			}
			tb.members = append(tb.members, model.PeakTrackMember{
				ID: uuid.NewString(), TrackID: tb.track.ID, PeakID: p.ID, TempC: p.TempC,
			})
			assigned[p.ID] = true
			builds = append(builds, tb)
		}
	}

	if err := svc.Store.DeleteCandidatesByBatch(batchID); err != nil {
		return nil, err
	}
	if err := svc.Store.DeleteTracksByBatch(batchID); err != nil {
		return nil, err
	}
	for i := range builds {
		if err := svc.Store.CreateTrack(&builds[i].track); err != nil {
			return nil, err
		}
		for j := range builds[i].members {
			if err := svc.Store.CreateTrackMember(&builds[i].members[j]); err != nil {
				return nil, err
			}
		}
	}
	if b.State == model.BatchPendingLink {
		_ = svc.Store.SetBatchState(batchID, model.BatchNeedsReview)
	}

	out := make([]model.PeakTrack, len(builds))
	for i := range builds {
		out[i] = builds[i].track
	}
	return out, nil
}
