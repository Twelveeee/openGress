package combat

import (
	"math"
	"math/rand"
	"testing"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/config"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/service/errors"
	"github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func testGameplayConfig() config.GameplayConfig {
	return config.GameplayConfig{
		XMCost: config.XMCostConfig{
			AttackXMPBase:     50,
			AttackXMPPerLevel: 50,
			AttackUSBase:      50,
			AttackUSPerLevel:  50,
		},
	}
}

func newCombatService(t *testing.T, cfg config.GameplayConfig) (*Service, *state.GameState) {
	t.Helper()
	gs := state.NewGameState()
	playerSvc := player.New(gs, cfg)
	svc := New(gs, cfg, playerSvc)
	svc.setRandSource(rand.NewSource(1))
	return svc, gs
}

func TestAttackInvalidWeaponType(t *testing.T) {
	svc, _ := newCombatService(t, testGameplayConfig())
	_, err := svc.Attack(AttackCmd{PlayerID: "p1", PortalID: "portal", WeaponType: "BAD", WeaponLevel: 1}, time.Now())
	if err == nil || !errors.IsKind(err, errors.KindBadRequest) {
		t.Fatalf("expected bad request error, got %v", err)
	}
}

func TestAttackAwardsAPByDestroyedResonatorCount(t *testing.T) {
	cfg := testGameplayConfig()
	cfg.APReward = config.APRewardConfig{
		DestroyResonator: 75,
	}
	svc, gs := newCombatService(t, cfg)
	gs.Players.Upsert(state.Player{
		ID:        "attacker",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		MaxXM:     3000,
		Inventory: map[string]int{gameplay.ItemIDXMP(2): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "enemy",
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 50},
		},
		Level:  1,
		Energy: 50,
	})

	result, err := svc.Attack(AttackCmd{PlayerID: "attacker", PortalID: "enemy", WeaponType: "XMP", WeaponLevel: 2}, time.Now())
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}
	if result.Payload.ResonatorsDestroyed <= 0 {
		t.Fatalf("expected at least one destroyed resonator")
	}
	if result.PlayerDelta.APGained != 75 {
		t.Fatalf("expected AP delta 75, got %d", result.PlayerDelta.APGained)
	}
}

func TestDefaultWeaponSpecMatchesItemAttributes(t *testing.T) {
	svc, _ := newCombatService(t, testGameplayConfig())
	xmpL4 := svc.attackCfg.weaponSpecs["XMP"][4]
	if xmpL4.MaxDamage != 900 || xmpL4.RadiusM != 72 || xmpL4.CostXM != 200 {
		t.Fatalf("unexpected XMP L4 spec: %+v", xmpL4)
	}
	usL8 := svc.attackCfg.weaponSpecs["US"][8]
	if usL8.MaxDamage != 5400 || usL8.RadiusM != 30 || usL8.CostXM != 400 {
		t.Fatalf("unexpected US L8 spec: %+v", usL8)
	}
}

func TestAttackOnlyDamagesEnemyPortals(t *testing.T) {
	svc, gs := newCombatService(t, testGameplayConfig())
	gs.Players.Upsert(state.Player{
		ID:        "attacker",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        5000,
		Inventory: map[string]int{gameplay.ItemIDXMP(8): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "enemy",
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 0, Longitude: 0.00005},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 8, Energy: 6000},
		},
		Level:  8,
		Energy: 6000,
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "friend",
		Faction:  "RESISTANCE",
		Position: state.Position{Latitude: 0, Longitude: 0.00005},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 8, Energy: 6000},
		},
		Level:  8,
		Energy: 6000,
	})

	_, err := svc.Attack(AttackCmd{PlayerID: "attacker", PortalID: "enemy", WeaponType: "XMP", WeaponLevel: 8}, time.Now())
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}
	enemy, _ := gs.Portals.Get("enemy")
	friend, _ := gs.Portals.Get("friend")
	if enemy.Energy >= 6000 {
		t.Fatalf("expected enemy portal damaged, got %d", enemy.Energy)
	}
	if friend.Energy != 6000 {
		t.Fatalf("friendly portal should not be damaged, got %d", friend.Energy)
	}
}

