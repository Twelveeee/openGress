# **Ingress Glyph Hacking 机制与资源获取概率技术分析报告**

## **摘要**

本报告旨在为 Ingress 游戏机制设计文档提供关于 Glyph Hacking（画图入侵）及其资源获取概率的详尽技术分析。报告基于广泛的社区数据、逆向工程推导及实战统计，深入解构了从基础入侵（Base Hack）到高级超频（Overclock）的各项收益模型。核心分析涵盖了不同阵营 Portal 的掉落偏好（Bias）、Glyph Hacking 的奖励算法公式、各类物品（Resonators, XMP, Mods, Keys 等）的加权分布概率，以及 ITO EN Transmuter 等强化套件对掉落池的动态修正机制。本文件将为设计高效的资源获取策略和编写精确的玩家指南提供底层数据支持。

## ---

**1\. 入侵（Hacking）机制概论**

在 Ingress 的增强现实（AR）游戏生态中，"Hacking"（入侵）是资源循环系统的核心输入端。它不仅仅是一个简单的交互动作，而是一个受多重变量影响的随机数生成（RNG）过程。对于试图量化资源获取的设计文档而言，必须首先区分“基础入侵”（Base Hack）与“Glyph Hacking”（画图入侵）这两个层级。基础入侵提供了保底收益，而 Glyph Hacking 则通过技能检定（Skill Check）引入了乘数效应。

### **1.1 基础入侵的运作逻辑**

当特工（Agent）在 Portal 40米范围之内点击“Hack”按钮时，服务器会根据以下参数立即进行一次判定：

1. **特工资格判定**：检查特工是否处于冷却（Cooldown）或烧毁（Burnout）状态，以及是否处于速度锁定（Speedlock）状态。速度锁定会导致“Hack acquired no items”（入侵未获得任何物品）的强制失败判定 1。  
2. **XM 成本扣除**：每次入侵消耗的 XM 量等于 Portal等级 × 50。例如，入侵一个 L8 Portal 需要消耗 400 XM 3。  
3. **阵营判定**：  
   * **友方 Portal（Friendly Portal）**：拥有“建设偏好”（Building Bias），且通常具有“保底机制”，即几乎肯定会掉落至少一个 Resonator 5。  
   * **敌方 Portal（Enemy Portal）**：拥有“攻击偏好”（Attack Bias），并伴随 100 AP 的奖励，但存在约 15%-25% 的概率不掉落任何物品（Hack acquired no items），且会触发反击伤害（Zap）5。  
4. **掉落池生成**：基于 Portal 等级生成基础物品列表。

### **1.2 物品等级分布模型（钟形曲线）**

Portal 的等级（L1-L8）直接决定了产出物品（Resonators, XMP, Ultra Strikes, Power Cubes）的等级分布。这一分布遵循严格的“N ± 2”钟形曲线原则 3：

* **峰值（Peak）**：掉落概率最高的是与 Portal 等级完全一致（N）的物品。  
* **次级（Secondary）**：掉落 N ± 1 等级的物品概率中等。  
* **边缘（Rare）**：掉落 N ± 2 等级的物品概率较低。  
* **截断（Cutoff）**：绝不会掉落超过 N ± 2 范围的物品。

**表 1.1：不同等级 Portal 的物品等级掉落分布估算表**

| Portal 等级 | L1 物品 | L2 物品 | L3 物品 | L4 物品 | L5 物品 | L6 物品 | L7 物品 | L8 物品 |
| :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- | :---- |
| **L1** | **60-70%** | 20-25% | 5-10% | 0% | 0% | 0% | 0% | 0% |
| **L2** | 20% | **60%** | 15% | 5% | 0% | 0% | 0% | 0% |
| **L3** | 5% | 15% | **60%** | 15% | 5% | 0% | 0% | 0% |
| **L4** | 0% | 5% | 15% | **60%** | 15% | 5% | 0% | 0% |
| **L5** | 0% | 0% | 5% | 15% | **60%** | 15% | 5% | 0% |
| **L6** | 0% | 0% | 0% | 5% | 15% | **60%** | 15% | 5% |
| **L7** | 0% | 0% | 0% | 0% | 5% | 15% | **60%** | 20% |
| **L8** | 0% | 0% | 0% | 0% | 0% | 5% | 15% | **80%** |

