package snapshot

import (
	"testing"
	"time"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/store"
)

func TestSnapshotRoundTripsFrozenPayload(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/exchangepeak.db")
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()

	want := &model.AssignmentSnapshot{
		ID: "snap-1", BatchID: "batch-1", State: model.SnapPublished,
		FrozenAt: time.Unix(10, 20).UTC(), Payload: `{"batch_id":"batch-1"}`,
	}
	if err := st.CreateSnapshot(want); err != nil {
		t.Fatalf("CreateSnapshot() error = %v", err)
	}
	got, err := st.GetSnapshot(want.ID)
	if err != nil {
		t.Fatalf("GetSnapshot() error = %v", err)
	}
	if got.BatchID != want.BatchID || got.State != want.State || got.Payload != want.Payload || !got.FrozenAt.Equal(want.FrozenAt) {
		t.Fatalf("snapshot round trip = %+v, want %+v", got, want)
	}
}
