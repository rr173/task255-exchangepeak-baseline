package peak

import (
	"math"

	"github.com/google/uuid"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

// Service 负责峰记录的写入、标注与内标校正。
type Service struct {
	Store *store.Store
}

// New 构造峰业务服务。
func New(s *store.Store) *Service { return &Service{Store: s} }

// AddInput 新增峰的入参。
type AddInput struct {
	BatchID       string  `json:"batch_id"`
	TemperatureID string  `json:"temperature_id"`
	TempC         float64 `json:"temp_c"`
	Unit          string  `json:"unit"`
	ObservedShift float64 `json:"observed_shift"`
	Intensity     float64 `json:"intensity"`
	WidthHz       float64 `json:"width_hz"`
	IsStandard    bool    `json:"is_standard"`
	Note          string  `json:"note"`
}

// Add 写入一条峰记录。单位必须为 ppm 或 hz；封存批次不可再添加。
func (svc *Service) Add(in AddInput) (*model.Peak, error) {
	if in.Unit != model.UnitPPM && in.Unit != model.UnitHz {
		return nil, model.ErrInvalidInput
	}
	if in.BatchID == "" || in.TemperatureID == "" {
		return nil, model.ErrInvalidInput
	}
	b, err := svc.Store.GetBatch(in.BatchID)
	if err != nil {
		return nil, err
	}
	if b.State == model.BatchSealed {
		return nil, model.ErrSealedBatch
	}
	tp, err := svc.Store.GetTemperaturePoint(in.TemperatureID)
	if err != nil {
		return nil, err
	}
	if tp.SampleID != b.SampleID || math.Abs(tp.TempC-in.TempC) > 1e-9 {
		return nil, model.ErrInvalidInput
	}
	p := &model.Peak{
		ID:             uuid.NewString(),
		BatchID:        in.BatchID,
		TemperatureID:  in.TemperatureID,
		TempC:          in.TempC,
		Unit:           in.Unit,
		ObservedShift:  in.ObservedShift,
		CorrectedShift: in.ObservedShift,
		Intensity:      in.Intensity,
		WidthHz:        in.WidthHz,
		IsStandard:     in.IsStandard,
		State:          model.PeakRaw,
		Note:           in.Note,
	}
	if err := svc.Store.CreatePeak(p); err != nil {
		return nil, err
	}
	return p, nil
}

// List 列出批次全部峰。
func (svc *Service) List(batchID string) ([]model.Peak, error) {
	return svc.Store.ListPeaksByBatch(batchID)
}

// ListAtTemp 列出批次在指定温度的峰。
func (svc *Service) ListAtTemp(batchID string, tempC float64) ([]model.Peak, error) {
	return svc.Store.ListPeaksAtTemp(batchID, tempC)
}

// MarkImpurity 将峰标记为杂质（不可恢复为有效峰）。
func (svc *Service) MarkImpurity(peakID string) error {
	return svc.markState(peakID, model.PeakImpurity)
}

// MarkExcluded 将峰排除出后续关联。
func (svc *Service) MarkExcluded(peakID string) error {
	return svc.markState(peakID, model.PeakExcluded)
}

func (svc *Service) markState(peakID, state string) error {
	p, err := svc.Store.GetPeak(peakID)
	if err != nil {
		return err
	}
	b, err := svc.Store.GetBatch(p.BatchID)
	if err != nil {
		return err
	}
	if b.State == model.BatchSealed {
		return model.ErrSealedBatch
	}
	return svc.Store.SetPeakState(peakID, state)
}
