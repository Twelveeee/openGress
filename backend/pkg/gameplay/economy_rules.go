package gameplay

import (
	"strings"

	"github.com/Twelveeee/openGress/backend/pkg/config"
)

const (
	defaultDeployResonatorBase     = 50
	defaultDeployResonatorPerLevel = 50
	defaultAttackXMPBase           = 50
	defaultAttackXMPPerLevel       = 50
	defaultAttackUSBase            = 50
	defaultAttackUSPerLevel        = 50
	defaultDeployModCommon         = 400
	defaultDeployModRare           = 800
	defaultDeployModVeryRare       = 1000
	defaultLinkCost                = 250
	defaultChargePerXM             = 1
)

const (
	defaultAPCapturePortal    = 500
	defaultAPDeployResonator  = 125
	defaultAPUpgradeResonator = 65
	defaultAPDeployMod        = 150
	defaultAPDestroyResonator = 75
	defaultAPCreateLink       = 313
	defaultAPCreateField      = 1250
	defaultAPHackPortal       = 50
)

var defaultDeployModByRarity = map[string]int{
	"COMMON":    defaultDeployModCommon,
	"RARE":      defaultDeployModRare,
	"VERY_RARE": defaultDeployModVeryRare,
}

var defaultDeployModByType = map[string]int{
	"FORCE_AMP":               defaultDeployModRare,
	"TURRET":                  defaultDeployModRare,
	"LINK_AMP":                defaultDeployModRare,
	"AEGIS_SHIELD":            defaultDeployModVeryRare,
	"ITO_EN_TRANSMUTER_PLUS":  defaultDeployModVeryRare,
	"ITO_EN_TRANSMUTER_MINUS": defaultDeployModVeryRare,
}

func normalizeEconomyKey(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func normalizeModRarityKey(raw string) string {
	switch normalizeEconomyKey(raw) {
	case "C":
		return "COMMON"
	case "R":
		return "RARE"
	case "VR":
		return "VERY_RARE"
	default:
		return normalizeEconomyKey(raw)
	}
}

func mapValueByNormalizedKey(source map[string]int, key string) (int, bool) {
	if len(source) == 0 || strings.TrimSpace(key) == "" {
		return 0, false
	}
	normalized := normalizeEconomyKey(key)
	if v, ok := source[normalized]; ok && v > 0 {
		return v, true
	}
	for rawKey, value := range source {
		if value <= 0 {
			continue
		}
		if normalizeEconomyKey(rawKey) == normalized {
			return value, true
		}
	}
	return 0, false
}

func positiveOrDefault(value, defaultValue int) int {
	if value > 0 {
		return value
	}
	return defaultValue
}

func APRewardCapturePortal(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.APReward.CapturePortal, defaultAPCapturePortal)
}

func APRewardDeployResonator(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.APReward.DeployResonator, defaultAPDeployResonator)
}

func APRewardUpgradeResonator(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.APReward.UpgradeResonator, defaultAPUpgradeResonator)
}

func APRewardDeployMod(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.APReward.DeployMod, defaultAPDeployMod)
}

func APRewardDestroyResonatorUnit(cfg config.GameplayConfig) int {
	if cfg.APReward.DestroyResonator > 0 {
		return cfg.APReward.DestroyResonator
	}
	if cfg.Attack.DestroyResonatorAP != nil && *cfg.Attack.DestroyResonatorAP > 0 {
		return *cfg.Attack.DestroyResonatorAP
	}
	return defaultAPDestroyResonator
}

func APRewardDestroyResonator(cfg config.GameplayConfig, count int) int {
	if count <= 0 {
		return 0
	}
	return APRewardDestroyResonatorUnit(cfg) * count
}

func APRewardCreateLink(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.APReward.CreateLink, defaultAPCreateLink)
}

func APRewardCreateField(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.APReward.CreateField, defaultAPCreateField)
}

func APRewardHackPortal(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.APReward.HackPortal, defaultAPHackPortal)
}

func resolveDeployResonatorBase(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.XMCost.DeployResonatorBase, defaultDeployResonatorBase)
}

func resolveDeployResonatorPerLevel(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.XMCost.DeployResonatorPerLevel, defaultDeployResonatorPerLevel)
}

func resolveAttackXMPBase(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.XMCost.AttackXMPBase, defaultAttackXMPBase)
}

func resolveAttackXMPPerLevel(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.XMCost.AttackXMPPerLevel, defaultAttackXMPPerLevel)
}

func resolveAttackUSBase(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.XMCost.AttackUSBase, defaultAttackUSBase)
}

func resolveAttackUSPerLevel(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.XMCost.AttackUSPerLevel, defaultAttackUSPerLevel)
}

func resolveDeployModCost(cfg config.GameplayConfig, modType, rarity string) int {
	if value, ok := mapValueByNormalizedKey(cfg.XMCost.DeployModByType, modType); ok {
		return value
	}
	if value, ok := mapValueByNormalizedKey(defaultDeployModByType, modType); ok {
		return value
	}
	if value, ok := mapValueByNormalizedKey(cfg.XMCost.DeployModByRarity, normalizeModRarityKey(rarity)); ok {
		return value
	}
	if value, ok := mapValueByNormalizedKey(defaultDeployModByRarity, normalizeModRarityKey(rarity)); ok {
		return value
	}
	if cfg.XMCost.DeployModFlat > 0 {
		return cfg.XMCost.DeployModFlat
	}
	return defaultDeployModCommon
}

func resolveLinkCost(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.XMCost.LinkFlat, defaultLinkCost)
}

func resolveChargePerXM(cfg config.GameplayConfig) int {
	return positiveOrDefault(cfg.XMCost.ChargePerXM, defaultChargePerXM)
}