*分析指出*：L8 Portal 的掉落分布存在特殊的“挤压效应”。由于游戏中不存在 L9 和 L10 物品，原本属于 N+1 和 N+2 的概率区间全部坍缩回 L8，使得 L8 Portal 产出顶级物资的效率远高于其他等级 5。

## ---

**2\. Glyph Hacking 算法与收益乘数**

Glyph Hacking（画图入侵）是提升资源获取效率的核心手段。从系统设计的角度看，Glyph Hacking 并非简单地增加物品数量，而是通过增加“独立的额外掉落判定次数”（Bonus Item Rolls）来实现收益的指数级增长。

### **2.1 奖励计算公式**

Glyph Hacking 的最终收益由两部分组成：**准确度奖励（Hacking Bonus）** 和 **速度奖励（Speed Bonus）**。

#### **2.1.1 准确度奖励 ($B_{hack}$)**

该奖励基于正确输入的 Glyph 数量。公式如下 11：

$$B_{hack} = (10\% \times G_{correct}) + (P \times B_{perfect})$$

$G_{correct}$：正确绘制的 Glyph 数量。每个正确图形提供 10% 奖励。
$P$：完美判定位（Binary）。如果序列全对则为 1，否则为 0。
$B_{perfect}$：完美奖励系数，随 Portal 等级非线性增长。

【开发者著】： 自动hack时 给当前porta的最大准确度奖励。


**表 2.1：各等级 Portal 的 Glyph 参数与完美奖励**

| Hack 等级 | Glyph 数量 | 输入限时 | 完美奖励 (Bperfect​) | 单图分值 (10%) | 最大准确度奖励 |
| :---- | :---- | :---- | :---- | :---- | :---- |
| **L1** | 1 | 20s | 28% | 10% | 38% |
| **L2** | 2 | 20s | 40% | 20% | 60% |
| **L3** | 3 | 20s | 55% | 30% | 85% |
| **L4** | 3 | 19s | 55% | 30% | 85% |
| **L5** | 3 | 18s | 55% | 30% | 85% |
| **L6** | 4 | 17s | 80% | 40% | 120% |
| **L7** | 4 | 16s | 80% | 40% | 120% |
| **L8** | 5 | 15s | 163% | 50% | **213%** |

*注意*：L8 的完美奖励高达 163%，这是 Ingress Prime 当前版本的数值（早期版本曾为 112% 或更低，现已调整以鼓励高等级操作）13。这导致 L8 的全对奖励产生质变。

#### **2.1.2 速度奖励 ($B_{speed}$)**

速度奖励仅在序列全对（$P=1$）时触发。其计算逻辑是剩余时间的线性函数 11：

$$B_{speed} = \lfloor 100 \times \frac{T_{left}}{T_{limit}} \rfloor \%$$

$T_{left}$：完成输入后的剩余时间。
$T_{limit}$：该等级的总限时（见表 2.1）。

这意味着，如果你在 L8 Portal（限时 15秒）上仅用 7.5秒完成了输入，你将获得 50% 的速度奖励。

#### **2.1.3 总奖励与额外判定次数（The "Rolls" Theory）**

社区研究和长期数据分析表明，系统并非直接将奖励百分比乘算到基础物品数量上，而是将总奖励百分比转换为“额外的掉落判定次数”。 经验法则为：**每 30% 的总奖励（Total Bonus）增加 1 次额外的物品掉落判定（Roll）** 13。

$$Total\ Bonus = B_{hack} + B_{speed}$$


$$Bonus\ Rolls \approx \lceil \frac{Total\ Bonus}{30\%} \rceil$$


**案例分析：完美 L8 Glyph Hack**

* 准确度奖励：213% (50% 基础 \+ 163% 完美)  
* 假设速度奖励：50% (极快的手速)  
* 总奖励：263%  
* 额外判定次数：$263 / 30 \approx 8.76$
  这意味着一次完美的 L8 入侵会触发约 **9 次额外的掉落判定**。如果基础入侵提供 3 个物品，加上这 9 次判定生成的物品（每次判定可能生成 1-2 个物品），最终收益可达 12-20 个物品。这就是 Glyph Hacking 收益巨大的数学本质。

