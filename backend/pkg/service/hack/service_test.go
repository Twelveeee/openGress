package hack

import (
	"strings"
	"testing"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	serr "github.com/Twelveeee/openGress/backend/pkg/service/errors"
	"github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestHackPortalCooldown(t *testing.T) {
	gs := state.NewGameState()
	playerSvc := player.New(gs, config.GameplayConfig{})
	svc := New(gs, playerSvc, config.GameplayConfig{})

	p := playerSvc.GetOrCreate("p1")
	p.Level = 8
	p.Position = state.Position{Latitude: 39.9, Longitude: 116.3}
	playerSvc.Upsert(p)

	gs.Portals.Upsert(state.Portal{ID: "portal-1", Level: 1, Position: state.Position{Latitude: 39.9001, Longitude: 116.3001}})

	first, err := svc.HackPortal("p1", "portal-1", time.Now())
	if err != nil || !first.Success {
		t.Fatalf("expected first hack success, err=%v result=%+v", err, first)
	}
	second, err := svc.HackPortal("p1", "portal-1", time.Now().Add(1*time.Second))
	if err != nil {
		t.Fatalf("unexpected error on second hack: %v", err)
	}
	if second.Success || second.CooldownSeconds <= 0 {
		t.Fatalf("expected cooldown failure, got %+v", second)
	}
}

func TestHackPortalRejectsWhenInventoryAlreadyOverCapacity(t *testing.T) {
	gs := state.NewGameState()
	playerSvc := player.New(gs, config.GameplayConfig{})
	svc := New(gs, playerSvc, config.GameplayConfig{})

	p := playerSvc.GetOrCreate("p-over")
	p.Level = 1
	p.Position = state.Position{Latitude: 39.9, Longitude: 116.3}
	p.Inventory = map[string]int{
		gameplay.ItemIDXMP(1): 501,
	}
	playerSvc.Upsert(p)

	gs.Portals.Upsert(state.Portal{
		ID:       "portal-over",
		Level:    1,
		Position: state.Position{Latitude: 39.9001, Longitude: 116.3001},
	})

	_, err := svc.HackPortal("p-over", "portal-over", time.Now())
	if !serr.IsKind(err, serr.KindBadRequest) {
		t.Fatalf("expected bad request error, got %v", err)
	}
	if err == nil || err.Error() != "inventory capacity exceeded" {
		t.Fatalf("expected inventory capacity exceeded, got %v", err)
	}

	after, ok := gs.Players.Get("p-over")
	if !ok {
		t.Fatalf("player not found")
	}
	if _, cooldownSet := after.HackCooldowns["portal-over"]; cooldownSet {
		t.Fatalf("expected no cooldown on rejected hack")
	}
}

func TestHackPortalAllowsOverflowAfterHack(t *testing.T) {
	gs := state.NewGameState()
	playerSvc := player.New(gs, config.GameplayConfig{})
	svc := newWithRand(gs, playerSvc, config.GameplayConfig{}, scriptedRand(
		0.0, 0.0,
		0.0, 0.0,
		0.0, 0.0,
		0.99,
		0.99,
	))

	p := playerSvc.GetOrCreate("p-cap")
	p.Level = 1
	p.Position = state.Position{Latitude: 39.9, Longitude: 116.3}
	p.Inventory = map[string]int{
		gameplay.ItemIDXMP(1): 500,
	}
	playerSvc.Upsert(p)

	gs.Portals.Upsert(state.Portal{
		ID:       "portal-cap",
		Level:    1,
		Position: state.Position{Latitude: 39.9001, Longitude: 116.3001},
	})

	result, err := svc.HackPortal("p-cap", "portal-cap", time.Now())
	if err != nil || !result.Success {
		t.Fatalf("expected hack success, err=%v result=%+v", err, result)
	}

	after, ok := gs.Players.Get("p-cap")
	if !ok {
		t.Fatalf("player not found")
	}
	generalUsed, keyUsed := gameplay.InventoryCounts(after.Inventory)
	generalCap, keyCap := gameplay.CapacityForLevel(after.Level)
	if generalUsed <= generalCap {
		t.Fatalf("expected general inventory overflow, got %d <= %d", generalUsed, generalCap)
	}
	if keyUsed > keyCap {
		t.Fatalf("unexpected key overflow, got %d > %d", keyUsed, keyCap)
	}
}

