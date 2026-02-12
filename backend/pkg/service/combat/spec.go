package combat

import (
	"math"
	"sort"
	"strings"

	"github.com/Twelveeee/openGress/backend/pkg/config"
)

type WeaponSpec struct {
	MaxDamage float64
	RadiusM   float64
	CostXM    int
}

type falloffTier struct {
	maxDistanceRatio float64
	multiplier       float64
}

type mitigationConfig struct {
	shieldValues  map[string]float64
	linkScale     float64
	linkDivisor   float64
	fieldPerField float64
	maxPercent    float64
}

type critConfig struct {
	chance     float64
	multiplier float64
}

type counterattackConfig struct {
	baseTriggerChance   float64
	maxTriggerChance    float64
	turretTriggerBonus  float64
	baseCritChance      float64
	turretCritBonus     float64
	critMultiplier      float64
	forceAmpBaseBonus   float64
	forceAmpExtraBonus  float64
	damageByPortalLevel map[int]int
}

type modDestroyConfig struct {
	enabled          bool
	rollMaxExclusive int
	stickiness       map[string]int
}

type runtimeAttackConfig struct {
	weaponSpecs        map[string]map[int]WeaponSpec
	falloffTiers       []falloffTier
	mitigation         mitigationConfig
	critByWeapon       map[string]critConfig
	counterattack      counterattackConfig
	modDestroy         modDestroyConfig
	destroyResonatorAP int
	keyDropChance      float64
}

func buildRuntimeAttackConfig(raw config.AttackConfig) runtimeAttackConfig {
	weaponSpecs := defaultWeaponSpecs()
	for weaponType, levels := range raw.WeaponSpecs {
		normalizedType := strings.ToUpper(strings.TrimSpace(weaponType))
		if normalizedType != "XMP" && normalizedType != "US" {
			continue
		}
		if _, ok := weaponSpecs[normalizedType]; !ok {
			weaponSpecs[normalizedType] = make(map[int]WeaponSpec)
		}
		for level, spec := range levels {
			if level < 1 || level > 8 {
				continue
			}
			if spec.Damage <= 0 || spec.RadiusM <= 0 || spec.CostXM <= 0 {
				continue
			}
			weaponSpecs[normalizedType][level] = WeaponSpec{
				MaxDamage: float64(spec.Damage),
				RadiusM:   spec.RadiusM,
				CostXM:    spec.CostXM,
			}
		}
	}

	falloff := defaultFalloffTiers()
	if len(raw.DistanceFalloffTiers) > 0 {
		custom := make([]falloffTier, 0, len(raw.DistanceFalloffTiers))
		for _, tier := range raw.DistanceFalloffTiers {
			if tier.MaxDistanceRatio <= 0 || tier.Multiplier < 0 {
				continue
			}
			custom = append(custom, falloffTier{
				maxDistanceRatio: tier.MaxDistanceRatio,
				multiplier:       tier.Multiplier,
			})
		}
		if len(custom) > 0 {
			sort.Slice(custom, func(i, j int) bool {
				return custom[i].maxDistanceRatio < custom[j].maxDistanceRatio
			})
			falloff = custom
		}
	}

	mitigation := defaultMitigationConfig()
	for key, value := range raw.Mitigation.ShieldValues {
		if value < 0 {
			continue
		}
		mitigation.shieldValues[normalizeStickinessKey(key)] = value
	}
	if raw.Mitigation.LinkScale > 0 {
		mitigation.linkScale = raw.Mitigation.LinkScale
	}
	if raw.Mitigation.LinkDivisor > 0 {
		mitigation.linkDivisor = raw.Mitigation.LinkDivisor
	}
	if raw.Mitigation.FieldPerField >= 0 {
		mitigation.fieldPerField = raw.Mitigation.FieldPerField
	}
	if raw.Mitigation.MaxPercent > 0 {
		mitigation.maxPercent = raw.Mitigation.MaxPercent
	}
	if mitigation.maxPercent > 99 {
		mitigation.maxPercent = 99
	}

	critByWeapon := defaultCritConfig()
	overrideCrit(&critByWeapon, "XMP", raw.Crit.XMP)
	overrideCrit(&critByWeapon, "US", raw.Crit.US)

	counter := defaultCounterattackConfig()
	if raw.Counterattack.BaseTriggerChance > 0 {
		counter.baseTriggerChance = raw.Counterattack.BaseTriggerChance
	}
	if raw.Counterattack.MaxTriggerChance > 0 {
		counter.maxTriggerChance = raw.Counterattack.MaxTriggerChance
	}
	if raw.Counterattack.TurretTriggerBonus > 0 {
		counter.turretTriggerBonus = raw.Counterattack.TurretTriggerBonus
	}
	if raw.Counterattack.BaseCritChance > 0 {
		counter.baseCritChance = raw.Counterattack.BaseCritChance
	}
	if raw.Counterattack.TurretCritBonus > 0 {
		counter.turretCritBonus = raw.Counterattack.TurretCritBonus
	}
	if raw.Counterattack.CritMultiplier > 0 {
		counter.critMultiplier = raw.Counterattack.CritMultiplier
	}
	if raw.Counterattack.ForceAmpBaseBonus >= 0 {
		counter.forceAmpBaseBonus = raw.Counterattack.ForceAmpBaseBonus
	}
	if raw.Counterattack.ForceAmpExtraBonus >= 0 {
		counter.forceAmpExtraBonus = raw.Counterattack.ForceAmpExtraBonus
	}
	for level, damage := range raw.Counterattack.DamageByPortalLevel {
		if level < 1 || level > 8 || damage <= 0 {
			continue
		}
		counter.damageByPortalLevel[level] = damage
	}

	modDestroy := defaultModDestroyConfig()
	if raw.ModDestroy.Enabled != nil {
		modDestroy.enabled = *raw.ModDestroy.Enabled
	}
	if raw.ModDestroy.RollMaxExclusive > 0 {
		modDestroy.rollMaxExclusive = raw.ModDestroy.RollMaxExclusive
	}
	for key, value := range raw.ModDestroy.Stickiness {
		if value < 0 {
			continue
		}
		modDestroy.stickiness[normalizeStickinessKey(key)] = value
	}
	destroyResonatorAP := 75
	if raw.DestroyResonatorAP != nil {
		destroyResonatorAP = *raw.DestroyResonatorAP
		if destroyResonatorAP < 0 {
			destroyResonatorAP = 0
		}
	}

	keyDropChance := 0.2
	if raw.KeyDropChance != nil {
		keyDropChance = clamp01(*raw.KeyDropChance)
	}

	return runtimeAttackConfig{
		weaponSpecs:        weaponSpecs,
		falloffTiers:       falloff,
		mitigation:         mitigation,
		critByWeapon:       critByWeapon,
		counterattack:      counter,
		modDestroy:         modDestroy,
		destroyResonatorAP: destroyResonatorAP,
		keyDropChance:      keyDropChance,
	}
}

