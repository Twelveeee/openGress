package portal

import (
	"testing"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	serviceerrors "github.com/Twelveeee/openGress/backend/pkg/service/errors"
	playersvc "github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestComputeBaseLinkRangeByResonatorPattern(t *testing.T) {
	portal := state.Portal{
		ID: "p1",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 2},
			2: {Slot: 2, Level: 2},
			3: {Slot: 3, Level: 2},
			4: {Slot: 4, Level: 2},
			5: {Slot: 5, Level: 1},
			6: {Slot: 6, Level: 1},
			7: {Slot: 7, Level: 1},
			8: {Slot: 8, Level: 1},
		},
	}
	if got := ComputeBaseLinkRangeMeters(portal); got != 810.0 {
		t.Fatalf("expected 810m, got %.1f", got)
	}
}

func TestComputePortalLevelRoundsAverage(t *testing.T) {
	portal := state.Portal{
		ID: "p1",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 8},
			2: {Slot: 2, Level: 8},
			3: {Slot: 3, Level: 8},
			4: {Slot: 4, Level: 8},
			5: {Slot: 5, Level: 7},
			6: {Slot: 6, Level: 7},
			7: {Slot: 7, Level: 7},
			8: {Slot: 8, Level: 7},
		},
	}
	if got := ComputePortalLevel(portal); got != 8 {
		t.Fatalf("expected rounded level 8, got %d", got)
	}
}

func TestDeployResonatorCaptureAwardsBaseAndCaptureAP(t *testing.T) {
	gameState := state.NewGameState()
	cfg := config.GameplayConfig{InteractRangeM: 40}
	players := playersvc.New(gameState, cfg)
	svc := New(gameState, cfg, players)

	gameState.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     5,
		XM:        3000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDResonator(5): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Faction:  "NEUTRAL",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	result, err := svc.DeployResonator(DeployResonatorCmd{
		PlayerID:        "p1",
		PortalID:        "portal-1",
		Slot:            1,
		Level:           5,
		ExpectedVersion: 0,
	}, time.Now())
	if err != nil {
		t.Fatalf("deploy resonator failed: %v", err)
	}
	if result.APGained != 625 {
		t.Fatalf("expected AP gained 625, got %d", result.APGained)
	}
	if result.Player.AP != 625 {
		t.Fatalf("expected player AP 625, got %d", result.Player.AP)
	}
}

func TestDeployResonatorUpgradeAwardsUpgradeAP(t *testing.T) {
	gameState := state.NewGameState()
	cfg := config.GameplayConfig{InteractRangeM: 40}
	players := playersvc.New(gameState, cfg)
	svc := New(gameState, cfg, players)

	gameState.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     5,
		XM:        3000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDResonator(5): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Portals.Upsert(state.Portal{
		ID:      "portal-1",
		Faction: "RESISTANCE",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 4, Energy: ResonatorEnergyForLevel(4), Version: 2, PlayerID: "p1"},
		},
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	result, err := svc.DeployResonator(DeployResonatorCmd{
		PlayerID:        "p1",
		PortalID:        "portal-1",
		Slot:            1,
		Level:           5,
		ExpectedVersion: 2,
	}, time.Now())
	if err != nil {
		t.Fatalf("deploy resonator failed: %v", err)
	}
	if result.APGained != 65 {
		t.Fatalf("expected AP gained 65, got %d", result.APGained)
	}
	if result.Player.AP != 65 {
		t.Fatalf("expected player AP 65, got %d", result.Player.AP)
	}
}

func TestDeployModNormalizesPortalShieldAlias(t *testing.T) {
	gameState := state.NewGameState()
	cfg := config.GameplayConfig{
		InteractRangeM: 40,
		XMCost: config.XMCostConfig{
			DeployModFlat: 100,
		},
	}
	players := playersvc.New(gameState, cfg)
	svc := New(gameState, cfg, players)

	gameState.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     5,
		XM:        3000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDMod("SHIELD", "COMMON"): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Faction:  "RESISTANCE",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	result, err := svc.DeployMod(DeployModCmd{
		PlayerID:        "p1",
		PortalID:        "portal-1",
		Slot:            1,
		ModType:         "PORTAL_SHIELD",
		Rarity:          "COMMON",
		ExpectedVersion: 0,
	}, time.Now())
	if err != nil {
		t.Fatalf("deploy mod failed: %v", err)
	}
	if result.ModType != "SHIELD" {
		t.Fatalf("expected normalized mod type SHIELD, got %s", result.ModType)
	}
}

