package config

import (
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/Twelveeee/golib/logger"
	"gopkg.in/yaml.v3"
)

type Config struct {
	// Server/Database/Auth/Log 为核心配置块。
	Server    ServerConfig      `yaml:"server"`
	Database  DatabaseConfig    `yaml:"database"`
	Auth      AuthConfig        `yaml:"auth"`
	Admin     AdminConfig       `yaml:"admin"`
	Gameplay  GameplayConfig    `yaml:"gameplay"`
	Persist   PersistenceConfig `yaml:"persistence"`
	LogConfig logger.Config     `yaml:"log"`
}

type ServerConfig struct {
	HTTPPort int             `yaml:"http_port"`
	WSPort   int             `yaml:"ws_port"`
	WS       WSRuntimeConfig `yaml:"ws"`
}

type WSRuntimeConfig struct {
	MovementTickMS      int `yaml:"movement_tick_ms"`
	MapTickMS           int `yaml:"map_tick_ms"`
	PlayerStatePushMS   int `yaml:"player_state_push_ms"`
	NearbyPlayersPushMS int `yaml:"nearby_players_push_ms"`
}

const (
	DefaultWSMovementTickMS      = 100
	DefaultWSMapTickMS           = 1000
	DefaultWSPlayerStatePushMS   = 300
	DefaultWSNearbyPlayersPushMS = 600
	MinWSRuntimeIntervalMS       = 20
)

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type AuthConfig struct {
	// JWTSecret 用于签发 access_token。
	JWTSecret        string `yaml:"jwt_secret"`
	AccessTTLSeconds int    `yaml:"access_ttl_seconds"`
}

type AdminConfig struct {
	// AdminToken 用于管理接口鉴权。
	Token string `yaml:"token"`
}

type GameplayConfig struct {
	// 地图边界（经纬度矩形）。
	MapBounds      MapBoundsConfig `yaml:"map_bounds"`
	ViewRadiusM    float64         `yaml:"view_radius_m"`
	InteractRangeM float64         `yaml:"interact_range_m"`
	LogCapacity    int             `yaml:"log_capacity"`
	XMCost         XMCostConfig    `yaml:"xm_cost"`
	APReward       APRewardConfig  `yaml:"ap_reward"`
	Attack         AttackConfig    `yaml:"attack"`
}

type MapBoundsConfig struct {
	MinLat float64 `yaml:"min_lat"`
	MaxLat float64 `yaml:"max_lat"`
	MinLon float64 `yaml:"min_lon"`
	MaxLon float64 `yaml:"max_lon"`
}

type XMCostConfig struct {
	DeployResonatorBase     int            `yaml:"deploy_resonator_base"`
	DeployResonatorPerLevel int            `yaml:"deploy_resonator_per_level"`
	AttackXMPBase           int            `yaml:"attack_xmp_base"`
	AttackXMPPerLevel       int            `yaml:"attack_xmp_per_level"`
	AttackUSBase            int            `yaml:"attack_us_base"`
	AttackUSPerLevel        int            `yaml:"attack_us_per_level"`
	DeployModFlat           int            `yaml:"deploy_mod_flat"`
	DeployModByRarity       map[string]int `yaml:"deploy_mod_by_rarity"`
	DeployModByType         map[string]int `yaml:"deploy_mod_by_type"`
	LinkFlat                int            `yaml:"link_flat"`
	ChargePerXM             int            `yaml:"charge_per_xm"`
}

type APRewardConfig struct {
	CapturePortal    int `yaml:"capture_portal"`
	DeployResonator  int `yaml:"deploy_resonator"`
	UpgradeResonator int `yaml:"upgrade_resonator"`
	DeployMod        int `yaml:"deploy_mod"`
	DestroyResonator int `yaml:"destroy_resonator"`
	CreateLink       int `yaml:"create_link"`
	CreateField      int `yaml:"create_field"`
	HackPortal       int `yaml:"hack_portal"`
}

type AttackConfig struct {
	WeaponSpecs          map[string]map[int]AttackWeaponSpec `yaml:"weapon_specs"`
	DistanceFalloffTiers []AttackFalloffTier                 `yaml:"distance_falloff_tiers"`
	Mitigation           AttackMitigationConfig              `yaml:"mitigation"`
	Crit                 AttackCritConfig                    `yaml:"crit"`
	Counterattack        AttackCounterattackConfig           `yaml:"counterattack"`
	ModDestroy           AttackModDestroyConfig              `yaml:"mod_destroy"`
	DestroyResonatorAP   *int                                `yaml:"destroy_resonator_ap"`
	KeyDropChance        *float64                            `yaml:"key_drop_chance"`
}

type AttackWeaponSpec struct {
	Damage  int     `yaml:"damage"`
	RadiusM float64 `yaml:"radius_m"`
	CostXM  int     `yaml:"cost_xm"`
}

