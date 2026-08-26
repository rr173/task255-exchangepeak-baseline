package exchange

import (
	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// Service 负责化学交换候选的评分与裁决。
type Service struct {
	Store *store.Store
}

// New 构造交换候选业务服务。
func New(s *store.Store) *Service { return &Service{Store: s} }

// Score 重新评分批次的化学交换候选（先清空再生成）。
func (svc *Service) Score(batchID string) ([]model.ExchangeCandidate, error) {
	return svc.score(batchID)
}

// List 列出批次全部交换候选。
func (svc *Service) List(batchID string) ([]model.ExchangeCandidate, error) {
	return svc.Store.ListCandidatesByBatch(batchID)
}

// Get 读取候选。
func (svc *Service) Get(id string) (*model.ExchangeCandidate, error) {
	return svc.Store.GetCandidate(id)
}

// Confirm 确认候选（研究者的裁决）。
func (svc *Service) Confirm(id string) error {
	return svc.Store.SetCandidateState(id, model.ExConfirmed)
}

// Reject 否决候选。
func (svc *Service) Reject(id string) error {
	return svc.Store.SetCandidateState(id, model.ExRejected)
}
