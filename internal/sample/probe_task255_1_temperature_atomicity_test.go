package sample_test

import (
	"errors"
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/store"
)

func TestBug01TemperatureAppendIsAtomic(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	svc := sample.New(st)
	smp, err := svc.Create(sample.CreateInput{Name: "s", Compound: "c", Solvent: "d", Concentration: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.AddTemperatures(smp.ID, []float64{25, 50, 50}); !errors.Is(err, model.ErrTempConflict) {
		t.Fatalf("AddTemperatures error = %v, want duplicate temperature", err)
	}
	points, err := svc.Temperatures(smp.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(points) != 0 {
		t.Fatalf("failed append left %d temperature points, want 0", len(points))
	}
}
