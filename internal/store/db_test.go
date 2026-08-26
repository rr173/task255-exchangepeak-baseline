package store

import "testing"

func TestOpenCreatesEmptyPersistentSchema(t *testing.T) {
	st, err := Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()

	stats, err := st.Stats()
	if err != nil {
		t.Fatalf("Stats() error = %v", err)
	}
	if stats.Samples != 0 || stats.Batches != 0 || stats.Peaks != 0 || stats.Tracks != 0 || stats.Candidates != 0 || stats.Snapshots != 0 {
		t.Fatalf("new store stats = %+v, want all zero", stats)
	}
}
