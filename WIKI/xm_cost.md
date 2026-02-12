# Ingress XM 消耗机制（用于修正当前项目实现）

> 更新时间：2026-02-11  
> 说明：本文整理官方口径 + 社区公认数据，供后端参数修正使用。官方未公开精确概率的部分已单独标注。

## 1. 主动操作 XM 消耗

### 1.1 部署 Resonator（`res`）

按物品等级消耗 XM：

| Resonator 等级 | XM 消耗 |
|---|---:|
| L1 | 50 |
| L2 | 100 |
| L3 | 150 |
| L4 | 200 |
| L5 | 250 |
| L6 | 300 |
| L7 | 350 |
| L8 | 400 |

规则可写为：`xmCost = 50 * itemLevel`

### 1.2 发射 XMP（`xmp`）

按物品等级消耗 XM（与 Resonator 同档）：

| XMP 等级 | XM 消耗 |
|---|---:|
| L1 | 50 |
| L2 | 100 |
| L3 | 150 |
| L4 | 200 |
| L5 | 250 |
| L6 | 300 |
| L7 | 350 |
| L8 | 400 |

规则可写为：`xmCost = 50 * itemLevel`

### 1.3 创建 Link（`link`）

- 固定消耗：`250 XM`

### 1.4 部署 Mod（`mod`）

按类型/稀有度消耗 XM：

按照稀有度来，Common 400xm,Rare 800xm ,Very Rare 1000xm 
Force Amp ,Turret ,Link Amp 都是 Rare

| Mod | XM 消耗 |
|---|---:|
| Portal Shield (Common) | 400 |
| Portal Shield (Rare) | 800 |
| Portal Shield (Very Rare) | 1000 |
| Aegis Shield | 1000 |
| Heat Sink (Common) | 400 |
| Heat Sink (Rare) | 800 |
| Heat Sink (Very Rare) | 1000 |
| Multi-Hack (Common) | 400 |
| Multi-Hack (Rare) | 800 |
| Multi-Hack (Very Rare) | 1000 |
| Force Amp | 800 |
| Turret | 800 |
| Link Amp | 800 |
| ITO EN Transmuter (+/-) | 1000 |

---

## 2. 被攻击概率与 XM 伤害概率口径

### 2.1 反击触发概率（官方状态）

- **官方未公开精确触发概率公式或固定百分比**。  
- 可确认的是：攻击敌方 Portal 时“可能被反击”，不是每次必触发。

### 2.2 反击伤害（命中后 XM 扣减）

常规反击伤害按 Portal 等级：

| Portal 等级 | 常规反击伤害（XM） |
|---|---:|
| L1 | 75 |
| L2 | 150 |
| L3 | 300 |
| L4 | 500 |
| L5 | 750 |
| L6 | 1125 |
| L7 | 1625 |
| L8 | 2500 |

此外存在 `critical counterattack`（偶发），社区口径通常按常规约 `3x` 处理。

### 2.3 Mod 对“被攻击概率/伤害”的影响

- `Turret`：提高反击触发相关概率/频率（更容易触发反击）。  
- `Force Amp`：提高反击伤害（不是触发概率）。  
- `Shield` 系列：影响防御与生存，不等于“消耗 XM”；其被摧毁更多是 stickiness（存活概率）判定。

---

## 3. 落地建议（后端实现）

1. `res` 和 `xmp` 统一使用 `50 * level`。  
2. `link` 固定 `250`。  
3. `mod` 建一张静态配置表（按 `modType + rarity` 映射）。  
4. 反击触发概率不要硬编码成“官方值”，建议：
   - 使用可配置参数（默认社区经验值）；
   - 在文档中标注“非官方公开概率”。

---

## 4. 参考来源

- 官方术语/机制说明：  
  - https://support.ingress.com/hc/en-us/articles/41140354084123-Glossary  
  - https://support.ingress.com/hc/en-us/articles/41140376184091-Hacking-Portals
- 社区数据（数值细项）：  
  - https://ingress.fandom.com/wiki/Resonator  
  - https://ingress.fandom.com/wiki/XMP_Burster  
  - https://ingress.fandom.com/wiki/Link  
  - https://ingress.fandom.com/wiki/Portal_Shield  
  - https://ingress.fandom.com/wiki/Heat_Sink  
  - https://ingress.fandom.com/wiki/Multi-Hack  
  - https://ingress.fandom.com/wiki/Force_Amp  
  - https://ingress.fandom.com/wiki/Turret  
  - https://ingress.fandom.com/wiki/Counterattack  
  - https://ingress.fandom.com/wiki/Stickiness
