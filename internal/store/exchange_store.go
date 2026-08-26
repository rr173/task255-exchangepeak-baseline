package store

import (
	"database/sql"
	"time"

	"task255-exchangepeak/internal/model"
)

// CreateCandidate 写入交换候选。
func (s *Store) CreateCandidate(c *model.ExchangeCandidate) error {
	_, err := s.DB.Exec(
		`INSERT INTO exchange_candidates
		 (id,batch_id,track_a_id,track_b_id,kind,score,state,reason,created_at)
		 VALUES (?,?,?,?,?,?,?,?,?)`,
		c.ID, c.BatchID, c.TrackAID, c.TrackBID, c.Kind, c.Score, c.State, c.Reason,
		c.CreatedAt.Format(time.RFC3339Nano),
	)
	return err
}

// GetCandidate 按 ID 读取候选。
func (s *Store) GetCandidate(id string) (*model.ExchangeCandidate, error) {
	row := s.DB.QueryRow(
		`SELECT id,batch_id,track_a_id,track_b_id,kind,score,state,reason,created_at
		 FROM exchange_candidates WHERE id=?`, id)
	return scanCandidate(row)
}

// ListCandidatesByBatch 列出批次全部交换候选。
func (s *Store) ListCandidatesByBatch(batchID string) ([]model.ExchangeCandidate, error) {
	rows, err := s.DB.Query(
		`SELECT id,batch_id,track_a_id,track_b_id,kind,score,state,reason,created_at
		 FROM exchange_candidates WHERE batch_id=? ORDER BY created_at`, batchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []model.ExchangeCandidate{}
	for rows.Next() {
		c, err := scanCandidate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

// SetCandidateState 更新候选状态（确认/否决）。
func (s *Store) SetCandidateState(id, state string) error {
	_, err := s.DB.Exec(`UPDATE exchange_candidates SET state=? WHERE id=?`, state, id)
	return err
}

// DeleteCandidatesByBatch 删除批次全部交换候选（用于重新评分前的清空）。
func (s *Store) DeleteCandidatesByBatch(batchID string) error {
	_, err := s.DB.Exec(`DELETE FROM exchange_candidates WHERE batch_id=?`, batchID)
	return err
}

func scanCandidate(sc scanner) (*model.ExchangeCandidate, error) {
	var c model.ExchangeCandidate
	var created string
	if err := sc.Scan(&c.ID, &c.BatchID, &c.TrackAID, &c.TrackBID, &c.Kind,
		&c.Score, &c.State, &c.Reason, &created); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	t, err := time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return nil, err
	}
	c.CreatedAt = t
	return &c, nil
}