func TestGrantHackItemsCanProduceUSAndMod(t *testing.T) {
	gs := state.NewGameState()
	playerSvc := player.New(gs, config.GameplayConfig{})
	svc := newWithRand(gs, playerSvc, config.GameplayConfig{}, scriptedRand(
		0.72, 0.50,
		0.10, 0.50,
		0.10, 0.50,
		0.00,
		0.95,
		0.99,
	))

	p := playerSvc.GetOrCreate("p-drop")
	p.Level = 8
	p.Faction = "RESISTANCE"
	p.Position = state.Position{Latitude: 39.9, Longitude: 116.3}
	p.Inventory = map[string]int{}
	playerSvc.Upsert(p)

	gs.Portals.Upsert(state.Portal{
		ID:       "portal-drop",
		Level:    8,
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 39.9001, Longitude: 116.3001},
	})

	result, err := svc.HackPortal("p-drop", "portal-drop", time.Now())
	if err != nil || !result.Success {
		t.Fatalf("expected hack success, err=%v result=%+v", err, result)
	}

	var hasUS, hasMod bool
	for _, item := range result.ItemsGained {
		if strings.HasPrefix(item, gameplay.ItemPrefixUS) {
			hasUS = true
		}
		if strings.HasPrefix(item, gameplay.ItemPrefixMod) {
			hasMod = true
		}
	}
	if !hasUS {
		t.Fatalf("expected US drop in %+v", result.ItemsGained)
	}
	if !hasMod {
		t.Fatalf("expected MOD drop in %+v", result.ItemsGained)
	}
}

func TestKeyDropRespectsPerPortalLimitEvenWhenCapacityIgnored(t *testing.T) {
	gs := state.NewGameState()
	playerSvc := player.New(gs, config.GameplayConfig{})
	svc := newWithRand(gs, playerSvc, config.GameplayConfig{}, scriptedRand(
		0.10, 0.50,
		0.10, 0.50,
		0.10, 0.50,
		0.99,
		0.00,
	))

	p := playerSvc.GetOrCreate("p-key")
	p.Level = 8
	p.Position = state.Position{Latitude: 39.9, Longitude: 116.3}
	keyID := gameplay.ItemIDKey("portal-key")
	p.Inventory = map[string]int{
		keyID: 2,
	}
	playerSvc.Upsert(p)

	gs.Portals.Upsert(state.Portal{
		ID:       "portal-key",
		Level:    8,
		Position: state.Position{Latitude: 39.9001, Longitude: 116.3001},
	})

	result, err := svc.HackPortal("p-key", "portal-key", time.Now())
	if err != nil || !result.Success {
		t.Fatalf("expected hack success, err=%v result=%+v", err, result)
	}

	after, ok := gs.Players.Get("p-key")
	if !ok {
		t.Fatalf("player not found")
	}
	if got := gameplay.KeyCount(after.Inventory, "portal-key"); got != 2 {
		t.Fatalf("expected portal key count to stay 2, got %d", got)
	}
	for _, item := range result.ItemsGained {
		if item == keyID {
			t.Fatalf("expected no key drop when already at key limit, got %+v", result.ItemsGained)
		}
	}
}

func TestModTypeWeightsByRarityCommonExcludesLinkAmp(t *testing.T) {
	commonWeights := modTypeWeightsByRarity(modRarityCommon)
	for _, option := range commonWeights {
		if option.Value == "LINK_AMP" {
			t.Fatalf("expected common mod pool excludes LINK_AMP, got %+v", commonWeights)
		}
	}

	rareWeights := modTypeWeightsByRarity(modRarityRare)
	hasRareLinkAmp := false
	for _, option := range rareWeights {
		if option.Value == "LINK_AMP" {
			hasRareLinkAmp = true
			break
		}
	}
	if !hasRareLinkAmp {
		t.Fatalf("expected rare mod pool includes LINK_AMP, got %+v", rareWeights)
	}
}

func scriptedRand(values ...float64) func() float64 {
	index := 0
	return func() float64 {
		if index >= len(values) {
			return 0.99
		}
		value := values[index]
		index++
		return value
	}
}
