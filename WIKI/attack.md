# Ingress 攻击机制（XMP）

> 更新时间：2026-02-11  
> 目的：给后端实现提供可落地的 XMP 攻击规则。  
> 说明：官方未公开的概率/精确公式已单独标注为“社区口径”。

## 1. XMP 基础参数

| XMP 等级 | 最大伤害（Power） | 爆炸半径（Range） | XM 消耗 |
|---|---:|---:|---:|
| L1 | 150 | 42m | 50 |
| L2 | 300 | 48m | 100 |
| L3 | 500 | 58m | 150 |
| L4 | 900 | 72m | 200 |
| L5 | 1200 | 90m | 250 |
| L6 | 1500 | 112m | 300 |
| L7 | 1800 | 138m | 350 |
| L8 | 2700 | 168m | 400 |

规则：`xmCost = 50 * level`

## 2. XMP 攻击结算流程

### 2.1 触发与消耗

1. 校验玩家背包中存在对应等级 XMP。  
2. 校验玩家 XM 足够（扣除 `xmCost`）。  
3. 以玩家当前位置作为爆心，计算半径内受影响目标（可同时影响多个敌方 Portal）。  

### 2.2 对每个 Resonator 计算伤害

1. 先取该 XMP 等级的 `Power`。  
2. 根据爆心到 Resonator 的距离做伤害衰减（社区口径，见 3.1）。  
3. 按 Portal 防御（Shield/Link/Field）计算 mitigation（见 3.2）。  
4. 得到最终伤害并扣减 Resonator 能量；能量<=0 时 Resonator 销毁。  

可用实现形式（建议）：

```text
rawDamage = power * chargeMultiplier * distanceMultiplier * critMultiplier
finalDamage = floor(rawDamage * (1 - mitigation))
```

### 2.3 连锁拓扑规则

- 当 Portal 的 Resonator 数量下降导致不满足链接条件时，相关 Link 会断开。  
- Link 断开后，依附该 Link 的 Field 同步消失。  
- 这会触发地图拓扑增量广播（Portal/Link/Field 一并更新）。  

## 3. 关键机制细节

### 3.1 距离衰减（社区口径）

官方未公开精确函数。社区长期采用“随距离递减”模型。推荐在服务端使用分段衰减（便于调参）：

| 距离占比（distance / range） | 伤害倍率建议 |
|---|---:|
| 0% ~ 20% | 1.00 |
| 20% ~ 40% | 0.85 |
| 40% ~ 60% | 0.65 |
| 60% ~ 80% | 0.45 |
| 80% ~ 100% | 0.25 |
| >100% | 0 |

> 说明：也可用线性衰减近似。若后续需要拟真，可改为更细粒度曲线。

### 3.2 Mitigation（减伤）

官方对“存在减伤机制”有描述，但未公开完整公式。常见社区实现口径：

1. `shieldMitigation`: 来自 Shield/Aegis。  
2. `linkMitigation`: 来自 Portal 链接数量（Links）。  
3. `fieldMitigation`: 来自 Portal 覆盖 Field 数量。  
4. 总减伤通常设置上限（常用 `95%`），避免无伤。  

建议实现：

```text
mitigation = min(0.95, shieldMitigation + linkMitigation + fieldMitigation)
```

### 3.3 暴击与 Mod 摧毁

- XMP 存在“暴击（critical hit）”概念，但官方未公开精确概率。  
- Mod 是否被摧毁通常按 stickiness（黏性/存活概率）判定；不是按 XM 直接扣减。  
- 建议把 `critChance`、`critMultiplier`、`modDestroyChance` 都配置化。

### 3.4 反击（Counterattack）

- 攻击敌方 Portal 时可能触发反击（概率官方未公开）。  
- 反击命中后会扣攻击者 XM。  
- 常规反击伤害（社区通用口径）：L1~L8 为 `75/150/300/500/750/1125/1625/2500`。  
- `Turret` 更偏向提高触发频率，`Force Amp` 更偏向提高反击伤害。

## 4. 后端实现建议（与当前项目对齐）

1. `attack_xmp_cost` 按 `50 * level` 固化。  
2. 将“距离衰减、减伤、暴击、反击”全部参数化，禁止写死在 handler。  
3. 保留官方未知项的开关与配置，文档标记为“社区口径默认值”。  
4. 攻击结算后统一触发：
   - 受损/销毁 Resonator 更新
   - Link/Field 拓扑清理
   - 受影响对象的地图增量广播

## 5. 参考来源

- 官方：  
  - https://support.ingress.com/hc/en-us/articles/41139790969875-Attacking-Opponent-Portals  
  - https://support.ingress.com/hc/en-us/articles/41140354084123-Glossary
- 社区数据（数值与机制细节）：  
  - https://ingress.fandom.com/wiki/XMP_Burster  
  - https://ingress.fandom.com/wiki/Damage  
  - https://ingress.fandom.com/wiki/Mitigation  
  - https://ingress.fandom.com/wiki/Counterattack  
  - https://ingress.fandom.com/wiki/Stickiness
