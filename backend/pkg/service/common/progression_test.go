package common

import (
	"testing"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestApplyAPGainSyncsLevelAndMaxXM(t *testing.T) {
	player := state.Player{
		ID:    "p1",
		Level: 1,
		AP:    2400,
		XM:    4500,
		MaxXM: 3000,
	}

	ApplyAPGain(&player, 200)

	if player.Level != 2 {
		t.Fatalf("expected level 2, got %d", player.Level)
	}
	if player.MaxXM != 4000 {
		t.Fatalf("expected maxXM 4000, got %d", player.MaxXM)
	}
	if player.XM != 4000 {
		t.Fatalf("expected XM clamped to 4000, got %d", player.XM)
	}
}
