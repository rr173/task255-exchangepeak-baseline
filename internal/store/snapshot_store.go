package store

import (
	"database/sql"
	"time"

	"task255-exchangepeak/internal/model"
)

// CreateSnapshot 写入归属快照（初始状态 draft）。
func (s *Store) CreateSnapshot(snap *model.AssignmentSnapshot) error {
	_, err := s.DB.Exec(
		`INSERT INTO assignment_snapshots (id,batch_id,state,frozen_at,payload)
		 VALUES (?,?,?,?,?)`,
		snap.ID, snap.BatchID, snap.State, snap.FrozenAt.Format(time.RFC3339Nano), snap.Payload,
	)
	return err
}

// GetSnapshot 按 ID 读取快照。
func (s *Store) GetSnapshot(id string) (*model.AssignmentSnapshot, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,state,frozen_at,payload FROM assignment_snapshots WHERE id=?`, id)
	return scanSnapshot(row)
}

// ListSnapshotsByBatch 列出批次全部快照（按冻结时间升序）。
func (s *Store) ListSnapshotsByBatch(batchID string) ([]model.AssignmentSnapshot, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,state,frozen_at,payload FROM assignment_snapshots
		 WHERE batch_id=? ORDER BY frozen_at`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.AssignmentSnapshot{}
	for rows.Next() {
		snap, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *snap)
	}
	return out, rows.Err()
}

// SupersedeSnapshot 把旧快照置为 superseded，新快照置为 published。
func (s *Store) SupersedeSnapshot(oldID, newID string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE assignment_snapshots SET state=? WHERE id=?`,
		model.SnapSuperseded, oldID); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`UPDATE assignment_snapshots SET state=? WHERE id=?`,
		model.SnapPublished, newID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func scanSnapshot(sc scanner) (*model.AssignmentSnapshot, error) {
	var snap model.AssignmentSnapshot
	var frozen string
	if err := sc.Scan(&snap.ID, &snap.BatchID, &snap.State, &frozen, &snap.Payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, frozen)
	if err != nil {
		return nil, err
	}
	snap.FrozenAt = t
	return &snap, nil
}