func TestAttackAllowsEmptyTargetAndStillConsumesResource(t *testing.T) {
	svc, gs := newCombatService(t, testGameplayConfig())
	gs.Players.Upsert(state.Player{
		ID:        "attacker",
		Faction:   "RESISTANCE",
		Level:     3,
		XM:        1000,
		Inventory: map[string]int{gameplay.ItemIDXMP(2): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})

	result, err := svc.Attack(AttackCmd{PlayerID: "attacker", PortalID: "", WeaponType: "XMP", WeaponLevel: 2}, time.Now())
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}
	if result.Payload.PortalID != "" {
		t.Fatalf("expected empty portalId in payload, got %q", result.Payload.PortalID)
	}
	if result.Payload.DamageDealt != 0 {
		t.Fatalf("expected empty-fire damage 0, got %d", result.Payload.DamageDealt)
	}
	if len(result.Payload.PortalDamages) != 0 {
		t.Fatalf("expected no portal damages on empty-fire")
	}
	if len(result.Payload.Counterattacks) != 0 {
		t.Fatalf("expected no counterattacks on empty-fire")
	}
	if result.PlayerDelta.InventoryDelta[gameplay.ItemIDXMP(2)] != -1 {
		t.Fatalf("expected weapon consume -1")
	}
	if result.PlayerDelta.XMDelta != -100 {
		t.Fatalf("expected xm delta -100, got %d", result.PlayerDelta.XMDelta)
	}
}

func TestCounterattacksIncludesAllHitPortals(t *testing.T) {
	cfg := testGameplayConfig()
	cfg.Attack = config.AttackConfig{
		Counterattack: config.AttackCounterattackConfig{
			BaseTriggerChance:   1,
			MaxTriggerChance:    1,
			TurretTriggerBonus:  0,
			BaseCritChance:      0,
			TurretCritBonus:     0,
			CritMultiplier:      1,
			ForceAmpBaseBonus:   0,
			ForceAmpExtraBonus:  0,
			DamageByPortalLevel: map[int]int{1: 120},
		},
		Crit: config.AttackCritConfig{
			XMP: config.AttackWeaponCritConfig{Chance: 0, Multiplier: 1},
		},
	}
	svc, gs := newCombatService(t, cfg)
	gs.Players.Upsert(state.Player{
		ID:        "attacker",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        1000,
		Inventory: map[string]int{gameplay.ItemIDXMP(1): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "enemy-1",
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 0, Longitude: 0.00005},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 8, Energy: 6000},
		},
		Level:  1,
		Energy: 6000,
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "enemy-2",
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 0, Longitude: 0.0001},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 8, Energy: 6000},
		},
		Level:  1,
		Energy: 6000,
	})

	result, err := svc.Attack(AttackCmd{PlayerID: "attacker", PortalID: "enemy-1", WeaponType: "XMP", WeaponLevel: 1}, time.Now())
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}
	if len(result.Payload.Counterattacks) != 2 {
		t.Fatalf("expected 2 counterattacks, got %d", len(result.Payload.Counterattacks))
	}
	if !result.Payload.Counterattack {
		t.Fatalf("expected counterattack aggregate flag true")
	}
	if result.Payload.CounterattackDamage != 240 {
		t.Fatalf("expected aggregate counterattack damage 240, got %d", result.Payload.CounterattackDamage)
	}
	if result.PlayerDelta.XMDelta != -290 {
		t.Fatalf("expected xm delta -290, got %d", result.PlayerDelta.XMDelta)
	}
}

func TestMitigationCapAtNinetyFivePercent(t *testing.T) {
	cfg := testGameplayConfig()
	cfg.Attack = config.AttackConfig{
		DistanceFalloffTiers: []config.AttackFalloffTier{{MaxDistanceRatio: 1, Multiplier: 1}},
		Mitigation: config.AttackMitigationConfig{
			MaxPercent: 95,
			ShieldValues: map[string]float64{
				"AEGIS": 70,
			},
		},
		Crit: config.AttackCritConfig{
			XMP: config.AttackWeaponCritConfig{Chance: 0, Multiplier: 1},
		},
	}
	svc, gs := newCombatService(t, cfg)
	gs.Players.Upsert(state.Player{
		ID:        "attacker",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        5000,
		Inventory: map[string]int{gameplay.ItemIDXMP(8): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "enemy",
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 0, Longitude: 0.00005},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Mods: map[int]state.ModSlot{
			1: {Slot: 1, ModType: "SHIELD", Rarity: "AEGIS"},
			2: {Slot: 2, ModType: "SHIELD", Rarity: "AEGIS"},
			3: {Slot: 3, ModType: "SHIELD", Rarity: "AEGIS"},
			4: {Slot: 4, ModType: "SHIELD", Rarity: "AEGIS"},
		},
		Level:  1,
		Energy: 1000,
	})

	result, err := svc.Attack(AttackCmd{PlayerID: "attacker", PortalID: "enemy", WeaponType: "XMP", WeaponLevel: 8}, time.Now())
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}
	portalState, _ := gs.Portals.Get("enemy")
	damage := 1000 - portalState.Energy
	if damage != 135 {
		t.Fatalf("expected 135 damage after 95%% mitigation, got %d", damage)
	}
	if math.Abs(result.Payload.MitigationApplied-0.95) > 0.0001 {
		t.Fatalf("unexpected mitigation ratio: %f", result.Payload.MitigationApplied)
	}
}

