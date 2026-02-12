package gameplay

import "github.com/Twelveeee/openGress/backend/pkg/config"

// XM 消耗计算。
func DeployResonatorCost(cfg config.GameplayConfig, level int) int {
	level = NormalizeLevel(level)
	return resolveDeployResonatorBase(cfg) + resolveDeployResonatorPerLevel(cfg)*(level-1)
}

func DeployModCost(cfg config.GameplayConfig) int {
	return resolveDeployModCost(cfg, "", "")
}

func DeployModCostWith(cfg config.GameplayConfig, modType, rarity string) int {
	return resolveDeployModCost(cfg, modType, rarity)
}

func LinkCost(cfg config.GameplayConfig) int {
	return resolveLinkCost(cfg)
}

func ChargeCost(cfg config.GameplayConfig, amount int) int {
	if amount <= 0 {
		return 0
	}
	return resolveChargePerXM(cfg) * amount
}

func AttackCost(cfg config.GameplayConfig, weaponType string, level int) int {
	level = NormalizeLevel(level)
	switch weaponType {
	case "XMP":
		return resolveAttackXMPBase(cfg) + resolveAttackXMPPerLevel(cfg)*(level-1)
	case "US":
		return resolveAttackUSBase(cfg) + resolveAttackUSPerLevel(cfg)*(level-1)
	default:
		return 0
	}
}
