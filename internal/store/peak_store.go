package store

import (
	"database/sql"

	"task255-exchangepeak/internal/model"
)

// CreatePeak 写入一条峰记录。
func (s *Store) CreatePeak(p *model.Peak) error {
	_, err := s.DB.Exec(
		`INSERT INTO peaks
		 (id,batch_id,temperature_id,temp_c,unit,observed_shift,corrected_shift,intensity,width_hz,is_standard,state,note)
		 VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.ID, p.BatchID, p.TemperatureID, p.TempC, p.Unit, p.ObservedShift,
		p.CorrectedShift, p.Intensity, p.WidthHz, boolToInt(p.IsStandard), p.State, p.Note,
	)
	return err
}

// GetPeak 按 ID 读取峰。
func (s *Store) GetPeak(id string) (*model.Peak, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,temperature_id,temp_c,unit,observed_shift,corrected_shift,
		        intensity,width_hz,is_standard,state,note FROM peaks WHERE id=?`, id)
	return scanPeak(row)
}

// ListPeaksByBatch 列出批次全部峰（按温度升序、化学位移升序）。
func (s *Store) ListPeaksByBatch(batchID string) ([]model.Peak, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,temperature_id,temp_c,unit,observed_shift,corrected_shift,
		        intensity,width_hz,is_standard,state,note
		 FROM peaks WHERE batch_id=? ORDER BY temp_c, observed_shift`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Peak{}
	for rows.Next() {
		p, err := scanPeak(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ListActivePeaksByBatch excludes impurity, duplicate, and excluded peaks.
func (s *Store) ListActivePeaksByBatch(batchID string) ([]model.Peak, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,temperature_id,temp_c,unit,observed_shift,corrected_shift,
		        intensity,width_hz,is_standard,state,note
		 FROM peaks WHERE batch_id=? AND state IN (?,?) ORDER BY temp_c, observed_shift`,
		batchID, model.PeakRaw, model.PeakCorrected)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Peak{}
	for rows.Next() {
		p, err := scanPeak(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// ListPeaksAtTemp 列出批次在指定温度下的峰。
func (s *Store) ListPeaksAtTemp(batchID string, tempC float64) ([]model.Peak, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,temperature_id,temp_c,unit,observed_shift,corrected_shift,
		        intensity,width_hz,is_standard,state,note
		 FROM peaks WHERE batch_id=? AND temp_c=? ORDER BY observed_shift`, batchID, tempC)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.Peak{}
	for rows.Next() {
		p, err := scanPeak(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// StandardPeakAtTemp 返回批次在指定温度下的参比峰（is_standard=1）。
func (s *Store) StandardPeakAtTemp(batchID string, tempC float64) (*model.Peak, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,temperature_id,temp_c,unit,observed_shift,corrected_shift,
		        intensity,width_hz,is_standard,state,note
		 FROM peaks WHERE batch_id=? AND temp_c=? AND is_standard=1
		 ORDER BY observed_shift LIMIT 1`, batchID, tempC)
	return scanPeak(row)
}

// SetPeakCorrected 写入校正后的化学位移。
func (s *Store) SetPeakCorrected(id string, corrected float64) error {
	_, err := s.DB.Exec(`UPDATE peaks SET corrected_shift=?, state=? WHERE id=?`,
		corrected, model.PeakCorrected, id)
	return err
}

// SetPeakState 更新峰状态（杂质/排除等）。
func (s *Store) SetPeakState(id, state string) error {
	_, err := s.DB.Exec(`UPDATE peaks SET state=? WHERE id=?`, state, id)
	return err
}

func scanPeak(sc scanner) (*model.Peak, error) {
	var p model.Peak
	var isStd int
	if err := sc.Scan(&p.ID, &p.BatchID, &p.TemperatureID, &p.TempC, &p.Unit,
		&p.ObservedShift, &p.CorrectedShift, &p.Intensity, &p.WidthHz, &isStd, &p.State, &p.Note); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	p.IsStandard = isStd != 0
	return &p, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
