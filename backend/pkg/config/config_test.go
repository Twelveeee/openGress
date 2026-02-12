package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const baseConfigYAML = `
server:
  http_port: 8080
  ws_port: 8081
database:
  host: 127.0.0.1
  port: 5432
  user: test
  password: test
  dbname: test
  sslmode: disable
auth:
  jwt_secret: dev-secret
  access_ttl_seconds: 3600
admin:
  token: dev-admin
gameplay:
  map_bounds:
    min_lat: -90
    max_lat: 90
    min_lon: -180
    max_lon: 180
  view_radius_m: 400
  interact_range_m: 40
  log_capacity: 50
  xm_cost:
    deploy_resonator_base: 50
    deploy_resonator_per_level: 50
    attack_xmp_base: 50
    attack_xmp_per_level: 50
    attack_us_base: 50
    attack_us_per_level: 50
    deploy_mod_by_rarity:
      COMMON: 400
      RARE: 800
      VERY_RARE: 1000
    deploy_mod_by_type:
      FORCE_AMP: 800
    deploy_mod_flat: 400
    link_flat: 250
    charge_per_xm: 1
  ap_reward:
    capture_portal: 500
    deploy_resonator: 125
    upgrade_resonator: 65
    deploy_mod: 150
    destroy_resonator: 75
    create_link: 313
    create_field: 1250
    hack_portal: 50
`

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return path
}

func TestLoadConfigWSRuntimeDefaults(t *testing.T) {
	path := writeConfigFile(t, baseConfigYAML)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.WS.MovementTickMS != DefaultWSMovementTickMS {
		t.Fatalf("unexpected movement tick default: %d", cfg.Server.WS.MovementTickMS)
	}
	if cfg.Server.WS.MapTickMS != DefaultWSMapTickMS {
		t.Fatalf("unexpected map tick default: %d", cfg.Server.WS.MapTickMS)
	}
	if cfg.Server.WS.PlayerStatePushMS != DefaultWSPlayerStatePushMS {
		t.Fatalf("unexpected player state push default: %d", cfg.Server.WS.PlayerStatePushMS)
	}
	if cfg.Server.WS.NearbyPlayersPushMS != DefaultWSNearbyPlayersPushMS {
		t.Fatalf("unexpected nearby players push default: %d", cfg.Server.WS.NearbyPlayersPushMS)
	}
}

func TestLoadConfigWSRuntimeEnvOverrides(t *testing.T) {
	t.Setenv("OPENGRESS_WS_MOVEMENT_TICK_MS", "120")
	t.Setenv("OPENGRESS_WS_MAP_TICK_MS", "1400")
	t.Setenv("OPENGRESS_WS_PLAYER_STATE_PUSH_MS", "350")
	t.Setenv("OPENGRESS_WS_NEARBY_PLAYERS_PUSH_MS", "800")

	path := writeConfigFile(t, baseConfigYAML)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.WS.MovementTickMS != 120 {
		t.Fatalf("unexpected movement tick: %d", cfg.Server.WS.MovementTickMS)
	}
	if cfg.Server.WS.MapTickMS != 1400 {
		t.Fatalf("unexpected map tick: %d", cfg.Server.WS.MapTickMS)
	}
	if cfg.Server.WS.PlayerStatePushMS != 350 {
		t.Fatalf("unexpected player state push: %d", cfg.Server.WS.PlayerStatePushMS)
	}
	if cfg.Server.WS.NearbyPlayersPushMS != 800 {
		t.Fatalf("unexpected nearby players push: %d", cfg.Server.WS.NearbyPlayersPushMS)
	}
}

func TestLoadConfigWSRuntimeInvalidEnvFallback(t *testing.T) {
	t.Setenv("OPENGRESS_WS_MOVEMENT_TICK_MS", "oops")
	t.Setenv("OPENGRESS_WS_MAP_TICK_MS", "0")
	t.Setenv("OPENGRESS_WS_PLAYER_STATE_PUSH_MS", "19")
	t.Setenv("OPENGRESS_WS_NEARBY_PLAYERS_PUSH_MS", "-1")

	path := writeConfigFile(t, baseConfigYAML)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.WS.MovementTickMS != DefaultWSMovementTickMS {
		t.Fatalf("unexpected movement tick fallback: %d", cfg.Server.WS.MovementTickMS)
	}
	if cfg.Server.WS.MapTickMS != DefaultWSMapTickMS {
		t.Fatalf("unexpected map tick fallback: %d", cfg.Server.WS.MapTickMS)
	}
	if cfg.Server.WS.PlayerStatePushMS != DefaultWSPlayerStatePushMS {
		t.Fatalf("unexpected player state fallback: %d", cfg.Server.WS.PlayerStatePushMS)
	}
	if cfg.Server.WS.NearbyPlayersPushMS != DefaultWSNearbyPlayersPushMS {
		t.Fatalf("unexpected nearby players fallback: %d", cfg.Server.WS.NearbyPlayersPushMS)
	}
}

