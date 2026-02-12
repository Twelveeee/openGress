package link

import (
	"testing"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestLinkCrossesExisting(t *testing.T) {
	from := state.Position{Latitude: 0, Longitude: 0}
	to := state.Position{Latitude: 1, Longitude: 1}
	links := []state.Link{{
		ID:           "l1",
		FromPortalID: "a",
		ToPortalID:   "b",
		FromPosition: state.Position{Latitude: 0, Longitude: 1},
		ToPosition:   state.Position{Latitude: 1, Longitude: 0},
	}}
	if !LinkCrossesExisting(from, to, "x", "y", links) {
		t.Fatalf("expected crossing link")
	}
}
