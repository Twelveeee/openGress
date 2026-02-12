package gameplay

import (
	"testing"

	"github.com/Twelveeee/openGress/backend/pkg/config"
)

func TestXMCostDefaultsFromEconomyRules(t *testing.T) {
	cfg := config.GameplayConfig{}
	if got := DeployResonatorCost(cfg, 3); got != 150 {
		t.Fatalf("expected deploy resonator cost 150, got %d", got)
	}
	if got := AttackCost(cfg, "XMP", 4); got != 200 {
		t.Fatalf("expected attack xmp cost 200, got %d", got)
	}
	if got := AttackCost(cfg, "US", 4); got != 200 {
		t.Fatalf("expected attack us cost 200, got %d", got)
	}
	if got := LinkCost(cfg); got != 250 {
		t.Fatalf("expected link cost 250, got %d", got)
	}
	if got := ChargeCost(cfg, 120); got != 120 {
		t.Fatalf("expected charge cost 120, got %d", got)
	}
}

func TestDeployModCostWithPriority(t *testing.T) {
	cfg := config.GameplayConfig{
		XMCost: config.XMCostConfig{
			DeployModByType: map[string]int{
				"LINK_AMP": 900,
			},
			DeployModByRarity: map[string]int{
				"COMMON": 450,
			},
			DeployModFlat: 300,
		},
	}

	if got := DeployModCostWith(cfg, "LINK_AMP", "COMMON"); got != 900 {
		t.Fatalf("expected type override cost 900, got %d", got)
	}
	if got := DeployModCostWith(cfg, "SHIELD", "COMMON"); got != 450 {
		t.Fatalf("expected rarity cost 450, got %d", got)
	}
	if got := DeployModCostWith(cfg, "UNKNOWN", "UNKNOWN"); got != 300 {
		t.Fatalf("expected legacy flat fallback 300, got %d", got)
	}
}

func TestDeployModCostDefaultTable(t *testing.T) {
	cfg := config.GameplayConfig{}
	if got := DeployModCostWith(cfg, "FORCE_AMP", "COMMON"); got != 800 {
		t.Fatalf("expected FORCE_AMP default type cost 800, got %d", got)
	}
	if got := DeployModCostWith(cfg, "SHIELD", "VERY_RARE"); got != 1000 {
		t.Fatalf("expected very rare rarity cost 1000, got %d", got)
	}
	if got := DeployModCostWith(cfg, "SHIELD", "C"); got != 400 {
		t.Fatalf("expected common shorthand cost 400, got %d", got)
	}
}

func TestAPRewardDefaultsAndFallback(t *testing.T) {
	cfg := config.GameplayConfig{}
	if got := APRewardCapturePortal(cfg); got != 500 {
		t.Fatalf("expected capture ap 500, got %d", got)
	}
	if got := APRewardDeployResonator(cfg); got != 125 {
		t.Fatalf("expected deploy resonator ap 125, got %d", got)
	}
	if got := APRewardUpgradeResonator(cfg); got != 65 {
		t.Fatalf("expected upgrade resonator ap 65, got %d", got)
	}
	if got := APRewardDeployMod(cfg); got != 150 {
		t.Fatalf("expected deploy mod ap 150, got %d", got)
	}
	if got := APRewardCreateLink(cfg); got != 313 {
		t.Fatalf("expected create link ap 313, got %d", got)
	}
	if got := APRewardCreateField(cfg); got != 1250 {
		t.Fatalf("expected create field ap 1250, got %d", got)
	}
	if got := APRewardHackPortal(cfg); got != 50 {
		t.Fatalf("expected hack ap 50, got %d", got)
	}
	if got := APRewardDestroyResonator(cfg, 2); got != 150 {
		t.Fatalf("expected destroy resonator ap 150, got %d", got)
	}
}

func TestAPRewardOverride(t *testing.T) {
	destroyAP := 90
	cfg := config.GameplayConfig{
		APReward: config.APRewardConfig{
			DeployMod: 210,
		},
		Attack: config.AttackConfig{
			DestroyResonatorAP: &destroyAP,
		},
	}
	if got := APRewardDeployMod(cfg); got != 210 {
		t.Fatalf("expected deploy mod ap override 210, got %d", got)
	}
	if got := APRewardDestroyResonator(cfg, 2); got != 180 {
		t.Fatalf("expected destroy resonator fallback from attack config 180, got %d", got)
	}
}
