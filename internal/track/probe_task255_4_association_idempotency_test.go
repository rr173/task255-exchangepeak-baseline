package track_test

import (
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
	"task255-exchangepeak/internal/store"
)

func TestBug04AssociationIsIdempotent(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/db.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := service.New(st)
	smp, _ := svc.Sample.Create(sample.CreateInput{Name: "s", Compound: "c", Solvent: "d", Concentration: 1})
	temps, _ := svc.Sample.AddTemperatures(smp.ID, []float64{25, 50, 75})
	batch, _ := svc.CreateBatch(smp.ID, "batch")
	std, _ := svc.CreateStandard(batch.ID, "TMS")
	for _, temp := range temps {
		if err := svc.AddStandardPoint(std.ID, temp.TempC, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temp.ID, TempC: temp.TempC, Unit: model.UnitPPM, ObservedShift: 0.05, Intensity: 1, IsStandard: true}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temp.ID, TempC: temp.TempC, Unit: model.UnitPPM, ObservedShift: 1 + temp.TempC/100, Intensity: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.LockStandard(std.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Peak.Calibrate(batch.ID); err != nil {
		t.Fatal(err)
	}
	first, err := svc.Track.Associate(batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	firstMembers := 0
	for _, tr := range first {
		members, err := svc.Track.Members(tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		firstMembers += len(members)
	}
	second, err := svc.Track.Associate(batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	secondMembers := 0
	for _, tr := range second {
		members, err := svc.Track.Members(tr.ID)
		if err != nil {
			t.Fatal(err)
		}
		secondMembers += len(members)
	}
	allTracks, err := svc.Store.ListTracksByBatch(batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(allTracks) != len(first) || len(second) != len(first) || secondMembers != firstMembers {
		t.Fatalf("association changed on rerun: first tracks/members=%d/%d second=%d/%d", len(first), firstMembers, len(second), secondMembers)
	}
}
