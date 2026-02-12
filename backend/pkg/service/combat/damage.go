package combat

import "math"

func computeDistanceMultiplier(distanceMeters, radiusMeters float64, tiers []falloffTier) float64 {
	if radiusMeters <= 0 || distanceMeters < 0 {
		return 0
	}
	ratio := distanceMeters / radiusMeters
	if ratio > 1 {
		return 0
	}
	for _, tier := range tiers {
		if ratio <= tier.maxDistanceRatio {
			return tier.multiplier
		}
	}
	return 0
}

func computeRawDamage(maxDamage, distanceMeters, radiusMeters, chargeBonus float64, tiers []falloffTier) float64 {
	multiplier := computeDistanceMultiplier(distanceMeters, radiusMeters, tiers)
	if multiplier <= 0 {
		return 0
	}
	return maxDamage * (1 + chargeBonus) * multiplier
}

func applyMitigation(rawDamage, mitigationPercent float64) int {
	if rawDamage <= 0 {
		return 0
	}
	afterMitigation := rawDamage * (1 - mitigationPercent/100.0)
	if afterMitigation <= 0 {
		return 0
	}
	return int(math.Floor(afterMitigation))
}
