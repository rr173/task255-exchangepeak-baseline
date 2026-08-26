package track

import (
	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// Service 负责峰轨迹的关联与查询。
type Service struct {
	Store *store.Store
}

// New 构造轨迹业务服务。
func New(s *store.Store) *Service { return &Service{Store: s} }

// Associate 对批次做峰随温度轨迹关联（算法见 associate.go）。
func (svc *Service) Associate(batchID string) ([]model.PeakTrack, error) {
	return svc.associate(batchID)
}

// List 列出批次全部轨迹。
func (svc *Service) List(batchID string) ([]model.PeakTrack, error) {
	return svc.Store.ListTracksByBatch(batchID)
}

// Get 读取轨迹。
func (svc *Service) Get(id string) (*model.PeakTrack, error) {
	return svc.Store.GetTrack(id)
}

// Members 读取轨迹成员（按温度升序）。
func (svc *Service) Members(trackID string) ([]model.PeakTrackMember, error) {
	return svc.Store.ListTrackMembers(trackID)
}
