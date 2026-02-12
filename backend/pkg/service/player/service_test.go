package player

import (
	"testing"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestMovePlayerTowards(t *testing.T) {
	start := state.Position{Latitude: 39.9, Longitude: 116.3}
	target := state.Position{Latitude: 39.91, Longitude: 116.31}
	p := state.Player{ID: "p1", Position: start, Target: &target}
	next, moved := MovePlayerTowards(p, 500)
	if !moved {
		t.Fatalf("expected moved")
	}
	if next.Position == start {
		t.Fatalf("expected position change")
	}
}
