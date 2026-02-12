package field

import (
	"testing"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestComputeFieldMUPositive(t *testing.T) {
	mu := ComputeFieldMU(
		state.Position{Latitude: 39.9, Longitude: 116.3},
		state.Position{Latitude: 39.91, Longitude: 116.31},
		state.Position{Latitude: 39.905, Longitude: 116.33},
	)
	if mu <= 0 {
		t.Fatalf("expected positive MU, got %d", mu)
	}
}

func TestFieldExists(t *testing.T) {
	ids := [3]string{"a", "b", "c"}
	fields := []state.Field{{ID: "f1", PortalIDs: ids}}
	if !FieldExists(fields, ids) {
		t.Fatalf("expected field exists")
	}
}
