package snapshot_test

import (
	"errors"
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
	"task255-exchangepeak/internal/store"
)

func TestBug09EmptyBatchCannotPublishSnapshot(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	smp, _ := svc.Sample.Create(sample.CreateInput{Name: "s", Compound: "c", Solvent: "d", Concentration: 1})
	batch, _ := svc.CreateBatch(smp.ID, "empty")
	if _, err := svc.Snapshot.Freeze(batch.ID); !errors.Is(err, model.ErrInvalidInput) {
		t.Fatalf("Freeze() error = %v, want invalid input", err)
	}
	snapshots, err := svc.Snapshot.List(batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("empty batch created %d snapshots", len(snapshots))
	}
}
