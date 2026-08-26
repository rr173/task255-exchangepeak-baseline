package store

import (
	"database/sql"
	"time"

	"task255-exchangepeak/internal/model"
)

// CreateTrack 写入峰轨迹。
func (s *Store) CreateTrack(t *model.PeakTrack) error {
	_, err := s.DB.Exec(
		`INSERT INTO peak_tracks (id,batch_id,label,created_at)
		 VALUES (?,?,?,?)`,
		t.ID, t.BatchID, t.Label, t.CreatedAt.Format(time.RFC3339Nano),
	)
	return err
}

// GetTrack 按 ID 读取轨迹。
func (s *Store) GetTrack(id string) (*model.PeakTrack, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,label,created_at FROM peak_tracks WHERE id=?`, id)
	var t model.PeakTrack
	var created string
	if err := row.Scan(&t.ID, &t.BatchID, &t.Label, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	tt, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	t.CreatedAt = tt
	return &t, nil
}

// ListTracksByBatch 列出批次全部轨迹。
func (s *Store) ListTracksByBatch(batchID string) ([]model.PeakTrack, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,label,created_at FROM peak_tracks
		 WHERE batch_id=? ORDER BY created_at`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PeakTrack{}
	for rows.Next() {
		var t model.PeakTrack
		var created string
		if err := rows.Scan(&t.ID, &t.BatchID, &t.Label, &created); err != nil {
			return nil, err
		}
		tt, _ := time.Parse(time.RFC3339Nano, created)
		t.CreatedAt = tt
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateTrackMember 向轨迹追加一个成员峰。
func (s *Store) CreateTrackMember(m *model.PeakTrackMember) error {
	_, err := s.DB.Exec(
		`INSERT INTO peak_track_members (id,track_id,peak_id,temp_c)
		 VALUES (?,?,?,?)`,
		m.ID, m.TrackID, m.PeakID, m.TempC,
	)
	return err
}

// DeleteTracksByBatch removes a prior association result before recompute.
func (s *Store) DeleteTracksByBatch(batchID string) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(
		`DELETE FROM peak_track_members WHERE track_id IN
		 (SELECT id FROM peak_tracks WHERE batch_id=?)`, batchID,
	); err != nil {
		_ = tx.Rollback()
		return err
	}
	if _, err := tx.Exec(`DELETE FROM peak_tracks WHERE batch_id=?`, batchID); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// ListTrackMembers 列出轨迹成员（按温度升序）。
func (s *Store) ListTrackMembers(trackID string) ([]model.PeakTrackMember, error) {
	rows, err := s.DB.Query(
		`SELECT id,track_id,peak_id,temp_c FROM peak_track_members
		 WHERE track_id=? ORDER BY temp_c`, trackID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.PeakTrackMember{}
	for rows.Next() {
		var m model.PeakTrackMember
		if err := rows.Scan(&m.ID, &m.TrackID, &m.PeakID, &m.TempC); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