## ---

**3\. 各类物质（Items）获取概率详解**

在设计文档中，不仅需要知道“掉落很多”，更需要具体的概率权重。以下数据综合了 FevGames 研究、Reddit 社区统计及大规模 Bot 数据泄露的历史分析 5。

### **3.1 基础物资：Resonators 与 Weapons**

阵营偏好（Faction Bias）在此类物品的掉落中起决定性作用。

**表 3.1：Resonator 与 Weapon 的掉落权重对比**

| 物品类型 | 友方 Portal (Friendly) | 敌方 Portal (Enemy) | 概率特征分析 |
| :---- | :---- | :---- | :---- |
| **Resonator (脚莫)** | **极高** (Bias+) | 低 (Bias-) | 友方 Portal 有保底机制，必定掉落至少 1 个 Resonator 5。敌方 Portal 掉率显著降低。 |
| **XMP Burster (炸)** | 中等 | **高** (Bias+) | 敌方 Portal 是获取 XMP 的主要来源，但需承担“无掉落”风险。 |
| **Ultra Strike (US)** | 低 | 中等 | 通常作为武器类掉落的附属品，敌方 Portal 掉率略高，且高等级 US 掉率极低 18。 |
| **Power Cube (糖)** | 中等 | 中等 | 阵营偏好不明显，掉落相对均匀。 |

*设计提示*：在文档中应强调，尽管敌方 Portal 倾向于掉落武器，但由于敌方 Portal 存在约 20% 的“Hack acquired no items”失败率 7，因此单位时间内获取 XMP 的总量，**友方 L8 Portal 配合 Glyph Hacking 往往仍优于敌方 Portal**，除非玩家极其缺乏物资且无法建立友方高级塔。

### **3.2 Portal Keys (钥匙)**

Portal Key 的掉落判定逻辑独立于物品池，是在物品判定之后进行的 20。

* **无 Key 状态**：基础掉落率约为 **75-80%**。  
* **有 Key 状态**：基础掉落率接近 **0%**（如果不使用指令）。  
* **Glyph 指令修正**：  
  * \*\*"MORE" 指令 (^) \*\*：强制请求 Key。即使背包中已有 Key，掉落率也恢复至 **75%** 左右。且在 Bonus Roll 中可能出现额外的 Key。完美 L8 Glyph Hack 配合 "MORE" 指令，一次最多可获得 **4 把 Key**（1 把基础 \+ 3 把奖励）13。  
  * **"LESS" 指令 (v)**：将 Key 掉落率强制降为 **0%**。  
  * **"SPEED" 指令**：忽略所有其他物品，极大缩短动画，仅进行 Key 的判定。

### **3.3 Mods (强化套件) \- 普通与稀有**

Mods 的掉落遵循指数衰减分布。所有 Mod 共享同一个掉落“槽位”的概率。

* **Common (白) Mod**：约 1/10 \- 1/12 次 Hack。  
* **Rare (蓝/绿) Mod**：约 1/35 \- 1/50 次 Hack。  
* **Very Rare (紫/红) Mod**：约 1/100 \- 1/200 次 Hack。

*注：Shield（盾）是最常见的 Mod，Turret (炮塔)、Force Amp (法放)、Link Amp (林安) 的掉落率相对 Shield 略低。* 23

### **3.4 极稀有物品 (Very Rare & Exotic)**

这是文档中最关键的数据点，因为这些物品决定了 Farm 的成败。概率基于**单次判定（Per Item Roll）**。

**表 3.2：极稀有物品掉落概率估算表**

