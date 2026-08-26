package track_test

import (
	"errors"
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
	"task255-exchangepeak/internal/store"
)

func TestBug05AssociationRequiresCalibration(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	smp, _ := svc.Sample.Create(sample.CreateInput{Name: "s", Compound: "c", Solvent: "d", Concentration: 1})
	temps, _ := svc.Sample.AddTemperatures(smp.ID, []float64{25})
	batch, _ := svc.CreateBatch(smp.ID, "batch")
	std, _ := svc.CreateStandard(batch.ID, "TMS")
	if _, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temps[0].ID, TempC: 25, Unit: model.UnitPPM, ObservedShift: 0.05, Intensity: 1, IsStandard: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temps[0].ID, TempC: 25, Unit: model.UnitPPM, ObservedShift: 1.2, Intensity: 1}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Track.Associate(batch.ID); !errors.Is(err, model.ErrInvalidInput) {
		t.Fatalf("uncalibrated Associate() error = %v, want invalid input", err)
	}
	if err := svc.AddStandardPoint(std.ID, 25, 0); err != nil {
		t.Fatal(err)
	}
	if err := svc.LockStandard(std.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Peak.Calibrate(batch.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Track.Associate(batch.ID); err != nil {
		t.Fatalf("calibrated Associate() error = %v", err)
	}
}