type AttackFalloffTier struct {
	MaxDistanceRatio float64 `yaml:"max_distance_ratio"`
	Multiplier       float64 `yaml:"multiplier"`
}

type AttackMitigationConfig struct {
	ShieldValues  map[string]float64 `yaml:"shield_values"`
	LinkScale     float64            `yaml:"link_scale"`
	LinkDivisor   float64            `yaml:"link_divisor"`
	FieldPerField float64            `yaml:"field_per_field"`
	MaxPercent    float64            `yaml:"max_percent"`
}

type AttackWeaponCritConfig struct {
	Chance     float64 `yaml:"chance"`
	Multiplier float64 `yaml:"multiplier"`
}

type AttackCritConfig struct {
	XMP AttackWeaponCritConfig `yaml:"xmp"`
	US  AttackWeaponCritConfig `yaml:"us"`
}

type AttackCounterattackConfig struct {
	BaseTriggerChance   float64     `yaml:"base_trigger_chance"`
	MaxTriggerChance    float64     `yaml:"max_trigger_chance"`
	TurretTriggerBonus  float64     `yaml:"turret_trigger_bonus"`
	BaseCritChance      float64     `yaml:"base_crit_chance"`
	TurretCritBonus     float64     `yaml:"turret_crit_bonus"`
	CritMultiplier      float64     `yaml:"crit_multiplier"`
	ForceAmpBaseBonus   float64     `yaml:"force_amp_base_bonus"`
	ForceAmpExtraBonus  float64     `yaml:"force_amp_extra_bonus"`
	DamageByPortalLevel map[int]int `yaml:"damage_by_portal_level"`
}

type AttackModDestroyConfig struct {
	Enabled          *bool          `yaml:"enabled"`
	RollMaxExclusive int            `yaml:"roll_max_exclusive"`
	Stickiness       map[string]int `yaml:"stickiness"`
}

type PersistenceConfig struct {
	Enabled                     bool `yaml:"enabled"`
	FlushIntervalSeconds        int  `yaml:"flush_interval_seconds"`
	ShutdownFlushTimeoutSeconds int  `yaml:"shutdown_flush_timeout_seconds"`
}

func DefaultWSRuntimeConfig() WSRuntimeConfig {
	return WSRuntimeConfig{
		MovementTickMS:      DefaultWSMovementTickMS,
		MapTickMS:           DefaultWSMapTickMS,
		PlayerStatePushMS:   DefaultWSPlayerStatePushMS,
		NearbyPlayersPushMS: DefaultWSNearbyPlayersPushMS,
	}
}

func (cfg WSRuntimeConfig) Normalize() WSRuntimeConfig {
	cfg.MovementTickMS = normalizeRuntimeInterval("server.ws.movement_tick_ms", cfg.MovementTickMS, DefaultWSMovementTickMS)
	cfg.MapTickMS = normalizeRuntimeInterval("server.ws.map_tick_ms", cfg.MapTickMS, DefaultWSMapTickMS)
	cfg.PlayerStatePushMS = normalizeRuntimeInterval("server.ws.player_state_push_ms", cfg.PlayerStatePushMS, DefaultWSPlayerStatePushMS)
	cfg.NearbyPlayersPushMS = normalizeRuntimeInterval("server.ws.nearby_players_push_ms", cfg.NearbyPlayersPushMS, DefaultWSNearbyPlayersPushMS)
	return cfg
}

func LoadConfig(path string) (*Config, error) {
	// 读取 YAML 配置并应用环境变量覆盖。
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	applyEnvOverrides(&cfg)
	cfg.Server.WS = cfg.Server.WS.Normalize()
	return &cfg, nil
}

