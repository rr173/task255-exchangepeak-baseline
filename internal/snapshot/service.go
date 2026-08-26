package snapshot

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// Service 负责把批次当前的峰归属证据冻结成不可变快照。
type Service struct {
	Store *store.Store
}

// New 构造归属快照业务服务。
func New(s *store.Store) *Service { return &Service{Store: s} }

// Freeze 采集批次当前的轨迹与交换候选证据并冻结为快照。
// 若批次已存在 published 快照，则将其置为 superseded，新快照成为 published。
func (svc *Service) Freeze(batchID string) (*model.AssignmentSnapshot, error) {
	b, err := svc.Store.GetBatch(batchID)
	if err != nil {
		return nil, err
	}
	if b.State == model.BatchSealed {
		return nil, model.ErrSealedBatch
	}
	payload, err := svc.collect(batchID)
	if err != nil {
		return nil, err
	}
	snap := &model.AssignmentSnapshot{
		ID:       uuid.NewString(),
		BatchID:  batchID,
		State:    model.SnapPublished,
		FrozenAt: time.Now(),
		Payload:  payload,
	}
	if err := svc.Store.CreateSnapshot(snap); err != nil {
		return nil, err
	}
	prev, err := svc.Store.ListSnapshotsByBatch(batchID)
	if err != nil {
		return nil, err
	}
	for _, p := range prev {
		if p.ID != snap.ID && p.State == model.SnapPublished {
			if err := svc.Store.SupersedeSnapshot(p.ID, snap.ID); err != nil {
				return nil, err
			}
		}
	}
	return snap, nil
}

// Get 读取快照。
func (svc *Service) Get(id string) (*model.AssignmentSnapshot, error) {
	return svc.Store.GetSnapshot(id)
}

// List 列出批次全部快照。
func (svc *Service) List(batchID string) ([]model.AssignmentSnapshot, error) {
	return svc.Store.ListSnapshotsByBatch(batchID)
}

// collect 把批次当前的可溯源证据序列化为 JSON 负载。
func (svc *Service) collect(batchID string) (string, error) {
	b, err := svc.Store.GetBatch(batchID)
	if err != nil {
		return "", err
	}
	tracks, err := svc.Store.ListTracksByBatch(batchID)
	if err != nil {
		return "", err
	}
	cands, err := svc.Store.ListCandidatesByBatch(batchID)
	if err != nil {
		return "", err
	}
	smp, err := svc.Store.GetSample(b.SampleID)
	if err != nil {
		return "", err
	}
	temps, err := svc.Store.ListTemperaturePoints(b.SampleID)
	if err != nil {
		return "", err
	}
	standards, err := svc.Store.ListStandardsByBatch(batchID)
	if err != nil {
		return "", err
	}
	standardEvidence := make([]map[string]interface{}, 0, len(standards))
	for _, std := range standards {
		points, err := svc.Store.ListStandardPoints(std.ID)
		if err != nil {
			return "", err
		}
		standardEvidence = append(standardEvidence, map[string]interface{}{
			"id": std.ID, "name": std.Name, "locked": std.Locked, "points": points,
		})
	}
	trackEvidence := make([]map[string]interface{}, 0, len(tracks))
	for _, tr := range tracks {
		members, err := svc.Store.ListTrackMembers(tr.ID)
		if err != nil {
			return "", err
		}
		trackEvidence = append(trackEvidence, map[string]interface{}{
			"id": tr.ID, "label": tr.Label, "members": members,
		})
	}
	confirmed := 0
	for _, c := range cands {
		if c.State == model.ExConfirmed {
			confirmed++
		}
	}
	view := map[string]interface{}{
		"batch_id":        batchID,
		"batch_label":     b.Label,
		"sample":          smp,
		"temperatures":    temps,
		"standards":       standardEvidence,
		"tracks":          trackEvidence,
		"track_count":     len(tracks),
		"candidate_count": len(cands),
		"confirmed_count": confirmed,
		"frozen_at":       time.Now().Format(time.RFC3339Nano),
		"candidates":      cands,
	}
	data, err := json.Marshal(view)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
