package store

import (
	"database/sql"
	"time"

	"task255-exchangepeak/internal/model"
)

// CreateBatch 写入谱图批次（初始状态 receiving）。
func (s *Store) CreateBatch(b *model.SpectrumBatch) error {
	_, err := s.DB.Exec(
		`INSERT INTO spectrum_batches (id,sample_id,label,state,created_at)
		 VALUES (?,?,?,?,?)`,
		b.ID, b.SampleID, b.Label, b.State, b.CreatedAt.Format(time.RFC3339Nano),
	)
	return err
}

// GetBatch 按 ID 读取批次。
func (s *Store) GetBatch(id string) (*model.SpectrumBatch, error) {
	row := s.DB.QueryRow(
		`SELECT id,sample_id,label,state,created_at FROM spectrum_batches WHERE id=?`, id)
	return scanBatch(row)
}

// ListBatches 列出批次，可按 sampleID 过滤（空串表示全部）。
func (s *Store) ListBatches(sampleID string) ([]model.SpectrumBatch, error) {
	var rows *sql.Rows
	var err error
	if sampleID == "" {
		rows, err = s.DB.Query(
			`SELECT id,sample_id,label,state,created_at FROM spectrum_batches ORDER BY created_at`)
	} else {
		rows, err = s.DB.Query(
			`SELECT id,sample_id,label,state,created_at FROM spectrum_batches
			 WHERE sample_id=? ORDER BY created_at`, sampleID)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.SpectrumBatch{}
	for rows.Next() {
		b, err := scanBatch(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *b)
	}
	return out, rows.Err()
}

// SetBatchState 校验合法流转后更新批次状态。
func (s *Store) SetBatchState(id, to string) error {
	b, err := s.GetBatch(id)
	if err != nil {
		return err
	}
	if !model.CanTransitionBatch(b.State, to) {
		return model.ErrInvalidState
	}
	_, err = s.DB.Exec(`UPDATE spectrum_batches SET state=? WHERE id=?`, to, id)
	return err
}

func scanBatch(sc scanner) (*model.SpectrumBatch, error) {
	var b model.SpectrumBatch
	var created string
	if err := sc.Scan(&b.ID, &b.SampleID, &b.Label, &b.State, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	b.CreatedAt = t
	return &b, nil
}