func SaveConfig(path string, cfg *Config) error {
	// 按 YAML 格式写回配置文件。
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func applyEnvOverrides(cfg *Config) {
	// 用环境变量覆盖配置（方便容器/部署环境）。
	if v, ok := os.LookupEnv("OPENGRESS_HTTP_PORT"); ok {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.HTTPPort = port
		}
	}
	if v, ok := os.LookupEnv("OPENGRESS_WS_PORT"); ok {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Server.WSPort = port
		}
	}
	if v, ok := os.LookupEnv("OPENGRESS_WS_MOVEMENT_TICK_MS"); ok {
		cfg.Server.WS.MovementTickMS = parseEnvIntervalOrDefault("OPENGRESS_WS_MOVEMENT_TICK_MS", v, DefaultWSMovementTickMS)
	}
	if v, ok := os.LookupEnv("OPENGRESS_WS_MAP_TICK_MS"); ok {
		cfg.Server.WS.MapTickMS = parseEnvIntervalOrDefault("OPENGRESS_WS_MAP_TICK_MS", v, DefaultWSMapTickMS)
	}
	if v, ok := os.LookupEnv("OPENGRESS_WS_PLAYER_STATE_PUSH_MS"); ok {
		cfg.Server.WS.PlayerStatePushMS = parseEnvIntervalOrDefault("OPENGRESS_WS_PLAYER_STATE_PUSH_MS", v, DefaultWSPlayerStatePushMS)
	}
	if v, ok := os.LookupEnv("OPENGRESS_WS_NEARBY_PLAYERS_PUSH_MS"); ok {
		cfg.Server.WS.NearbyPlayersPushMS = parseEnvIntervalOrDefault("OPENGRESS_WS_NEARBY_PLAYERS_PUSH_MS", v, DefaultWSNearbyPlayersPushMS)
	}
	if v, ok := os.LookupEnv("OPENGRESS_LOG_FILE"); ok && v != "" {
		cfg.LogConfig.FileName = v
	}
	if v, ok := os.LookupEnv("OPENGRESS_LOG_LEVEL"); ok && v != "" {
		if level, ok := parseLogLevel(v); ok {
			cfg.LogConfig.Level = level
		}
	}
	if v, ok := os.LookupEnv("OPENGRESS_DB_HOST"); ok && v != "" {
		cfg.Database.Host = v
	}
	if v, ok := os.LookupEnv("OPENGRESS_DB_PORT"); ok {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = port
		}
	}
	if v, ok := os.LookupEnv("OPENGRESS_DB_USER"); ok && v != "" {
		cfg.Database.User = v
	}
	if v, ok := os.LookupEnv("OPENGRESS_DB_PASSWORD"); ok && v != "" {
		cfg.Database.Password = v
	}
	if v, ok := os.LookupEnv("OPENGRESS_DB_NAME"); ok && v != "" {
		cfg.Database.DBName = v
	}
	if v, ok := os.LookupEnv("OPENGRESS_DB_SSLMODE"); ok && v != "" {
		cfg.Database.SSLMode = v
	}
	if v, ok := os.LookupEnv("OPENGRESS_JWT_SECRET"); ok && v != "" {
		cfg.Auth.JWTSecret = v
	}
	if v, ok := os.LookupEnv("OPENGRESS_JWT_TTL"); ok {
		if seconds, err := strconv.Atoi(v); err == nil {
			cfg.Auth.AccessTTLSeconds = seconds
		}
	}
	if v, ok := os.LookupEnv("OPENGRESS_ADMIN_TOKEN"); ok && v != "" {
		cfg.Admin.Token = v
	}
	if v, ok := os.LookupEnv("OPENGRESS_PERSISTENCE_ENABLED"); ok {
		if enabled, err := strconv.ParseBool(v); err == nil {
			cfg.Persist.Enabled = enabled
		}
	}
	if v, ok := os.LookupEnv("OPENGRESS_PERSISTENCE_FLUSH_INTERVAL_SECONDS"); ok {
		if seconds, err := strconv.Atoi(v); err == nil {
			cfg.Persist.FlushIntervalSeconds = seconds
		}
	}
	if v, ok := os.LookupEnv("OPENGRESS_PERSISTENCE_SHUTDOWN_FLUSH_TIMEOUT_SECONDS"); ok {
		if seconds, err := strconv.Atoi(v); err == nil {
			cfg.Persist.ShutdownFlushTimeoutSeconds = seconds
		}
	}
}

func parseEnvIntervalOrDefault(envKey, raw string, defaultValue int) int {
	value, err := strconv.Atoi(raw)
	if err != nil {
		slog.Warn("invalid ws interval env, fallback to default", "env", envKey, "value", raw, "default", defaultValue)
		return defaultValue
	}
	if value < MinWSRuntimeIntervalMS {
		slog.Warn("ws interval env too small, fallback to default", "env", envKey, "value", value, "min", MinWSRuntimeIntervalMS, "default", defaultValue)
		return defaultValue
	}
	return value
}

func normalizeRuntimeInterval(field string, value int, defaultValue int) int {
	if value < MinWSRuntimeIntervalMS {
		slog.Warn("invalid ws runtime interval, fallback to default", "field", field, "value", value, "min", MinWSRuntimeIntervalMS, "default", defaultValue)
		return defaultValue
	}
	return value
}

func parseLogLevel(raw string) (slog.Level, bool) {
	// 解析日志级别字符串，返回 level 与是否匹配。
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case "DEBUG":
		return slog.LevelDebug, true
	case "INFO":
		return slog.LevelInfo, true
	case "WARN", "WARNING":
		return slog.LevelWarn, true
	case "ERROR":
		return slog.LevelError, true
	default:
		return slog.LevelInfo, false
	}
}
