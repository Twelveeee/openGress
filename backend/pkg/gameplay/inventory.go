package gameplay

// CapacityForLevel 返回 General/Key 槽位容量。
func CapacityForLevel(level int) (general int, keys int) {
	if level <= 0 {
		level = 1
	}
	if level <= 8 {
		general = 500 + (level-1)*50
		keys = 500 + (level-1)*50
		return general, keys
	}
	if level > 16 {
		level = 16
	}
	general = 850 + (level-8)*100
	keys = 850 + (level-8)*100
	return general, keys
}

// InventoryCounts 统计 General/Key 槽位占用。
func InventoryCounts(inventory map[string]int) (general int, keys int) {
	if inventory == nil {
		return 0, 0
	}
	for itemID, count := range inventory {
		if count <= 0 {
			continue
		}
		if _, ok := IsKey(itemID); ok {
			keys += count
		} else {
			general += count
		}
	}
	return general, keys
}

// CanAddItem 判断是否还有容量容纳新增物品。
func CanAddItem(inventory map[string]int, itemID string, amount int, level int) bool {
	if amount <= 0 {
		return true
	}
	generalCap, keyCap := CapacityForLevel(level)
	currentGeneral, currentKeys := InventoryCounts(inventory)
	if _, ok := IsKey(itemID); ok {
		if currentKeys+amount <= keyCap {
			return true
		}
		// Key 槽满时，可占用 General 槽。
		return currentGeneral+amount <= generalCap
	}
	return currentGeneral+amount <= generalCap
}

// CanAddKey 判断是否超过单 Portal Key 上限与容量限制。
func CanAddKey(inventory map[string]int, portalID string, amount int, level int) bool {
	if amount <= 0 {
		return true
	}
	if KeyCount(inventory, portalID)+amount > MaxKeyPerPortal {
		return false
	}
	return CanAddItem(inventory, ItemIDKey(portalID), amount, level)
}
