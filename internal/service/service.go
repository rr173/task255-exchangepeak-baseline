package service

import (
	"time"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/exchange"
	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/snapshot"
	"task255-exchangepeak/internal/store"
	"task255-exchangepeak/internal/track"
)

// Service 是顶层编排服务，组合各业务包并暴露跨域工作流。
type Service struct {
	Store    *store.Store
	Sample   *sample.Service
	Peak     *peak.Service
	Track    *track.Service
	Exchange *exchange.Service
	Snapshot *snapshot.Service
}

// New 构造顶层编排服务。
func New(s *store.Store) *Service {
	return &Service{
		Store:    s,
		Sample:   sample.New(s),
		Peak:     peak.New(s),
		Track:    track.New(s),
		Exchange: exchange.New(s),
		Snapshot: snapshot.New(s),
	}
}

// CreateBatch 创建谱图批次（校验样品存在，初始状态 receiving）。
func (svc *Service) CreateBatch(sampleID, label string) (*model.SpectrumBatch, error) {
	if _, err := svc.Store.GetSample(sampleID); err != nil {
		return nil, err
	}
	if label == "" {
		return nil, model.ErrInvalidInput
	}
	b := &model.SpectrumBatch{
		ID:        uuid.NewString(),
		SampleID:  sampleID,
		Label:     label,
		State:     model.BatchReceiving,
		CreatedAt: time.Now(),
	}
	if err := svc.Store.CreateBatch(b); err != nil {
		return nil, err
	}
	return b, nil
}

// SetBatchState 推进批次状态（校验合法流转）。
func (svc *Service) SetBatchState(batchID, to string) error {
	return svc.Store.SetBatchState(batchID, to)
}

// CreateStandard 为批次创建内标定义。
func (svc *Service) CreateStandard(batchID, name string) (*model.InternalStandard, error) {
	if _, err := svc.Store.GetBatch(batchID); err != nil {
		return nil, err
	}
	if name == "" {
		return nil, model.ErrInvalidInput
	}
	std := &model.InternalStandard{
		ID:        uuid.NewString(),
		BatchID:   batchID,
		Name:      name,
		Locked:    false,
		CreatedAt: time.Now(),
	}
	if err := svc.Store.CreateStandard(std); err != nil {
		return nil, err
	}
	return std, nil
}

// AddStandardPoint 追加内标在某温度的真值点。
func (svc *Service) AddStandardPoint(standardID string, tempC, trueShift float64) error {
	std, err := svc.Store.GetStandard(standardID)
	if err != nil {
		return err
	}
	if std.Locked {
		return model.ErrStandardLocked
	}
	pt := &model.InternalStandardPoint{
		ID:         uuid.NewString(),
		StandardID: standardID,
		TempC:      tempC,
		TrueShift:  trueShift,
	}
	return svc.Store.AddStandardPoint(pt)
}

// LockStandard 锁定内标。
func (svc *Service) LockStandard(standardID string) error {
	return svc.Store.LockStandard(standardID)
}

// Analyze 端到端分析工作流：内标校正 → 轨迹关联 → 交换评分。
// 三者严格按序执行，任一步失败即整体返回错误，保证批次处于一致状态。
func (svc *Service) Analyze(batchID string) error {
	if _, err := svc.Peak.Calibrate(batchID); err != nil {
		return err
	}
	if _, err := svc.Track.Associate(batchID); err != nil {
		return err
	}
	if _, err := svc.Exchange.Score(batchID); err != nil {
		return err
	}
	return nil
}

// SelfCheck 返回各实体当前行数，用于运行自检。
func (svc *Service) SelfCheck() (store.Stats, error) {
	return svc.Store.Stats()
}
