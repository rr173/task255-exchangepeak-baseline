package peak

import (
	"testing"

	"task255-exchangepeak/internal/model"
)

func TestAddRejectsUnsupportedUnitBeforeStoreAccess(t *testing.T) {
	svc := &Service{}
	_, err := svc.Add(AddInput{BatchID: "batch", TemperatureID: "temp", Unit: "khz"})
	if err != model.ErrInvalidInput {
		t.Fatalf("Add returned %v, want %v", err, model.ErrInvalidInput)
	}
}
