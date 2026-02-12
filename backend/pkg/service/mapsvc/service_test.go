package mapsvc

import (
	"testing"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestCollectPortals(t *testing.T) {
	gs := state.NewGameState()
	gs.Portals.Upsert(state.Portal{ID: "in", Position: state.Position{Latitude: 39.9, Longitude: 116.3}})
	gs.Portals.Upsert(state.Portal{ID: "out", Position: state.Position{Latitude: 10, Longitude: 10}})
	svc := New(gs)
	got := svc.CollectPortals(state.Bounds{MinLat: 39.8, MaxLat: 40.0, MinLon: 116.2, MaxLon: 116.4})
	if len(got) != 1 || got[0].ID != "in" {
		t.Fatalf("unexpected portals: %+v", got)
	}
}
