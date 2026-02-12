package gameplay

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	ItemPrefixResonator = "RESO_L"
	ItemPrefixXMP       = "XMP_L"
	ItemPrefixUS        = "US_L"
	ItemPrefixCube      = "CUBE_L"
	ItemPrefixMod       = "MOD:"
	ItemPrefixKey       = "KEY:"
	MaxKeyPerPortal     = 2
)

var cubeXMByLevel = map[int]int{
	1: 1000,
	2: 2500,
	3: 5000,
	4: 10000,
	5: 20000,
	6: 40000,
	7: 70000,
	8: 100000,
}

// ItemIDResonator 返回谐振器物品 ID。
func ItemIDResonator(level int) string {
	return fmt.Sprintf("%s%d", ItemPrefixResonator, level)
}

// ItemIDXMP 返回 XMP 物品 ID。
func ItemIDXMP(level int) string {
	return fmt.Sprintf("%s%d", ItemPrefixXMP, level)
}

// ItemIDUS 返回 Ultra Strike 物品 ID。
func ItemIDUS(level int) string {
	return fmt.Sprintf("%s%d", ItemPrefixUS, level)
}

// ItemIDWeapon 返回武器物品 ID。
func ItemIDWeapon(weaponType string, level int) string {
	switch strings.ToUpper(weaponType) {
	case "XMP":
		return ItemIDXMP(level)
	case "US":
		return ItemIDUS(level)
	default:
		return ""
	}
}

// ItemIDCube 返回能量块物品 ID。
func ItemIDCube(level int) string {
	return fmt.Sprintf("%s%d", ItemPrefixCube, level)
}

// ItemIDMod 返回 Mod 物品 ID。
func ItemIDMod(modType, rarity string) string {
	return fmt.Sprintf("%s%s:%s", ItemPrefixMod, strings.ToUpper(modType), strings.ToUpper(rarity))
}

// ItemIDKey 返回 Portal Key 物品 ID。
func ItemIDKey(portalID string) string {
	return ItemPrefixKey + portalID
}

// NormalizeCubeID 兼容旧 CUBE 写法。
func NormalizeCubeID(itemID string) string {
	if strings.EqualFold(strings.TrimSpace(itemID), "CUBE") {
		return ItemIDCube(1)
	}
	return strings.ToUpper(itemID)
}

// NormalizeInventoryItemID 规范化背包道具 ID（用于使用/回收等场景）。
// KEY 仅标准化前缀，保留 portalId 原始大小写，避免大小写变化导致 key 无法命中库存。
func NormalizeInventoryItemID(itemID string) string {
	itemID = strings.TrimSpace(itemID)
	if itemID == "" {
		return ""
	}
	if strings.EqualFold(itemID, "CUBE") {
		return ItemIDCube(1)
	}
	if len(itemID) >= len(ItemPrefixKey) && strings.EqualFold(itemID[:len(ItemPrefixKey)], ItemPrefixKey) {
		return ItemIDKey(itemID[len(ItemPrefixKey):])
	}
	return strings.ToUpper(itemID)
}

// ParseLevelItem 解析带等级的物品（RESO/XMP/US/CUBE）。
func ParseLevelItem(itemID string) (kind string, level int, ok bool) {
	itemID = strings.ToUpper(strings.TrimSpace(itemID))
	switch {
	case strings.HasPrefix(itemID, ItemPrefixResonator):
		level, ok = parseLevel(itemID, ItemPrefixResonator)
		return "RESO", level, ok
	case strings.HasPrefix(itemID, ItemPrefixXMP):
		level, ok = parseLevel(itemID, ItemPrefixXMP)
		return "XMP", level, ok
	case strings.HasPrefix(itemID, ItemPrefixUS):
		level, ok = parseLevel(itemID, ItemPrefixUS)
		return "US", level, ok
	case strings.HasPrefix(itemID, ItemPrefixCube):
		level, ok = parseLevel(itemID, ItemPrefixCube)
		return "CUBE", level, ok
	default:
		return "", 0, false
	}
}

// CubeXM 返回能量块恢复的 XM。
func CubeXM(level int) int {
	if xm, ok := cubeXMByLevel[level]; ok {
		return xm
	}
	return 0
}

// IsKey 判断是否为 Portal Key，并返回 portalId。
func IsKey(itemID string) (string, bool) {
	itemID = strings.TrimSpace(itemID)
	if strings.HasPrefix(itemID, ItemPrefixKey) {
		return strings.TrimPrefix(itemID, ItemPrefixKey), true
	}
	return "", false
}

// KeyCount 返回指定 Portal Key 数量。
func KeyCount(inventory map[string]int, portalID string) int {
	if inventory == nil {
		return 0
	}
	return inventory[ItemIDKey(portalID)]
}

func parseLevel(itemID, prefix string) (int, bool) {
	raw := strings.TrimPrefix(itemID, prefix)
	level, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return level, true
}
