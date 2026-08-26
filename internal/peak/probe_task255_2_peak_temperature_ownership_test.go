package peak_test

import (
	"errors"
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
	"task255-exchangepeak/internal/store"
)

func TestBug02PeakRejectsForeignTemperaturePoint(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)

	a, err := svc.Sample.Create(sample.CreateInput{Name: "a", Compound: "ca", Solvent: "d", Concentration: 1})
	if err != nil {
		t.Fatal(err)
	}
	b, err := svc.Sample.Create(sample.CreateInput{Name: "b", Compound: "cb", Solvent: "d", Concentration: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Sample.AddTemperatures(a.ID, []float64{25}); err != nil {
		t.Fatal(err)
	}
	foreign, err := svc.Sample.AddTemperatures(b.ID, []float64{25})
	if err != nil {
		t.Fatal(err)
	}
	batch, err := svc.CreateBatch(a.ID, "batch")
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Peak.Add(structuredPeak(batch.ID, foreign[0].ID, 25))
	if !errors.Is(err, model.ErrInvalidInput) {
		t.Fatalf("foreign temperature point error = %v, want invalid input", err)
	}
	peaks, err := svc.Peak.List(batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(peaks) != 0 {
		t.Fatalf("foreign temperature request created %d peaks", len(peaks))
	}
}

func structuredPeak(batchID, temperatureID string, temp float64) peak.AddInput {
	var in peak.AddInput
	in.BatchID, in.TemperatureID, in.TempC = batchID, temperatureID, temp
	in.Unit, in.ObservedShift, in.Intensity = model.UnitPPM, 1, 1
	return in
}
