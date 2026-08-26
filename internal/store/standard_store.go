package store

import (
	"database/sql"
	"time"

	"task255-exchangepeak/internal/model"
)

// CreateStandard 写入内标定义。
func (s *Store) CreateStandard(std *model.InternalStandard) error {
	_, err := s.DB.Exec(
		`INSERT INTO internal_standards (id,batch_id,name,locked,created_at)
		 VALUES (?,?,?,?,?)`,
		std.ID, std.BatchID, std.Name, boolToInt(std.Locked), std.CreatedAt.Format(time.RFC3339Nano),
	)
	return err
}

// GetStandard 按 ID 读取内标。
func (s *Store) GetStandard(id string) (*model.InternalStandard, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,name,locked,created_at FROM internal_standards WHERE id=?`, id)
	var std model.InternalStandard
	var locked int
	var created string
	if err := row.Scan(&std.ID, &std.BatchID, &std.Name, &locked, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	std.Locked = locked != 0
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	std.CreatedAt = t
	return &std, nil
}

// ListStandardsByBatch 列出批次的内标。
func (s *Store) ListStandardsByBatch(batchID string) ([]model.InternalStandard, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,name,locked,created_at FROM internal_standards
		 WHERE batch_id=? ORDER BY created_at`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.InternalStandard{}
	for rows.Next() {
		var std model.InternalStandard
		var locked int
		var created string
		if err := rows.Scan(&std.ID, &std.BatchID, &std.Name, &locked, &created); err != nil {
			return nil, err
		}
		std.Locked = locked != 0
		t, _ := time.Parse(time.RFC3339Nano, created)
		std.CreatedAt = t
		out = append(out, std)
	}
	return out, rows.Err()
}

// AddStandardPoint 追加内标在某一温度的真值点。
func (s *Store) AddStandardPoint(pt *model.InternalStandardPoint) error {
	_, err := s.DB.Exec(
		`INSERT INTO internal_standard_points (id,standard_id,temp_c,true_shift)
		 VALUES (?,?,?,?)`,
		pt.ID, pt.StandardID, pt.TempC, pt.TrueShift,
	)
	return err
}

// ListStandardPoints 列出内标的全部真值点（按温度升序）。
func (s *Store) ListStandardPoints(standardID string) ([]model.InternalStandardPoint, error) {
	rows, err := s.DB.Query(
		`SELECT id,standard_id,temp_c,true_shift FROM internal_standard_points
		 WHERE standard_id=? ORDER BY temp_c`, standardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.InternalStandardPoint{}
	for rows.Next() {
		var pt model.InternalStandardPoint
		if err := rows.Scan(&pt.ID, &pt.StandardID, &pt.TempC, &pt.TrueShift); err != nil {
			return nil, err
		}
		out = append(out, pt)
	}
	return out, rows.Err()
}

// LockStandard 锁定内标（不可再改真值点与参比峰）。
func (s *Store) LockStandard(id string) error {
	_, err := s.DB.Exec(`UPDATE internal_standards SET locked=1 WHERE id=?`, id)
	return err
}