func TestCounterattackAndResourceDelta(t *testing.T) {
	cfg := testGameplayConfig()
	cfg.Attack = config.AttackConfig{
		Counterattack: config.AttackCounterattackConfig{
			BaseTriggerChance:   1,
			MaxTriggerChance:    1,
			TurretTriggerBonus:  0,
			BaseCritChance:      0,
			TurretCritBonus:     0,
			CritMultiplier:      1,
			ForceAmpBaseBonus:   0,
			ForceAmpExtraBonus:  0,
			DamageByPortalLevel: map[int]int{4: 500},
		},
		Crit: config.AttackCritConfig{
			XMP: config.AttackWeaponCritConfig{Chance: 0, Multiplier: 1},
		},
	}
	svc, gs := newCombatService(t, cfg)
	gs.Players.Upsert(state.Player{
		ID:        "attacker",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        1000,
		Inventory: map[string]int{gameplay.ItemIDXMP(1): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "enemy",
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 0, Longitude: 0.00005},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Level:  4,
		Energy: 1000,
	})

	result, err := svc.Attack(AttackCmd{PlayerID: "attacker", PortalID: "enemy", WeaponType: "XMP", WeaponLevel: 1}, time.Now())
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}
	if !result.Payload.Counterattack {
		t.Fatalf("expected counterattack triggered")
	}
	if result.Payload.CounterattackDamage != 500 {
		t.Fatalf("expected counterattack damage 500, got %d", result.Payload.CounterattackDamage)
	}
	if result.PlayerDelta.XMDelta != -550 {
		t.Fatalf("expected xm delta -550, got %d", result.PlayerDelta.XMDelta)
	}
	if result.PlayerDelta.InventoryDelta[gameplay.ItemIDXMP(1)] != -1 {
		t.Fatalf("expected xmp inventory delta -1")
	}
}

func TestModsDestroyedOnCrit(t *testing.T) {
	cfg := testGameplayConfig()
	enabled := true
	cfg.Attack = config.AttackConfig{
		Crit: config.AttackCritConfig{
			US: config.AttackWeaponCritConfig{Chance: 1, Multiplier: 1},
		},
		ModDestroy: config.AttackModDestroyConfig{
			Enabled:          &enabled,
			RollMaxExclusive: 10,
			Stickiness: map[string]int{
				"COMMON": -1,
			},
		},
	}
	svc, gs := newCombatService(t, cfg)
	svc.setRandSource(rand.NewSource(3))
	gs.Players.Upsert(state.Player{
		ID:        "attacker",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        3000,
		Inventory: map[string]int{gameplay.ItemIDUS(8): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "enemy",
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 0, Longitude: 0.00001},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Mods: map[int]state.ModSlot{
			1: {Slot: 1, ModType: "SHIELD", Rarity: "COMMON"},
			2: {Slot: 2, ModType: "FORCE_AMP", Rarity: "COMMON"},
		},
		Level:  1,
		Energy: 1000,
	})

	result, err := svc.Attack(AttackCmd{PlayerID: "attacker", PortalID: "enemy", WeaponType: "US", WeaponLevel: 8}, time.Now())
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}
	if len(result.Payload.ModsDestroyed) == 0 {
		t.Fatalf("expected destroyed mods on guaranteed crit")
	}
	portalState, _ := gs.Portals.Get("enemy")
	if len(portalState.Mods) >= 2 {
		t.Fatalf("expected at least one mod removed")
	}
}

func TestChargeBonusClamp(t *testing.T) {
	svc, gs := newCombatService(t, testGameplayConfig())
	gs.Players.Upsert(state.Player{
		ID:        "attacker",
		Faction:   "RESISTANCE",
		Level:     8,
		XM:        5000,
		Inventory: map[string]int{gameplay.ItemIDXMP(8): 1},
		Position:  state.Position{Latitude: 0, Longitude: 0},
	})
	gs.Portals.Upsert(state.Portal{
		ID:       "enemy",
		Faction:  "ENLIGHTENED",
		Position: state.Position{Latitude: 0, Longitude: 0.00005},
		Resonators: map[int]state.ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Level:  1,
		Energy: 1000,
	})

	result, err := svc.Attack(AttackCmd{PlayerID: "attacker", PortalID: "enemy", WeaponType: "XMP", WeaponLevel: 8, ChargeBonus: 1.4}, time.Now())
	if err != nil {
		t.Fatalf("attack failed: %v", err)
	}
	if result.Payload.ChargeBonus != 0.2 {
		t.Fatalf("expected charge bonus clamped to 0.2, got %f", result.Payload.ChargeBonus)
	}
}