func TestLoadConfigWSRuntimeInvalidYAMLFallback(t *testing.T) {
	content := strings.Replace(baseConfigYAML, "  ws_port: 8081", `  ws_port: 8081
  ws:
    movement_tick_ms: 10
    map_tick_ms: 0
    player_state_push_ms: 25
    nearby_players_push_ms: -5`, 1)
	path := writeConfigFile(t, content)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Server.WS.MovementTickMS != DefaultWSMovementTickMS {
		t.Fatalf("unexpected movement tick yaml fallback: %d", cfg.Server.WS.MovementTickMS)
	}
	if cfg.Server.WS.MapTickMS != DefaultWSMapTickMS {
		t.Fatalf("unexpected map tick yaml fallback: %d", cfg.Server.WS.MapTickMS)
	}
	if cfg.Server.WS.PlayerStatePushMS != 25 {
		t.Fatalf("unexpected player state yaml value: %d", cfg.Server.WS.PlayerStatePushMS)
	}
	if cfg.Server.WS.NearbyPlayersPushMS != DefaultWSNearbyPlayersPushMS {
		t.Fatalf("unexpected nearby players yaml fallback: %d", cfg.Server.WS.NearbyPlayersPushMS)
	}
}

func TestLoadConfigGameplayEconomyFields(t *testing.T) {
	path := writeConfigFile(t, baseConfigYAML)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Gameplay.XMCost.LinkFlat != 250 {
		t.Fatalf("unexpected link flat cost: %d", cfg.Gameplay.XMCost.LinkFlat)
	}
	if got := cfg.Gameplay.XMCost.DeployModByRarity["COMMON"]; got != 400 {
		t.Fatalf("unexpected deploy mod rarity cost COMMON=%d", got)
	}
	if got := cfg.Gameplay.XMCost.DeployModByType["FORCE_AMP"]; got != 800 {
		t.Fatalf("unexpected deploy mod type cost FORCE_AMP=%d", got)
	}
	if cfg.Gameplay.APReward.DeployMod != 150 {
		t.Fatalf("unexpected ap reward deploy mod: %d", cfg.Gameplay.APReward.DeployMod)
	}
	if cfg.Gameplay.APReward.HackPortal != 50 {
		t.Fatalf("unexpected ap reward hack portal: %d", cfg.Gameplay.APReward.HackPortal)
	}
}

func TestLoadConfigAttackSpecs(t *testing.T) {
	content := strings.Replace(baseConfigYAML, "  xm_cost:", `  attack:
    weapon_specs:
      XMP:
        8: { damage: 2700, radius_m: 168, cost_xm: 400 }
      US:
        8: { damage: 5400, radius_m: 30, cost_xm: 400 }
    counterattack:
      base_trigger_chance: 0.2
      damage_by_portal_level:
        8: 2500
  xm_cost:`, 1)
	path := writeConfigFile(t, content)
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	xmp8 := cfg.Gameplay.Attack.WeaponSpecs["XMP"][8]
	if xmp8.Damage != 2700 || xmp8.RadiusM != 168 || xmp8.CostXM != 400 {
		t.Fatalf("unexpected xmp8 attack spec: %+v", xmp8)
	}
	if cfg.Gameplay.Attack.Counterattack.BaseTriggerChance != 0.2 {
		t.Fatalf("unexpected counterattack base trigger chance: %f", cfg.Gameplay.Attack.Counterattack.BaseTriggerChance)
	}
	if cfg.Gameplay.Attack.Counterattack.DamageByPortalLevel[8] != 2500 {
		t.Fatalf("unexpected counterattack level8 damage: %d", cfg.Gameplay.Attack.Counterattack.DamageByPortalLevel[8])
	}
}
