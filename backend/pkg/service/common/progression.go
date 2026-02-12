package common

import (
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func SyncPlayerProgression(player *state.Player) {
	if player == nil {
		return
	}
	derived := gameplay.LevelForAP(player.AP)
	if derived > player.Level {
		player.Level = derived
	}
	player.Level = gameplay.NormalizeLevel(player.Level)
	player.MaxXM = gameplay.MaxXMForLevel(player.Level)
	if player.XM > player.MaxXM {
		player.XM = player.MaxXM
	}
	if player.XM < 0 {
		player.XM = 0
	}
}

func ApplyAPGain(player *state.Player, amount int) {
	if player == nil || amount <= 0 {
		return
	}
	player.AP += amount
	SyncPlayerProgression(player)
}
