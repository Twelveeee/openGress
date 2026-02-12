package combat

import (
	"math"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func buildLinkCounts(links []state.Link) map[string]int {
	counts := make(map[string]int, len(links))
	for _, linkState := range links {
		counts[linkState.FromPortalID]++
		counts[linkState.ToPortalID]++
	}
	return counts
}

func buildFieldCounts(fields []state.Field) map[string]int {
	counts := make(map[string]int, len(fields))
	for _, fieldState := range fields {
		counts[fieldState.PortalIDs[0]]++
		counts[fieldState.PortalIDs[1]]++
		counts[fieldState.PortalIDs[2]]++
	}
	return counts
}

func calculateMitigationPercent(portal state.Portal, linkCount, fieldCount int, cfg mitigationConfig) float64 {
	shield := shieldMitigationPercent(portal, cfg)
	link := linkMitigationPercent(linkCount, cfg)
	field := cfg.fieldPerField * float64(fieldCount)
	total := shield + link + field
	if total < 0 {
		return 0
	}
	if total > cfg.maxPercent {
		return cfg.maxPercent
	}
	return total
}

func shieldMitigationPercent(portal state.Portal, cfg mitigationConfig) float64 {
	if len(portal.Mods) == 0 {
		return 0
	}
	total := 0.0
	for _, mod := range portal.Mods {
		modType := normalizeModType(mod.ModType)
		if modType != "SHIELD" && modType != "AEGIS" {
			continue
		}
		key := normalizeStickinessKey(mod.Rarity)
		if modType == "AEGIS" {
			key = "AEGIS"
		}
		if value, ok := cfg.shieldValues[key]; ok {
			total += value
		}
	}
	return total
}

func linkMitigationPercent(linkCount int, cfg mitigationConfig) float64 {
	if linkCount <= 0 {
		return 0
	}
	divisor := cfg.linkDivisor
	if divisor <= 0 {
		divisor = math.E
	}
	return cfg.linkScale * math.Atan(float64(linkCount)/divisor)
}
