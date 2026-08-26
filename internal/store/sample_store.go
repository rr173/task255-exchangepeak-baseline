package store

import (
	"database/sql"
	"time"

	"task255-exchangepeak/internal/model"
)

// CreateSample 写入样品。
func (s *Store) CreateSample(smp *model.Sample) error {
	_, err := s.DB.Exec(
		`INSERT INTO samples (id,name,compound,solvent,concentration,created_at)
		 VALUES (?,?,?,?,?,?)`,
		smp.ID, smp.Name, smp.Compound, smp.Solvent, smp.Concentration,
		smp.CreatedAt.Format(time.RFC3339Nano),
	)
	return err
}

// GetSample 按 ID 读取样品。
func (s *Store) GetSample(id string) (*model.Sample, error) {
	row := s.DB.QueryRow(
		`SELECT id,name,compound,solvent,concentration,created_at FROM samples WHERE id=?`, id)
	return scanSample(row)
}

// ListSamples 列出全部样品。
func (s *Store) ListSamples() ([]model.Sample, error) {
	rows, err := s.DB.Query(
		`SELECT id,name,compound,solvent,concentration,created_at FROM samples ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Sample{}
	for rows.Next() {
		smp, err := scanSample(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *smp)
	}
	return out, rows.Err()
}

// DeleteSample 删除样品及其温度序列。
func (s *Store) DeleteSample(id string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM temperature_points WHERE sample_id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM samples WHERE id=?`, id); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// AddTemperaturePoint 向样品温度序列追加一档温度。
func (s *Store) AddTemperaturePoint(tp *model.TemperaturePoint) error {
	_, err := s.DB.Exec(
		`INSERT INTO temperature_points (id,sample_id,temp_c,sort_order)
		 VALUES (?,?,?,?)`,
		tp.ID, tp.SampleID, tp.TempC, tp.SortOrder,
	)
	return err
}

// AddTemperaturePoints appends a temperature batch atomically.
func (s *Store) AddTemperaturePoints(points []model.TemperaturePoint) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	for i := range points {
		p := &points[i]
		if _, err := tx.Exec(
			`INSERT INTO temperature_points (id,sample_id,temp_c,sort_order)
			 VALUES (?,?,?,?)`, p.ID, p.SampleID, p.TempC, p.SortOrder,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// GetTemperaturePoint reads a declared temperature point by ID.
func (s *Store) GetTemperaturePoint(id string) (*model.TemperaturePoint, error) {
	row := s.DB.QueryRow(
		`SELECT id,sample_id,temp_c,sort_order FROM temperature_points WHERE id=?`, id)
	var tp model.TemperaturePoint
	if err := row.Scan(&tp.ID, &tp.SampleID, &tp.TempC, &tp.SortOrder); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	return &tp, nil
}

// ListTemperaturePoints 列出样品温度序列（按 sort_order 升序）。
func (s *Store) ListTemperaturePoints(sampleID string) ([]model.TemperaturePoint, error) {
	rows, err := s.DB.Query(
		`SELECT id,sample_id,temp_c,sort_order FROM temperature_points
		 WHERE sample_id=? ORDER BY sort_order`, sampleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.TemperaturePoint{}
	for rows.Next() {
		var tp model.TemperaturePoint
		if err := rows.Scan(&tp.ID, &tp.SampleID, &tp.TempC, &tp.SortOrder); err != nil {
			return nil, err
		}
		out = append(out, tp)
	}
	return out, rows.Err()
}

func scanSample(sc scanner) (*model.Sample, error) {
	var smp model.Sample
	var created string
	if err := sc.Scan(&smp.ID, &smp.Name, &smp.Compound, &smp.Solvent, &smp.Concentration, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	smp.CreatedAt = t
	return &smp, nil
}
