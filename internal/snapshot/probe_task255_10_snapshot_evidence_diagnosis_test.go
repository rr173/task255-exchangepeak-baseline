package snapshot_test

import (
	"encoding/json"
	"testing"

	"task255-exchangepeak/internal/model"
	"task255-exchangepeak/internal/peak"
	"task255-exchangepeak/internal/sample"
	"task255-exchangepeak/internal/service"
	"task255-exchangepeak/internal/store"
)

func TestBug10SnapshotRetainsCrossEntityEvidenceAfterRestart(t *testing.T) {
	dbPath := t.TempDir() + "/db.sqlite"
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	svc := service.New(st)
	smp, _ := svc.Sample.Create(sample.CreateInput{Name: "ethanol", Compound: "C2H5OH", Solvent: "CDCl3", Concentration: .5})
	temps, _ := svc.Sample.AddTemperatures(smp.ID, []float64{25, 50, 75})
	batch, _ := svc.CreateBatch(smp.ID, "evidence")
	std, _ := svc.CreateStandard(batch.ID, "TMS")
	for _, temp := range temps {
		if err := svc.AddStandardPoint(std.ID, temp.TempC, 0); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temp.ID, TempC: temp.TempC, Unit: model.UnitPPM, ObservedShift: .05, Intensity: 1, IsStandard: true}); err != nil {
			t.Fatal(err)
		}
	}
	observedA := map[float64]float64{25: 1.5, 50: 1.4, 75: 1.3}
	observedB := map[float64]float64{25: 1.8, 50: 1.5, 75: 1.32}
	for _, temp := range temps {
		if _, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temp.ID, TempC: temp.TempC, Unit: model.UnitPPM, ObservedShift: observedA[temp.TempC], Intensity: 1}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Peak.Add(peak.AddInput{BatchID: batch.ID, TemperatureID: temp.ID, TempC: temp.TempC, Unit: model.UnitPPM, ObservedShift: observedB[temp.TempC], Intensity: 1}); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.LockStandard(std.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Peak.Calibrate(batch.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Track.Associate(batch.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Exchange.Score(batch.ID); err != nil {
		t.Fatal(err)
	}
	snap, err := svc.Snapshot.Freeze(batch.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	snap, err = st.GetSnapshot(snap.ID)
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(snap.Payload), &payload); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"sample", "temperatures", "standards", "tracks", "candidates"} {
		if _, ok := payload[key]; !ok {
			t.Fatalf("snapshot payload missing %q: %s", key, snap.Payload)
		}
	}
	tracks, ok := payload["tracks"].([]interface{})
	if !ok || len(tracks) == 0 {
		t.Fatalf("snapshot tracks evidence = %#v", payload["tracks"])
	}
	first, ok := tracks[0].(map[string]interface{})
	if !ok {
		t.Fatalf("snapshot first track = %#v", tracks[0])
	}
	members, ok := first["members"].([]interface{})
	if !ok || len(members) == 0 {
		t.Fatalf("snapshot track members = %#v", first["members"])
	}
}
