package sample

import (
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// Service 负责样品与温度序列的创建与维护。
type Service struct {
	Store *store.Store
}

// New 构造样品业务服务。
func New(s *store.Store) *Service { return &Service{Store: s} }

// CreateInput 创建样品的入参。
type CreateInput struct {
	Name          string
	Compound      string
	Solvent       string
	Concentration float64
}

// Create 校验并写入样品。
func (svc *Service) Create(in CreateInput) (*model.Sample, error) {
	if in.Name == "" || in.Compound == "" {
		return nil, model.ErrInvalidInput
	}
	if in.Concentration <= 0 {
		return nil, model.ErrInvalidInput
	}
	smp := &model.Sample{
		ID:            uuid.NewString(),
		Name:          in.Name,
		Compound:      in.Compound,
		Solvent:       in.Solvent,
		Concentration: in.Concentration,
		CreatedAt:     time.Now(),
	}
	if err := svc.Store.CreateSample(smp); err != nil {
		return nil, err
	}
	return smp, nil
}

// AddTemperatures 向样品温度序列追加若干档温度，保证序列内温度值唯一。
// 整体原子：只要请求内或与已有序列存在重复温度，本次追加整体失败，
// 不留下任何部分新增的温度；否则整批按原顺序一次性写入。
func (svc *Service) AddTemperatures(sampleID string, temps []float64) ([]model.TemperaturePoint, error) {
	if _, err := svc.Store.GetSample(sampleID); err != nil {
		return nil, err
	}
	existing, err := svc.Store.ListTemperaturePoints(sampleID)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, e := range existing {
		seen[tempKey(e.TempC)] = true
	}
	out := make([]model.TemperaturePoint, 0, len(temps))
	points := make([]model.TemperaturePoint, 0, len(temps))
	order := len(existing)
	for _, t := range temps {
		k := tempKey(t)
		if seen[k] {
			return nil, model.ErrTempConflict
		}
		seen[k] = true
		tp := model.TemperaturePoint{
			ID:        uuid.NewString(),
			SampleID:  sampleID,
			TempC:     t,
			SortOrder: order,
		}
		points = append(points, tp)
		out = append(out, tp)
		order++
	}
	// 全部校验通过后再整批原子写入，避免重复请求留下半套温度。
	if err := svc.Store.AddTemperaturePoints(points); err != nil {
		return nil, err
	}
	return out, nil
}

// List 列出全部样品。
func (svc *Service) List() ([]model.Sample, error) {
	return svc.Store.ListSamples()
}

// Get 读取样品。
func (svc *Service) Get(id string) (*model.Sample, error) {
	return svc.Store.GetSample(id)
}

// Temperatures 读取样品温度序列。
func (svc *Service) Temperatures(sampleID string) ([]model.TemperaturePoint, error) {
	return svc.Store.ListTemperaturePoints(sampleID)
}

// Delete 删除样品及其温度序列。
func (svc *Service) Delete(id string) error {
	return svc.Store.DeleteSample(id)
}

func tempKey(t float64) string {
	return fmt.Sprintf("%.3f", math.Round(t*1000)/1000)
}