func defaultWeaponSpecs() map[string]map[int]WeaponSpec {
	return map[string]map[int]WeaponSpec{
		"XMP": {
			1: {MaxDamage: 150, RadiusM: 42, CostXM: 50},
			2: {MaxDamage: 300, RadiusM: 48, CostXM: 100},
			3: {MaxDamage: 500, RadiusM: 58, CostXM: 150},
			4: {MaxDamage: 900, RadiusM: 72, CostXM: 200},
			5: {MaxDamage: 1200, RadiusM: 90, CostXM: 250},
			6: {MaxDamage: 1500, RadiusM: 112, CostXM: 300},
			7: {MaxDamage: 1800, RadiusM: 138, CostXM: 350},
			8: {MaxDamage: 2700, RadiusM: 168, CostXM: 400},
		},
		"US": {
			1: {MaxDamage: 300, RadiusM: 10, CostXM: 50},
			2: {MaxDamage: 600, RadiusM: 13, CostXM: 100},
			3: {MaxDamage: 1000, RadiusM: 16, CostXM: 150},
			4: {MaxDamage: 1800, RadiusM: 18, CostXM: 200},
			5: {MaxDamage: 2400, RadiusM: 21, CostXM: 250},
			6: {MaxDamage: 3000, RadiusM: 24, CostXM: 300},
			7: {MaxDamage: 3600, RadiusM: 27, CostXM: 350},
			8: {MaxDamage: 5400, RadiusM: 30, CostXM: 400},
		},
	}
}

func defaultFalloffTiers() []falloffTier {
	return []falloffTier{
		{maxDistanceRatio: 0.2, multiplier: 1.0},
		{maxDistanceRatio: 0.4, multiplier: 0.85},
		{maxDistanceRatio: 0.6, multiplier: 0.65},
		{maxDistanceRatio: 0.8, multiplier: 0.45},
		{maxDistanceRatio: 1.0, multiplier: 0.25},
	}
}

func defaultMitigationConfig() mitigationConfig {
	return mitigationConfig{
		shieldValues: map[string]float64{
			"COMMON":    30,
			"RARE":      40,
			"VERY_RARE": 60,
			"AEGIS":     70,
		},
		linkScale:     400.0 / 9.0,
		linkDivisor:   math.E,
		fieldPerField: 0,
		maxPercent:    95,
	}
}

func defaultCritConfig() map[string]critConfig {
	return map[string]critConfig{
		"XMP": {chance: 0.08, multiplier: 1.5},
		"US":  {chance: 0.35, multiplier: 1.8},
	}
}

func overrideCrit(dst *map[string]critConfig, weaponType string, raw config.AttackWeaponCritConfig) {
	current := (*dst)[weaponType]
	if raw.Chance > 0 {
		current.chance = clamp01(raw.Chance)
	}
	if raw.Multiplier > 0 {
		current.multiplier = raw.Multiplier
	}
	(*dst)[weaponType] = current
}

func defaultCounterattackConfig() counterattackConfig {
	return counterattackConfig{
		baseTriggerChance:  0.15,
		maxTriggerChance:   0.95,
		turretTriggerBonus: 0.25,
		baseCritChance:     0.05,
		turretCritBonus:    0.20,
		critMultiplier:     1.5,
		forceAmpBaseBonus:  1.0,
		forceAmpExtraBonus: 0.25,
		damageByPortalLevel: map[int]int{
			1: 75,
			2: 150,
			3: 300,
			4: 500,
			5: 750,
			6: 1125,
			7: 1625,
			8: 2500,
		},
	}
}

func defaultModDestroyConfig() modDestroyConfig {
	return modDestroyConfig{
		enabled:          true,
		rollMaxExclusive: 650000,
		stickiness: map[string]int{
			"COMMON":    0,
			"RARE":      150000,
			"VERY_RARE": 450000,
			"AEGIS":     550000,
		},
	}
}

func clamp01(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
