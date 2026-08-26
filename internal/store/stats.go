package store

// Stats 汇总各实体的行数，供运行自检使用。
type Stats struct {
	Samples    int `json:"samples"`
	Batches    int `json:"batches"`
	Peaks      int `json:"peaks"`
	Tracks     int `json:"tracks"`
	Candidates int `json:"candidates"`
	Snapshots  int `json:"snapshots"`
}

// Stats 统计全部实体的当前行数。
func (s *Store) Stats() (Stats, error) {
	var st Stats
	tables := map[string]*int{
		"samples":             &st.Samples,
		"spectrum_batches":    &st.Batches,
		"peaks":               &st.Peaks,
		"peak_tracks":         &st.Tracks,
		"exchange_candidates": &st.Candidates,
		"assignment_snapshots": &st.Snapshots,
	}
	for t, dst := range tables {
		var n int
		if err := s.DB.QueryRow("SELECT COUNT(*) FROM " + t).Scan(&n); err != nil {
			return st, err
		}
		*dst = n
	}
	return st, nil
}
