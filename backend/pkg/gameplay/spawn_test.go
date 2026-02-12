package gameplay

import (
	"math/rand"
	"testing"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestRandomPositionInBoundsWithinRange(t *testing.T) {
	bounds := config.MapBoundsConfig{
		MinLat: 10,
		MaxLat: 10.1,
		MinLon: 20,
		MaxLon: 20.1,
	}
	rnd := rand.New(rand.NewSource(1))

	for i := 0; i < 32; i++ {
		pos := RandomPositionInBounds(bounds, rnd)
		if !PositionWithinBounds(pos, bounds) {
			t.Fatalf("expected position within bounds, got %+v", pos)
		}
	}
}

func TestRandomPositionInBoundsWithUnconfiguredBoundsReturnsZero(t *testing.T) {
	pos := RandomPositionInBounds(config.MapBoundsConfig{}, rand.New(rand.NewSource(1)))
	if pos.Latitude != 0 || pos.Longitude != 0 {
		t.Fatalf("expected zero position, got %+v", pos)
	}
}

func TestRandomPositionInBoundsWithInvalidBoundsReturnsZero(t *testing.T) {
	bounds := config.MapBoundsConfig{
		MinLat: 2,
		MaxLat: 1,
		MinLon: 4,
		MaxLon: 3,
	}
	pos := RandomPositionInBounds(bounds, rand.New(rand.NewSource(1)))
	if pos.Latitude != 0 || pos.Longitude != 0 {
		t.Fatalf("expected zero position for invalid bounds, got %+v", pos)
	}
}

func TestEnsurePositionInBoundsRepairsOutOfBounds(t *testing.T) {
	bounds := config.MapBoundsConfig{
		MinLat: 10,
		MaxLat: 11,
		MinLon: 20,
		MaxLon: 21,
	}
	origin := state.Position{Latitude: 9, Longitude: 19}
	pos, changed := EnsurePositionInBounds(origin, bounds, rand.New(rand.NewSource(2)))
	if !changed {
		t.Fatalf("expected changed=true for out-of-bounds position")
	}
	if !PositionWithinBounds(pos, bounds) {
		t.Fatalf("expected repaired position within bounds, got %+v", pos)
	}
}

func TestEnsurePositionInBoundsKeepsValidPosition(t *testing.T) {
	bounds := config.MapBoundsConfig{
		MinLat: 10,
		MaxLat: 11,
		MinLon: 20,
		MaxLon: 21,
	}
	origin := state.Position{Latitude: 10.4, Longitude: 20.6}
	pos, changed := EnsurePositionInBounds(origin, bounds, rand.New(rand.NewSource(3)))
	if changed {
		t.Fatalf("expected changed=false for in-bounds position")
	}
	if pos != origin {
		t.Fatalf("expected unchanged position, got %+v", pos)
	}
}