| 物品名称 | 估算概率 (每次 Roll) | 平均获取所需 Hack 次数 (Base Hack) | 备注 |
| :---- | :---- | :---- | :---- |
| **Jarvis Virus (绿毒)** | \~0.04% \- 0.1% | 1,000 \- 2,500 | 阵营无关，蓝绿塔均掉落 17。 |
| **ADA Refactor (蓝毒)** | \~0.04% \- 0.1% | 1,000 \- 2,500 | 同上，掉率极低，通常被视为 1/1000 级别的掉落 25。 |
| **Aegis Shield (红盾)** | \~0.05% | \~2,000 | 替代了旧版的 AXA Shield。防御力最高，粘性最高 27。 |
| **VR Multi-Hack (红多)** | \~0.04% | \~2,500 | 极为稀有，用于构建持续 Farm 基地 23。 |
| **VR Heat Sink (红热)** | \~0.04% | \~2,500 | 用于重置冷却和烧毁计数。 |
| **Kinetic Capsule (红桶)** | 极低 (动态调整) | 不定 | 掉率曾经历调整，目前依然非常稀有，经常被混淆为不再掉落 16。 |
| **Quantum Capsule (量子)** | **0% (已移除)** | N/A | 历史上曾掉落，现已被 Kinetic Capsule 替代或不再产出 29。 |

*深入洞察*：虽然单次 Roll 的概率仅为 1/1000，但进行一次完美的 L8 Glyph Hack 会触发约 10 次 Roll。根据二项分布  $1 - (1-p)^{10}$，单次完美画图获得至少一个 VR 物品的概率提升至约 **1%**。这解释了为何核心玩家必须依赖 Glyph Hacking 来积累战略物资（毒、红盾）。

## ---

**4\. 高级修正因子与倍率系统**

在基础概率之上，Ingress 引入了多种机制来修正或倍增上述概率。设计文档需特别标注这些“修正器”（Modifiers）。

### **4.1 ITO EN Transmuter (+/-) 的转化机制**

ITO EN 是一种特殊的 Mod，它不增加掉落数量，而是改变掉落物品的**类型（Class）**。它通过重写掉落表来实现 31。

* **ITO EN (+)**：防御向转化。  
  * **机制**：将所有被判定为“攻击性”（XMP, Ultra Strike）的 Roll 强制转换为“防御性/建设性”（Resonator, Shield）。  
  * **结果**：获得大量 Resonator 和 Shield（可能出现“双倍盾”现象），**0** 个 XMP/US。  
  * **适用场景**：极速补充 Resonator，补给防御物资。  
* **ITO EN (-)**：攻击向转化。  
  * **机制**：将所有被判定为“防御性”的 Roll 强制转换为“攻击性”。  
  * **结果**：获得大量 XMP 和 US，**0** 个 Resonator/Shield。  
  * **结果修正**：它能有效规避敌方 Portal 的“不掉落 Resonator”的劣势，甚至在友方 Portal 上创造出超越敌方 Portal 的武器产出效率，且没有反击伤害。  
* **免疫物品**：Power Cubes、Keys、VR 道具（如毒、红盾）通常不受转化影响，独立掉落 33。

### **4.2 Portal Fracker (压裂器) \- 翻倍器**

【开发者著】：本次开发忽略 Portal Fracker 

Fracker 是付费道具，持续 10 分钟（或 150 次 Hack）。

* **机制**：它不是增加 Roll 的次数，而是直接对最终生成的物品清单（包括 Glyph 奖励）进行数量 x2 的操作 34。  
* **策略**：配合 L8 Glyph Hack，原本 1 个 VR 物品的掉落会变成 2 个。这是“刷毒”队伍的标准配置。

### **4.3 Overclock (超频) \- 4倍收益**

【开发者著】：本次开发忽略 Overclock

Overclock 是 Ingress Prime 引入的基于 VPS（视觉定位）的高级功能。

* **前置条件**：Portal 必须已被扫描并激活 VPS，玩家需在 20 米范围内启用 AR 模式。  
* **机制**：玩家需要在 3D 空间中快速校准并输入 Glyph。  
* **收益**：  
  * 单次操作消耗 **4 次** Hack 的 Burnout 计数。  
  * 获得 **4 倍** 的 Glyph Hack 物品收益。  
  * 获得极高的 AP 奖励。  
* **概率影响**：它本质上是“时间压缩”。它并没有改变单次 Roll 的概率，而是让你在 30 秒内完成了 4 次完美 Glyph Hack 的工作量。对于在此期间恰好触发的 1/1000 概率事件，效率提升了 4 倍 35。

### **4.4 7-Day Streak (七天签到奖励)**

【开发者著】：本次开发忽略 7-Day Streak

