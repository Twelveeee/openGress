package combat

import (
	"math"

	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

type counterattackResult struct {
	triggered bool
	damage    int
	critical  bool
}

func calculateCounterattack(target state.Portal, cfg counterattackConfig, rollChance func(float64) bool) counterattackResult {
	if target.Faction == "" || target.Faction == "NEUTRAL" || len(target.Resonators) == 0 {
		return counterattackResult{}
	}
	turretCount, forceAmpCount := countAttackMods(target.Mods)
	triggerChance := cfg.baseTriggerChance + float64(turretCount)*cfg.turretTriggerBonus
	triggerChance = clamp01(math.Min(triggerChance, cfg.maxTriggerChance))
	if !rollChance(triggerChance) {
		return counterattackResult{}
	}

	level := gameplay.NormalizeLevel(target.Level)
	if level < 1 {
		level = 1
	}
	baseDamage, ok := cfg.damageByPortalLevel[level]
	if !ok || baseDamage <= 0 {
		baseDamage = cfg.damageByPortalLevel[1]
	}
	damageMultiplier := 1.0
	if forceAmpCount > 0 {
		damageMultiplier += cfg.forceAmpBaseBonus + float64(forceAmpCount-1)*cfg.forceAmpExtraBonus
	}
	damage := int(math.Floor(float64(baseDamage) * damageMultiplier))
	if damage <= 0 {
		return counterattackResult{}
	}

	critChance := clamp01(cfg.baseCritChance + float64(turretCount)*cfg.turretCritBonus)
	critical := rollChance(critChance)
	if critical {
		damage = int(math.Floor(float64(damage) * cfg.critMultiplier))
	}
	if damage <= 0 {
		return counterattackResult{}
	}
	return counterattackResult{
		triggered: true,
		damage:    damage,
		critical:  critical,
	}
}

func countAttackMods(mods map[int]state.ModSlot) (turretCount int, forceAmpCount int) {
	for _, mod := range mods {
		switch normalizeModType(mod.ModType) {
		case "TURRET":
			turretCount++
		case "FORCE_AMP":
			forceAmpCount++
		}
	}
	return turretCount, forceAmpCount
}
