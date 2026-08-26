package sample

import "testing"

func TestCreateRejectsInvalidConcentrationBeforePersistence(t *testing.T) {
	svc := &Service{}
	for _, concentration := range []float64{0, -1} {
		if _, err := svc.Create(CreateInput{Name: "sample", Compound: "compound", Concentration: concentration}); err == nil {
			t.Fatalf("Create accepted concentration %v", concentration)
		}
	}
}

func TestTempKeyUsesStableMillidegreeRepresentation(t *testing.T) {
	if got := tempKey(25.12349); got != "25.123" {
		t.Fatalf("tempKey(25.12349) = %q, want 25.123", got)
	}
}