* **机制**：连续第 7 天的第一次 Hack 会获得 **2 倍** 收益。  
* **堆叠效应**：如果第 7 天的首 Hack 结合了 Fracker 和 Apex（AP翻倍），收益将极其可观（物品 4 倍，AP 可能更高）。34

## ---



**6\. 历史数据对比与文档准确性警告**

在编写文档时，需特别注意剔除过时数据，以免误导用户：

1. **Quantum Capsule (量子桶)**：不再通过 Hack 掉落，也不再具有增殖 VR 道具的能力（已被 Kinetic Capsule 替代）29。  
2. **MUFG Capsule**：已更名为 Quantum Capsule 并最终退役。  
3. **AXA Shield**：已更名为 Aegis Shield，且粘性（Stickiness）有过调整（从 800k 降至 550k，但仍是最高）27。  
4. **L8 完美奖励**：从早期的 112% 提升至 **163%** 13。引用旧数据会导致收益计算严重偏差。  
5. **每日免费道具**：自 2025 年起，商店的每日免费道具改为**每周轮换制**（Bi-weekly schedule），例如周日固定给 Resonator，周五给 XMP 等 38。文档需更新此获取途径。

## ---

**7\. 结论**

Ingress 的 Glyph Hacking 系统是一个精密的概率工程。对于玩家而言，**准确度（Accuracy）是第一生产力**。一次失败的 L8 Glyph Hack（例如错一个图）会导致失去 163% 的完美奖励，直接损失约 5-6 次物品掉落判定机会，这是任何手速都无法弥补的。

在您的“Hack 文档”中，建议将 **Portal 等级**（决定物品上限）、**Glyph 准确度**（决定物品下限和总量）以及 **Mod 配置**（决定物品类型）作为三大支柱进行阐述。结合上述概率表，用户将能够建立起对“投入时间/XM”与“产出物资”的精确数学预期。

\[报告结束\]

#### **引用的著作**

