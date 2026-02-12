package combat

import (
	"strings"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

type DestroyedMod struct {
	PortalID string `json:"portalId"`
	Slot     int    `json:"slot"`
	ModType  string `json:"modType"`
	Rarity   string `json:"rarity"`
}

func normalizeModType(raw string) string {
	modType := strings.ToUpper(strings.TrimSpace(raw))
	switch modType {
	case "PORTAL_SHIELD":
		return "SHIELD"
	case "FORCE":
		return "FORCE_AMP"
	case "LINK":
		return "LINK_AMP"
	case "HEATSINK":
		return "HEAT_SINK"
	case "MULTI":
		return "MULTI_HACK"
	case "AXA", "AEGIS":
		return "AEGIS"
	default:
		return modType
	}
}

func normalizeStickinessKey(raw string) string {
	key := strings.ToUpper(strings.TrimSpace(raw))
	switch key {
	case "C":
		return "COMMON"
	case "R":
		return "RARE"
	case "VR":
		return "VERY_RARE"
	case "AXA", "AEGIS":
		return "AEGIS"
	default:
		return key
	}
}

func modStickiness(mod state.ModSlot, cfg modDestroyConfig) int {
	key := normalizeStickinessKey(mod.Rarity)
	if normalizeModType(mod.ModType) == "AEGIS" {
		key = "AEGIS"
	}
	if value, ok := cfg.stickiness[key]; ok {
		return value
	}
	return 0
}

func destroyModsOnCrit(portal *state.Portal, cfg modDestroyConfig, rollIntn func(int) int) []DestroyedMod {
	if portal == nil || len(portal.Mods) == 0 || !cfg.enabled || cfg.rollMaxExclusive <= 1 {
		return nil
	}
	destroyed := make([]DestroyedMod, 0)
	for slot, mod := range portal.Mods {
		roll := rollIntn(cfg.rollMaxExclusive)
		if roll <= modStickiness(mod, cfg) {
			continue
		}
		destroyed = append(destroyed, DestroyedMod{
			PortalID: portal.ID,
			Slot:     slot,
			ModType:  mod.ModType,
			Rarity:   mod.Rarity,
		})
		delete(portal.Mods, slot)
	}
	return destroyed
}