func TestDeployModAwardsAPAndConsumesXMByRarity(t *testing.T) {
	gameState := state.NewGameState()
	cfg := config.GameplayConfig{
		InteractRangeM: 40,
		XMCost: config.XMCostConfig{
			DeployModByRarity: map[string]int{
				"COMMON": 400,
			},
		},
		APReward: config.APRewardConfig{
			DeployMod: 150,
		},
	}
	players := playersvc.New(gameState, cfg)
	svc := New(gameState, cfg, players)

	gameState.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     5,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDMod("SHIELD", "COMMON"): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Portals.Upsert(state.Portal{
		ID:       "portal-1",
		Faction:  "RESISTANCE",
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	result, err := svc.DeployMod(DeployModCmd{
		PlayerID:        "p1",
		PortalID:        "portal-1",
		Slot:            1,
		ModType:         "SHIELD",
		Rarity:          "COMMON",
		ExpectedVersion: 0,
	}, time.Now())
	if err != nil {
		t.Fatalf("deploy mod failed: %v", err)
	}
	if result.APGained != 150 {
		t.Fatalf("expected AP gained 150, got %d", result.APGained)
	}
	if result.XMCost != 400 {
		t.Fatalf("expected XM cost 400, got %d", result.XMCost)
	}
	if result.Player.AP != 150 {
		t.Fatalf("expected player AP 150, got %d", result.Player.AP)
	}
	if result.Player.XM != 600 {
		t.Fatalf("expected player XM 600, got %d", result.Player.XM)
	}
}

func TestChargePortalNoOpWhenPortalAlreadyFull(t *testing.T) {
	gameState := state.NewGameState()
	cfg := config.GameplayConfig{
		InteractRangeM: 40,
		XMCost: config.XMCostConfig{
			ChargePerXM: 1,
		},
	}
	players := playersvc.New(gameState, cfg)
	svc := New(gameState, cfg, players)

	gameState.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Portals.Upsert(state.Portal{
		ID:      "portal-1",
		Faction: "RESISTANCE",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: ResonatorEnergyForLevel(1)},
		},
		Energy:   ResonatorEnergyForLevel(1),
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	result, err := svc.Charge(ChargeCmd{
		PlayerID: "p1",
		PortalID: "portal-1",
		Amount:   500,
	}, time.Now())
	if err != nil {
		t.Fatalf("charge failed: %v", err)
	}
	if result.AppliedAmount != 0 {
		t.Fatalf("expected applied amount 0, got %d", result.AppliedAmount)
	}
	if result.XMCost != 0 {
		t.Fatalf("expected xm cost 0, got %d", result.XMCost)
	}
	if result.Player.XM != 1000 {
		t.Fatalf("expected player xm unchanged, got %d", result.Player.XM)
	}
	if result.Portal.Energy != ResonatorEnergyForLevel(1) {
		t.Fatalf("expected portal energy unchanged, got %d", result.Portal.Energy)
	}
}

func TestChargePortalCostUsesActualAppliedAmount(t *testing.T) {
	gameState := state.NewGameState()
	cfg := config.GameplayConfig{
		InteractRangeM: 40,
		XMCost: config.XMCostConfig{
			ChargePerXM: 1,
		},
	}
	players := playersvc.New(gameState, cfg)
	svc := New(gameState, cfg, players)

	gameState.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Portals.Upsert(state.Portal{
		ID:      "portal-1",
		Faction: "RESISTANCE",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 980},
		},
		Energy:   980,
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	result, err := svc.Charge(ChargeCmd{
		PlayerID: "p1",
		PortalID: "portal-1",
		Amount:   500,
	}, time.Now())
	if err != nil {
		t.Fatalf("charge failed: %v", err)
	}
	if result.AppliedAmount != 20 {
		t.Fatalf("expected applied amount 20, got %d", result.AppliedAmount)
	}
	if result.XMCost != 20 {
		t.Fatalf("expected xm cost 20, got %d", result.XMCost)
	}
	if result.Player.XM != 980 {
		t.Fatalf("expected player xm 980, got %d", result.Player.XM)
	}
	if result.Portal.Energy != 1000 {
		t.Fatalf("expected portal energy 1000, got %d", result.Portal.Energy)
	}
}

func TestChargePortalRejectsWhenXMInsufficient(t *testing.T) {
	gameState := state.NewGameState()
	cfg := config.GameplayConfig{
		InteractRangeM: 40,
		XMCost: config.XMCostConfig{
			ChargePerXM: 1,
		},
	}
	players := playersvc.New(gameState, cfg)
	svc := New(gameState, cfg, players)

	gameState.Players.Upsert(state.Player{
		ID:        "p1",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        10,
		MaxXM:     3000,
		Inventory: map[string]int{},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gameState.Portals.Upsert(state.Portal{
		ID:      "portal-1",
		Faction: "RESISTANCE",
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 900},
		},
		Energy:   900,
		Position: state.Position{Latitude: 0, Longitude: 0},
	})

	_, err := svc.Charge(ChargeCmd{
		PlayerID: "p1",
		PortalID: "portal-1",
		Amount:   500,
	}, time.Now())
	if err == nil {
		t.Fatalf("expected insufficient xm error")
	}
	if !serviceerrors.IsKind(err, serviceerrors.KindBadRequest) {
		t.Fatalf("expected bad request, got %v", err)
	}
	if err.Error() != "insufficient xm" {
		t.Fatalf("expected insufficient xm message, got %q", err.Error())
	}
	player, _ := gameState.Players.Get("p1")
	if player.XM != 10 {
		t.Fatalf("expected player xm unchanged, got %d", player.XM)
	}
	portal, _ := gameState.Portals.Get("portal-1")
	if portal.Energy != 900 {
		t.Fatalf("expected portal energy unchanged, got %d", portal.Energy)
	}
}