1. Hack acquired no items : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/3vgl46/hack\_acquired\_no\_items/](https://www.reddit.com/r/Ingress/comments/3vgl46/hack_acquired_no_items/)  
2. Unable to hack or recharge after traveling faster than 40MPH. : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/1fv8kn/unable\_to\_hack\_or\_recharge\_after\_traveling\_faster/](https://www.reddit.com/r/Ingress/comments/1fv8kn/unable_to_hack_or_recharge_after_traveling_faster/)  
3. Hack | Fev Games, 访问时间为 二月 11, 2026， [https://fevgames.net/ingress/ingress-guide/actions/hack/](https://fevgames.net/ingress/ingress-guide/actions/hack/)  
4. Ingress Bootcamp v1.0.1 | PDF \- Scribd, 访问时间为 二月 11, 2026， [https://www.scribd.com/document/672805943/Ingress-Bootcamp-v1-0-1](https://www.scribd.com/document/672805943/Ingress-Bootcamp-v1-0-1)  
5. What is the difference between hacking a lvl 8 portal that is your side vs the other side, reward wise? : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/6mfx0z/what\_is\_the\_difference\_between\_hacking\_a\_lvl\_8/](https://www.reddit.com/r/Ingress/comments/6mfx0z/what_is_the_difference_between_hacking_a_lvl_8/)  
6. Does farming enemy or friendly portals yield more bursters? : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/2qj2qu/does\_farming\_enemy\_or\_friendly\_portals\_yield\_more/](https://www.reddit.com/r/Ingress/comments/2qj2qu/does_farming_enemy_or_friendly_portals_yield_more/)  
7. Why do I keep getting "Hack acquired no items" in Ingress? \- Arqade \- Stack Exchange, 访问时间为 二月 11, 2026， [https://gaming.stackexchange.com/questions/95861/why-do-i-keep-getting-hack-acquired-no-items-in-ingress](https://gaming.stackexchange.com/questions/95861/why-do-i-keep-getting-hack-acquired-no-items-in-ingress)  
8. How do I level up in ingress? \- Arqade \- Stack Exchange, 访问时间为 二月 11, 2026， [https://gaming.stackexchange.com/questions/94377/how-do-i-level-up-in-ingress](https://gaming.stackexchange.com/questions/94377/how-do-i-level-up-in-ingress)  
9. Lower drop rates for L8 resonators? : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/13g52hw/lower\_drop\_rates\_for\_l8\_resonators/](https://www.reddit.com/r/Ingress/comments/13g52hw/lower_drop_rates_for_l8_resonators/)  
10. Level distribution of hacked resonators and weapons : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/31j570/level\_distribution\_of\_hacked\_resonators\_and/](https://www.reddit.com/r/Ingress/comments/31j570/level_distribution_of_hacked_resonators_and/)  
11. How are the glyph hack bonuses calculated? \- Arqade \- Stack Exchange, 访问时间为 二月 11, 2026， [https://gaming.stackexchange.com/questions/214779/how-are-the-glyph-hack-bonuses-calculated](https://gaming.stackexchange.com/questions/214779/how-are-the-glyph-hack-bonuses-calculated)  
12. Glyph Hack \- Fev Games, 访问时间为 二月 11, 2026， [https://fevgames.net/ingress/ingress-guide/actions/glyph-hack/](https://fevgames.net/ingress/ingress-guide/actions/glyph-hack/)  
13. Glyph Hacking \- Ingress Wiki \- Fandom, 访问时间为 二月 11, 2026， [https://ingress.fandom.com/wiki/Glyph\_Hacking](https://ingress.fandom.com/wiki/Glyph_Hacking)  
14. Does successful Glyph Hacks have any effect on the drop rates of rare items? \- Arqade, 访问时间为 二月 11, 2026， [https://gaming.stackexchange.com/questions/315826/does-successful-glyph-hacks-have-any-effect-on-the-drop-rates-of-rare-items](https://gaming.stackexchange.com/questions/315826/does-successful-glyph-hacks-have-any-effect-on-the-drop-rates-of-rare-items)  
15. Glyph Hacking – how many more items do you really get? : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/bqfehw/glyph\_hacking\_how\_many\_more\_items\_do\_you\_really/](https://www.reddit.com/r/Ingress/comments/bqfehw/glyph_hacking_how_many_more_items_do_you_really/)  
16. Item Hack/Drop Rates : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/1ge6f0e/item\_hackdrop\_rates/](https://www.reddit.com/r/Ingress/comments/1ge6f0e/item_hackdrop_rates/)  
17. How rare our jarvis viruses? : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/37j1ud/how\_rare\_our\_jarvis\_viruses/](https://www.reddit.com/r/Ingress/comments/37j1ud/how_rare_our_jarvis_viruses/)  
18. The Secret World Legends (first impressions) \- Hurricane Electric Internet Services, 访问时间为 二月 11, 2026， [http://toad.he.net/cgi-bin/suid/\~stupid/news\_display.cgi?action=archive](http://toad.he.net/cgi-bin/suid/~stupid/news_display.cgi?action=archive)  
19. Hacking | Ingress Wiki \- Fandom, 访问时间为 二月 11, 2026， [https://ingress.fandom.com/wiki/Hacking](https://ingress.fandom.com/wiki/Hacking)  
20. Is there a way to increase the odds of getting a portal key? \- Arqade, 访问时间为 二月 11, 2026， [https://gaming.stackexchange.com/questions/97986/is-there-a-way-to-increase-the-odds-of-getting-a-portal-key](https://gaming.stackexchange.com/questions/97986/is-there-a-way-to-increase-the-odds-of-getting-a-portal-key)  
21. What are the odds of getting a portal key through hacking? \- Arqade, 访问时间为 二月 11, 2026， [https://gaming.stackexchange.com/questions/112153/what-are-the-odds-of-getting-a-portal-key-through-hacking](https://gaming.stackexchange.com/questions/112153/what-are-the-odds-of-getting-a-portal-key-through-hacking)  
22. Help me understand the logic behind the rarity of portal keys : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/mzbyeh/help\_me\_understand\_the\_logic\_behind\_the\_rarity\_of/](https://www.reddit.com/r/Ingress/comments/mzbyeh/help_me_understand_the_logic_behind_the_rarity_of/)  
23. Mod | Fev Games, 访问时间为 二月 11, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/](https://fevgames.net/ingress/ingress-guide/items/mod/)  
24. Why are Shields so rare lately? : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/16r6aw4/why\_are\_shields\_so\_rare\_lately/](https://www.reddit.com/r/Ingress/comments/16r6aw4/why_are_shields_so_rare_lately/)  
25. ADA Refactor | Ingress Wiki \- Fandom, 访问时间为 二月 11, 2026， [https://ingress.fandom.com/wiki/ADA\_Refactor](https://ingress.fandom.com/wiki/ADA_Refactor)  
26. What are the chances of receiving a Jarvis/ADA after hacking a portal? : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/1nru2wq/what\_are\_the\_chances\_of\_receiving\_a\_jarvisada/](https://www.reddit.com/r/Ingress/comments/1nru2wq/what_are_the_chances_of_receiving_a_jarvisada/)  
27. Portal Shield \- Fev Games, 访问时间为 二月 11, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/portal-shield/](https://fevgames.net/ingress/ingress-guide/items/mod/portal-shield/)  
28. Anyone else noticing an increased drop rate of VR mods lately? : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/1ikupcp/anyone\_else\_noticing\_an\_increased\_drop\_rate\_of\_vr/](https://www.reddit.com/r/Ingress/comments/1ikupcp/anyone_else_noticing_an_increased_drop_rate_of_vr/)  
29. Drop rate of kinetic capsules : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/116es5j/drop\_rate\_of\_kinetic\_capsules/](https://www.reddit.com/r/Ingress/comments/116es5j/drop_rate_of_kinetic_capsules/)  
30. The Rise and Fall of the AXA Shield & MUFG Capsule | Fev Games, 访问时间为 二月 11, 2026， [https://fevgames.net/the-rise-and-fall-of-the-aegis-axa-shield-mufg-capsule/](https://fevgames.net/the-rise-and-fall-of-the-aegis-axa-shield-mufg-capsule/)  
31. Transmuter (mod) | Ingress | FevGames, 访问时间为 二月 11, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/transmuter/](https://fevgames.net/ingress/ingress-guide/items/mod/transmuter/)  
32. ITO EN Transmuter | Ingress Wiki \- Fandom, 访问时间为 二月 11, 2026， [https://ingress.fandom.com/wiki/ITO\_EN\_Transmuter](https://ingress.fandom.com/wiki/ITO_EN_Transmuter)  
33. Clarification about ito en transmuters : r/Ingress \- Reddit, 访问时间为 二月 11, 2026， [https://www.reddit.com/r/Ingress/comments/tg2yho/clarification\_about\_ito\_en\_transmuters/](https://www.reddit.com/r/Ingress/comments/tg2yho/clarification_about_ito_en_transmuters/)  
34. How many items do you get when you Overclock, Fracker and 7 Day Streak together? Ingress Gameplay \- YouTube, 访问时间为 二月 11, 2026， [https://www.youtube.com/watch?v=HU0rokvPd60](https://www.youtube.com/watch?v=HU0rokvPd60)  
35. Overclock \- Ingress Wiki \- Fandom, 访问时间为 二月 11, 2026， [https://ingress.fandom.com/wiki/Overclock](https://ingress.fandom.com/wiki/Overclock)  
36. Overclock | Fev Games, 访问时间为 二月 11, 2026， [https://fevgames.net/ingress/ingress-guide/actions/overclock/](https://fevgames.net/ingress/ingress-guide/actions/overclock/)  
37. Overclock Glyph Hacking | Ingress Prime Gameplay Tutorial \- YouTube, 访问时间为 二月 11, 2026， [https://www.youtube.com/watch?v=jFM-PpkDlN8](https://www.youtube.com/watch?v=jFM-PpkDlN8)  
38. Update to Daily Free Items: Introducing a Weekly Schedule \- Ingress, 访问时间为 二月 11, 2026， [https://ingress.com/en/news/2025-dailyitemweekrotation](https://ingress.com/en/news/2025-dailyitemweekrotation)  
39. Update to Daily Free Items: Introducing a Weekly Schedule \- Ingress, 访问时间为 二月 11, 2026， [https://ingress.com/news/2025-dailyitemweekrotation](https://ingress.com/news/2025-dailyitemweekrotation)

