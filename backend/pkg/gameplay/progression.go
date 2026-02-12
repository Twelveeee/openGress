package gameplay

var apThresholdByLevel = map[int]int{
	1:  0,
	2:  2500,
	3:  20000,
	4:  70000,
	5:  150000,
	6:  300000,
	7:  600000,
	8:  1200000,
	9:  2400000,
	10: 4000000,
	11: 6000000,
	12: 8400000,
	13: 12000000,
	14: 17000000,
	15: 24000000,
	16: 40000000,
}

var maxXMByLevel = map[int]int{
	1:  3000,
	2:  4000,
	3:  5000,
	4:  6000,
	5:  7000,
	6:  8000,
	7:  9000,
	8:  10000,
	9:  11500,
	10: 13000,
	11: 14500,
	12: 16000,
	13: 17500,
	14: 19000,
	15: 20500,
	16: 22000,
}

// NormalizeLevel 将等级限制在 [1, 16]。
func NormalizeLevel(level int) int {
	if level < 1 {
		return 1
	}
	if level > 16 {
		return 16
	}
	return level
}

// LevelForAP 根据累计 AP 计算当前等级（L1-L16）。
func LevelForAP(ap int) int {
	if ap < 0 {
		ap = 0
	}
	level := 1
	for i := 2; i <= 16; i++ {
		if ap >= apThresholdByLevel[i] {
			level = i
		}
	}
	return level
}

// MaxXMForLevel 返回指定等级的 XM 上限。
func MaxXMForLevel(level int) int {
	level = NormalizeLevel(level)
	if v, ok := maxXMByLevel[level]; ok {
		return v
	}
	return maxXMByLevel[1]
}
