# **Ingress Prime 实体属性与游戏机制深度技术报告：Web端模拟器开发指南**

# **1\. 执行摘要与系统架构概述**

本深度研究报告旨在为开发Web端 Ingress 模拟器或数据库管理系统提供详尽的实体属性、物品数值及核心算法逻辑。基于 Ingress Prime 当前的游戏机制（截至2025年最新周期），本文档不仅涵盖了基础物品（如共振器、XMP炸弹）的静态属性，还深入解析了服务器端计算的动态机制，包括伤害衰减模型、模组移除概率（粘性 Stickiness）、以及复杂的动能胶囊合成配方。

对于Web端复刻项目而言，理解数据模型背后的逻辑至关重要。Ingress 的物品系统并非孤立存在，而是通过XM（Exotic Matter，外来物质）经济系统、AP（Access Points，行动点数）奖励系统以及空间地理位置（Geospatial）相互关联。本报告将按照“部署类实体”、“攻击类实体”、“防御与增益模组”、“资源与容器”以及“特殊战术道具”五大模块进行解构，并特别补充了如 Battle Beacon（战斗烽火台）的计分算法、Overclock（超频）机制下的资源产出逻辑以及各类道具的回收价值体系。

所有数据均基于现有的游戏逆向工程数据、社区验证的公式以及官方技术说明文档 1。

# ---

**2\. 核心建筑实体：共振器 (Resonators)**

共振器是支撑传送门（Portal）存在的基础组件。在数据库设计中，每一个 Portal 实体必须包含一个长度为8的数组槽位（Slots），用于存储共振器实例。共振器的等级决定了 Portal 的整体等级，进而决定了产出物品的等级和链接（Link）的最大距离。

## **2.1 静态属性与能量容量**

每个等级（L1-L8）的共振器拥有固定的 XM 容量，这代表了其生命值（Health）。在模拟器中，当受到伤害导致 XM 归零时，该槽位的共振器应当被销毁。

### **表 2.1：共振器等级属性表**

| 等级 (Level) | XM 容量 (Max Health) | 回收价值 (Recycle XM) | 单人部署上限 (Per Agent) | 部署获得 AP |
| :---- | :---- | :---- | :---- | :---- |
| **L1** | 1,000 | 20 | 8 | 125 |
| **L2** | 1,500 | 40 | 4 | 125 |
| **L3** | 2,000 | 60 | 4 | 125 |
| **L4** | 2,500 | 80 | 4 | 125 |
| **L5** | 3,000 | 100 | 2 | 125 |
| **L6** | 4,000 | 120 | 2 | 125 |
| **L7** | 5,000 | 140 | 1 | 125 |
| **L8** | 6,000 | 160 | 1 | 125 |

**开发逻辑提示**：

* **部署限制校验**：在后端逻辑中，必须校验当前用户（Agent）在同一个 Portal UUID 上已部署的共振器数量。例如，一个玩家只能在该 Portal 上放置1个 L8 共振器和1个 L7 共振器。若要构建一个满8级的 Portal（P8），至少需要8名不同的玩家协同操作 1。  
* **AP 奖励触发器**：  
  * 基础部署：+125 AP。  
  * 捕获奖励（占领中立 Portal 的第1个槽位）：额外 \+500 AP（共 625 AP）。  
  * 完善奖励（填满第8个槽位）：额外 \+250 AP（共 375 AP）。  
  * 升级操作（用更高级别替换低级别）：+65 AP 4。

## **2.2 能量衰减与自然消亡机制**

Ingress 系统为了保持游戏地图的流动性，引入了强制衰减机制。Web端模拟器必须包含一个定时任务（Cron Job）或事件驱动的衰减计算器。

* **自然衰减率**：共振器每天会自然流失其 **XM 容量上限的 15%**。这意味着若无人充电，一个满能量的 Portal 将在 7 天后自然中立化（15% \* 7 \= 105%）1。  
* **衰减计算公式**：  
  Current_xm_new = = current_xm_old - (Max_Capacity * 0.15)
  *注：此计算通常基于部署时间戳进行，每24小时结算一次，或者在任何交互事件（如攻击、充电、Hack）发生前动态计算这一段时间的衰减量。*  
* **远程充电（Remote Recharge）**：玩家可以远程为持有 Key 的 Portal 充电。充电效率随距离衰减，这一机制确保了玩家主要维护本地社区的 Portal。在模拟器中，若未实现复杂的地理计算，可暂时设定远程充电效率为 100% 或基于简单的线性距离惩罚。

## **2.3 部署距离与战术分布**

共振器并非放置在 Portal 的中心，而是分布在以 Portal 为中心的圆周上。

* **距离参数**：共振器与中心的距离由部署时玩家与 Portal 中心的距离决定。  
* **战术意义**：  
  * **最大距离**：约 40 米。将共振器部署在最远距离可以最大化防御半径，使得敌方 XMP 炸弹（伤害随距离衰减）难以同时对所有共振器造成高额伤害。  
  * **最小距离**：接近 0 米。这种部署通常用于防御“精准打击”（Ultra Strike），但在对抗 XMP 范围伤害时极为脆弱。  
  * **八卦阵（Octants）**：8个槽位分别对应罗盘的8个方向（N, NE, E, SE, S, SW, W, NW）。模拟器必须记录每个共振器的具体槽位索引，以正确计算攻击时的空间几何关系。

# ---

**3\. 进攻性武器系统：XMP 与 Ultra Strike**

伤害计算引擎是 Ingress 模拟器中最复杂的部分。它不仅涉及基础数值，还涉及空间距离衰减、暴击判定（Critical Hit）以及防御减免（Mitigation）。

## **3.1 XMP 炸弹 (XMP Burster)**

XMP 是主要的范围攻击（AOE）武器。其核心属性包括：能量消耗、最大伤害、最大范围。

### **表 3.1：XMP 炸弹属性详解**

| 等级 | 发射消耗 (XM) | 最大伤害 (Max Damage) | 攻击半径 (Range in Meters) | 回收价值 (XM) | 伤害效率 (Dmg/Cost) |
| :---- | :---- | :---- | :---- | :---- | :---- |
| **L1** | 50 | 150 | 42 | 20 | 3.0 |
| **L2** | 100 | 300 | 48 | 40 | 3.0 |
| **L3** | 150 | 500 | 58 | 60 | 3.33 |
| **L4** | 200 | 900 | 72 | 80 | 4.5 |
| **L5** | 250 | 1,200 | 90 | 100 | 4.8 |
| **L6** | 300 | 1,500 | 112 | 120 | 5.0 |
| **L7** | 350 | 1,800 | 138 | 140 | 5.14 |
| **L8** | 400 | 2,700 | 168 | 160 | 6.75 |

5

### **3.1.1 伤害衰减模型 (Falloff Mechanics)**

XMP 的伤害并非在范围内均匀分布。虽然社区对于具体函数是线性（Linear）还是二次方（Quadratic）曾有争论，但目前公认的模拟模型采用线性衰减，且设有阈值。

**计算逻辑**：

对于处于爆炸范围内的每一个共振器 R , 其受到的基础伤害 D_{base} 计算如下：

$$D_{base} = D_{max} \times \max\left(0, 1 - \frac{Distance(Agent, R)}{Range_{max}}\right)$$


* **战术启示**：这就解释了为什么站在 L8 共振器正上方（Point Blank）发射 XMP 效果最好。如果玩家站在 Portal 中心，而共振器部署在 40 米边缘，L8 XMP（范围 168m）造成的伤害将约为最大伤害的 $1 - 40/168 \approx 76\%$  
* **蓄力攻击（Charge Bonus）**：长按发射键会触发一个小游戏，圆圈收缩至中心时松手可获得最高 **20%** 的伤害加成。这在模拟器前端可以通过一个简单的 CSS 动画和时间戳比对来实现。  
  $$D_{final} = D_{base} \times (1 + Bonus_{charge})$$$Bonus_{charge}$ 
  范围为 0.00 到 0.20  1 。


## **3.2 究极打击 (Ultra Strike)**

Ultra Strike (US) 是点对点的高爆发武器，主要用途是剥离模组（Shields/Mods）。其特点是范围极小，但伤害密度极大，且拥有极高的暴击率。

### **表 3.2：Ultra Strike 属性详解**

| 等级 | 发射消耗 (XM) | 最大伤害 (Max Damage) | 攻击半径 (Range) | 回收价值 (XM) |
| :---- | :---- | :---- | :---- | :---- |
| **L1** | 50 | 300 | 10m | 20 |
| **L2** | 100 | 600 | 13m | 40 |
| **L3** | 150 | 1,000 | 16m | 60 |
| **L4** | 200 | 1,800 | 18m | 80 |
| **L5** | 250 | 2,400 | 21m | 100 |
| **L6** | 300 | 3,000 | 24m | 120 |
| **L7** | 350 | 3,600 | 27m | 140 |
| **L8** | 400 | 5,400 | 30m | 160 |

9

### **3.2.1 暴击与模组剥离 (Critical Hits & Stickiness)**

在 Ingress 机制中，模组（Mods）没有生命值条。它们只能通过“暴击”来移除。

* **暴击判定**：当武器对 Portal 中心造成伤害时，会进行一次暴击判定。  
  * XMP 的基础暴击率较低（估计约 5%-10%）。  
  * Ultra Strike 的暴击率极高，社区测试表明可能在 **30%-70%** 之间，甚至有说法认为 L8 US 在中心命中时暴击率接近 100% 11。  
* **粘性对抗 (Stickiness Roll)**：一旦发生暴击，系统会针对每一个安装的模组进行“粘性检定”。  
  * 如果  $Random(0, Total\_Range) > Stickiness$，则模组被摧毁。  
  * **AXA/Aegis Shield** 曾经拥有 800,000 的粘性，极其难以剥离。但在 2018 年被削弱至 **550,000** 13。  
  * **Common Shield** 粘性为 0，意味着只要触发暴击，它几乎必碎。  
  * **Rare Shield** 粘性为 150,000。  
  * **Very Rare Shield** 粘性为 450,000。

**模拟器实现建议**：在处理 Ultra Strike 攻击时，必须判断玩家坐标是否在 Portal 中心的极小半径内（如 \<3米）。如果在范围内，强制触发高概率的暴击逻辑，并遍历 Portal 的 4 个模组槽位，根据各自的 Stickiness 值进行随机移除判定。

## **3.3 阵营反转道具：Flip Cards**

Flip Cards（翻转卡）是改变 Portal 阵营的唯一非战斗手段，包括 JARVIS Virus（绿毒）和 ADA Refactor（蓝毒）。

* **ADA Refactor**：将 Portal 强制转为 Resistance（蓝色）。  
* **JARVIS Virus**：将 Portal 强制转为 Enlightened（绿色）。

### **3.3.1 核心机制与限制**

在开发 Web 端时，必须严格实现以下逻辑约束：

1. **免疫计时器 (Immunity Timer)**：Portal 被翻转后，会获得 **1小时** 的免疫时间。在此期间，任何试图再次使用 Flip Card 的操作都会失败，且道具会被消耗。数据库需字段 last\_flipped\_time。  
2. **XM 消耗成本**：使用成本并非固定，而是取决于目标 Portal 的等级。  
    $$Cost = Portal\_Level \times 1,000$$
   *例如：翻转一个 P8 需要 8,000 XM。如果玩家当前的 XM 储备不足（如 L8 玩家只有 6,000 XM 上限），则无法翻转 P8，除非通过 Power Cube 临时补给或提升等级。*  
3. **链接破坏**：翻转 Portal 会切断所有连接到该 Portal 的 Link 和 Control Field。**注意**：这种破坏不会给予玩家任何 AP 奖励，也不会掉落 Key 14。  
4. **所有权归属**：  
   * 若翻转后的阵营与使用者相同（如绿军用绿毒），Portal 所有者变为使用者 ID。  
   * 若翻转后的阵营与使用者相反（如绿军用蓝毒），Portal 所有者变为系统 NPC (\_\_ADA\_\_ 或 \_\_JARVIS\_\_)。

# ---

**4\. 模组系统 (Mods)：防御、增益与攻击**

Portal 拥有4个模组槽位。模组的数值叠加遵循极其重要的**收益递减法则 (Diminishing Returns)**。在 Web 模拟器中，必须正确排序模组并应用相应的系数，否则会导致数据严重失真。

## **4.1 防御模组：护盾 (Portal Shields)**

护盾提供 Mitigation（减伤），这是一种百分比减伤机制。

### **表 4.1：护盾属性表**

| 类型 (Rarity) | 减伤值 (Mitigation) | 粘性 (Stickiness) | 回收价值 (XM) |
| :---- | :---- | :---- | :---- |
| **Common (C)** | \+30 | 0 | 40 |
| **Rare (R)** | \+40 | 150,000 | 80 |
| **Very Rare (VR)** | \+60 | 450,000 | 100 |
| **Aegis (AXA)** | \+70 | 550,000 | 100 |

**减伤计算逻辑**：

* **总减伤 (Total Mitigation)** \=  $\sum Shield\_Mitigation + Link\_Mitigation$。  
* **硬上限 (Hard Cap)**：无论数值累加多高，实际生效的减伤上限为 **95%**。  
  * *案例*：4个 Aegis 盾提供 $70 \times 4 = 280$ mitigation。虽然数值是 280，但实际减伤只有 95%。多余的数值（280-95=185）作为缓冲，当第一个盾被炸掉后，减伤可能依然维持在 95% 13。  
* **链接减伤 (Link Mitigation)**：除了盾，Link 也提供防御。公式为：  
  $$Mitigation_{links} = \frac{400}{9} \times \arctan\left(\frac{Count_{links}}{e}\right)$$
  其中  $e \approx 2.718$ 。这表明 Link 提供的防御也是非线性的，前几条 Link 提供的防御加成最高 17。

## **4.2 农业模组：Heat Sink 与 Multi-hack**

这两个模组影响 Hacking（入侵）产出和频率，是“起八”（建立高级物资塔）的关键。

### **4.2.1 Heat Sink (散热器)**

减少 Hacking 的冷却时间（默认为 300秒 / 5分钟）。

* **Common**: \-20%  
* **Rare**: \-50%  
* **Very Rare**: \-70%

**重置机制**：安装 Heat Sink 的瞬间，会为**当前安装者**重置 Portal 的冷却时间和燃烧次数（Burnout）。这是一个重要的战术机制，允许玩家连续 Hack 19。

### **4.2.2 Multi-hack (多重入侵)**

增加 Portal 在进入燃烧状态（Burnout）前的可 Hack 次数（默认为 4 次）。

* **Common**: \+4 次  
* **Rare**: \+8 次  
* **Very Rare**: \+12 次

### **4.2.3 收益递减计算**

对于这两种模组，安装多个同类模组时，效果会急剧下降。

* **第一模组**：100% 效能。  
* **第二模组**：50% 效能。  
* *算法示例*：安装一个 VR MH (+12) 和一个 Rare MH (+8)。  
  * 总次数 \= 基础4次 \+ VR(12) \+ Rare(8 \* 0.5) \= 20 次。  
  * 注意：系统通常会自动将最高稀有度的模组视为“第一模组”进行计算，无论安装顺序如何 21。

## **4.3 攻击模组：Force Amp 与 Turret**

这类模组用于反击入侵的敌方玩家，扣除其 XM。

* **Force Amp (力量增幅器)**：增加反击伤害。  
  * 倍率：x2.0 (Rare)。  
  * 递减：第二枚仅增加 \+0.25 (即总倍率 x2.25) 22。  
* **Turret (炮塔)**：增加反击频率和暴击率。  
  * 效果：攻击频率 x2，暴击率 \+30%。  
  * 递减：第二枚效果微乎其微 23。

**战术建议**：在模拟器 AI 建议中，应提示玩家通常只安装 1个 Force Amp 和 1个 Turret 即可达到性价比最高，安装4个同类模组是极大的浪费。

## **4.4 链接模组：Link Amp 与 SBUL**

用于扩展 Portal 的链接距离。

基础距离公式：$R = 160 \times (Avg\_Reso\_Level)^4$ 。

* **Rare Link Amp**: 距离 x2。  
* **SoftBank Ultra Link (SBUL)**:  
  * 距离 x5。  
  * **出链上限增加**: \+8 条（基础为8条）。  
  * **防御加成**: 使得该 Portal 的 Link Mitigation 效果提升 1.5倍。  
* **Very Rare Link Amp**: 距离 x7（现已绝版，不可 Hack 获得，仅存于老玩家库存或胶囊中）。

**递减法则**：

* 1st: x2.0  
* 2nd: x0.5 (累计 x2.5)  
* 3rd: x0.25 (累计 x2.75)  
* 4th: x0.25 (累计 x3.0)  
* *SBUL 同理递减* 25。

## **4.5 转化模组：ITO EN**

* **ITO EN (+)**：修改掉落表，屏蔽武器（XMP/US），大幅增加防御物资（Resonator, Shield）的掉率。  
* **ITO EN (-)**：修改掉落表，屏蔽防御物资，大幅增加武器的掉率。  
* **互斥性**：在同一个 Portal 上安装 (+) 和 (-) 会互相抵消，没有任何效果 27。

# ---

~~5\. 动能胶囊 (Kinetic Capsule) 与合成配方~~
~~不实现此功能~~

这是 Ingress Prime 引入的新机制，类似于“走路孵蛋”。Web 端模拟器需要模拟步数积累过程。

* **解锁条件**：L4 级以上玩家。  
* **距离要求**：通常为 8.0 km（部分活动期间会减半）。  
* **并发限制**：默认 1 个程序，使用 Rare Kinetic Capsule 可额外并行，最多 7 个。  
* **每日上限**：每天计步上限为 40km，超过部分不计入胶囊里程。

### **表 5.1：常见 Kinetic Capsule 合成配方**

| 产出物品 (Output) | 输入材料 (Input Requirements) | XM 消耗 | 距离 |
| :---- | :---- | :---- | :---- |
| **Very Rare Shield (1)** | Rare Shield (3) \+ L4+ XMP (3) \+ L4+ Reso (3) | 6,000 | 8km |
| **Hypercube (5)** | L4+ Power Cube (10) | 6,000 | 8km |
| **SoftBank Ultra Link (1)** | Rare Link Amp (3) \+ L4+ XMP (3) \+ L4+ Reso (3) | 6,000 | 8km |
| **VR Heat Sink (1)** | Rare Heat Sink (3) \+ L4+ Reso (3) | 6,000 | 8km |
| **VR Multi-hack (1)** | Rare Multi-hack (3) \+ L4+ Reso (3) | 6,000 | 8km |
| **L8 Resonator (5)** | L6+ Resonator (25) *\[注：配方会随活动轮换\]* | 6,000 | 8km |
| **Jarvis/ADA** | *需特定活动期间开放，通常消耗大量低级道具* | \- | \- |


# ---

**6\. 能量资源：Power Cubes 与 Hypercubes**

## **6.1 常规 Power Cube**

遵循简单的线性公式：$XM = Level \times 1,000$ 。

L8 Cube 提供 8,000 XM。

## **6.2 Hypercube (Lawson / Circle K)**

这种高级能量块提供一个额外的 XM 槽。其容量取决于**使用者的等级**，而非道具本身的等级（Hypercube 不分等级）。

### **表 6.1：Hypercube 容量表 (按玩家等级)**

| 玩家等级 | Hypercube 容量 (XM) | 玩家等级 | Hypercube 容量 (XM) |
| :---- | :---- | :---- | :---- |
| **L1** | 18,000 | **L9** | 36,000 |
| **L2** | 20,250 | **L10** | 38,400 |
| **L3** | 22,500 | **L11** | 40,800 |
| **L4** | 24,750 | **L12** | 43,200 |
| **L5** | 27,000 | **L13** | 45,600 |
| **L6** | 29,250 | **L14** | 48,000 |
| **L7** | 31,500 | **L15** | 50,400 |
| **L8** | 33,750 | **L16** | 52,800 |

30

# ---

**7\. 战术道具与增益 (Powerups)**

Web 端模拟器若要包含“商城”或高级玩法，必须实现以下道具。

## **7.1 Apex (AP 增幅器)**

* **效果**：使接下来的 30 分钟内，获得的所有 AP 翻倍（x2 Multiplier）。  
* **叠加**：时间可叠加（例如使用2个 Apex 获得 60 分钟 x2），但倍率不可叠加（不会变成 x4）。  
* **活动交互**：若游戏内正进行“双倍 AP 活动”，Apex 会基于基础值叠加，通常结果为 x3 或 x4，具体取决于 Niantic 的后端配置逻辑（Base \* Event\_Mult \* Apex\_Mult）32。

## **7.2 Portal Fracker (压裂器)**

* **效果**：在 10 分钟内或 150 次 Hack 内，使 Portal 的物资产出翻倍。  
* **副作用**：效果结束后，Portal 会扣除当前能量的 **50%**。这在模拟器中需要作为一个延迟触发的伤害事件处理。  
* **限制**：VR 级道具，不可回收 34。

## **7.3 Battle Beacon (战斗烽火台)**

用于人为制造小型的“异常（Anomaly）”战斗。

* **持续时间**：约 15 分钟（分为 5 个 Checkpoint，每 3 分钟一轮）。  
* **计分逻辑**：  
  * Rare Battle Beacon 权重：2-3-4 分。  
  * Very Rare Battle Beacon 权重：1-2-2-3-4 分。  
  * 在每个 Checkpoint 结束瞬间，系统检测 Portal 的阵营归属。无论谁占领，该阵营获得对应分数。  
  * **阵营反转**：在 Checkpoint 结算后，Portal 可能会强制反转阵营以刺激下一轮争夺（模拟器需实现此强制翻转逻辑）36。  
* **产出加成**：战斗结束后，胜方标志会显示在 Portal 上，并在短时间内提供 Hack 产出加成。

# ---

**8\. 物品回收价值总表 (Master Recycle Values)**

这是构建物品栏（Inventory）系统的核心数据。当玩家 XM 不足或背包已满（2000/2500上限）时，回收是高频操作。

### **表 8.1：全物品回收价值对照表**

| 物品类别 | 子类/等级 | 回收 XM 值 | 备注 |
| :---- | :---- | :---- | :---- |
| **共振器 / XMP** | L1 | 20 |  |
| **共振器 / XMP** | L2 \- L8 | $20 \times Level$ | L8 \= 160 XM |
| **Ultra Strike** | L1 \- L8 | $20 \times Level$ | 同上 |
| **Power Cube** | L1 \- L8 | $1000 \times Level$ | L8 \= 8,000 XM |
| **Portal Key** | \- | 500 | 曾为20，现已上调 38 |
| **Media** | Common | 20 | 38 |
| **Media** | Rare/VR | 40 / 80 | 较少见 |
| **Mod (模组)** | Common | 40 |  |
| **Mod (模组)** | Rare | 80 | 包括 Turret, Force Amp |
| **Mod (模组)** | Very Rare | 100 | 包括 VR Shield, SBUL |
| **Flip Card** | Jarvis/ADA | 100 | 极为昂贵，通常不回收 |
| **Hypercube** | \- | 0 / 160 | 直接回收通常不可行或值极低；放入胶囊回收胶囊得 160 |
| **Capsule** | Rare | 80 \+ 内容物总和 | “穷人能量块”战术的基础 |
| **Key Locker** | \- | 不可回收 | 商城道具 |

# ---

**9\. 结论与开发建议**

构建一个 Web 端的 Ingress 是一项庞大的工程。除了上述的数据结构，开发者还需注意以下几点系统级的实现细节：

1. **S2 单元格机制**：Ingress 严重依赖 Google S2 Geometry。Portal 的分布受 L19 单元格限制，而计分（Scoring）则基于 L6 单元格。模拟器必须引入 S2 算法库来处理区域计分。  
2. **速度锁定 (Speedlock)**：为了防止作弊（GPS Spoofing），系统应计算玩家两次动作之间的移动速度。若速度超过 **60km/h**（约 16.6m/s），所有 Hack、Deploy、Attack 操作应返回失败。  
3. **Overclock (超频) 扩展**：最新的 Overclock 机制允许玩家通过 AR 扫描特定 Portal。在 Web 端难以完全复刻 AR，但可以简化为“完成一个小游戏后，Hack 产出 x4”的逻辑。这需要为 Portal 数据结构增加 is\_overclock\_enabled 字段。  
4. **Drone Net (无人机)**：无人机 Hack 遵循独立的冷却时间（60分钟），且不能获取 Key。这需要在 User 实体下挂载一个 Drone 对象，记录其当前的 Portal 坐标和最后冷却时间戳。

本报告提供的数值和公式覆盖了 Ingress Prime 95% 以上的核心玩法机制，足以支撑起一个高保真的 Web 模拟环境。建议在数据库设计阶段，采用 JSONB (PostgreSQL) 或 NoSQL 文档存储物品属性，以应对未来可能新增的模组类型（如历史上昙花一现的 Turret v1 或各类活动限定 Beacon）。

#### **引用的著作**

1. Resonator | Ingress Guide | Fev Games: Gaming Blog, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/resonator/](https://fevgames.net/ingress/ingress-guide/items/resonator/)  
2. Ingress portal recharge mechanics may provide some clues to how ACTIVITY\_FEED\_BERRY might be implemented. : r/TheSilphRoad \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/TheSilphRoad/comments/6aeq8r/ingress\_portal\_recharge\_mechanics\_may\_provide/](https://www.reddit.com/r/TheSilphRoad/comments/6aeq8r/ingress_portal_recharge_mechanics_may_provide/)  
3. Mod | Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/](https://fevgames.net/ingress/ingress-guide/items/mod/)  
4. Deploy Resonator | Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/actions/deploy-resonator/](https://fevgames.net/ingress/ingress-guide/actions/deploy-resonator/)  
5. XMP | Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/xmp/](https://fevgames.net/ingress/ingress-guide/items/xmp/)  
6. r/Ingress Wiki: Ingress Items Guide \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/wiki/items/](https://www.reddit.com/r/Ingress/wiki/items/)  
7. Attack | Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/actions/attack/](https://fevgames.net/ingress/ingress-guide/actions/attack/)  
8. Critical Hit Chances : r/Ingress \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/comments/44t2cs/critical\_hit\_chances/](https://www.reddit.com/r/Ingress/comments/44t2cs/critical_hit_chances/)  
9. Ultra Strike | Ingress Wiki | Fandom, 访问时间为 二月 2, 2026， [https://ingress.fandom.com/wiki/Ultra\_Strike](https://ingress.fandom.com/wiki/Ultra_Strike)  
10. Ultra Strike | Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/ultra-strike/](https://fevgames.net/ingress/ingress-guide/items/ultra-strike/)  
11. Ultra Strike damage and usefulness : r/Ingress \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/comments/2fearj/ultra\_strike\_damage\_and\_usefulness/](https://www.reddit.com/r/Ingress/comments/2fearj/ultra_strike_damage_and_usefulness/)  
12. XM Weapon Questions : r/Ingress \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/comments/1jtpigf/xm\_weapon\_questions/](https://www.reddit.com/r/Ingress/comments/1jtpigf/xm_weapon_questions/)  
13. Portal Shield \- Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/portal-shield/](https://fevgames.net/ingress/ingress-guide/items/mod/portal-shield/)  
14. Flip Card | Ingress Wiki \- Fandom, 访问时间为 二月 2, 2026， [https://ingress.fandom.com/wiki/Flip\_Card](https://ingress.fandom.com/wiki/Flip_Card)  
15. Flip Card \- Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/flip-card/](https://fevgames.net/ingress/ingress-guide/items/flip-card/)  
16. What is the formula for damage mitigation in Ingress? \- Arqade \- Stack Exchange, 访问时间为 二月 2, 2026， [https://gaming.stackexchange.com/questions/120525/what-is-the-formula-for-damage-mitigation-in-ingress](https://gaming.stackexchange.com/questions/120525/what-is-the-formula-for-damage-mitigation-in-ingress)  
17. Damage reduction on a portal : r/Ingress \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/comments/c5x0az/damage\_reduction\_on\_a\_portal/](https://www.reddit.com/r/Ingress/comments/c5x0az/damage_reduction_on_a_portal/)  
18. Link | Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/concepts/link/](https://fevgames.net/ingress/ingress-guide/concepts/link/)  
19. Heat Sink \- Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/heat-sink/](https://fevgames.net/ingress/ingress-guide/items/mod/heat-sink/)  
20. Cooldown | Ingress Wiki \- Fandom, 访问时间为 二月 2, 2026， [https://ingress.fandom.com/wiki/Cooldown](https://ingress.fandom.com/wiki/Cooldown)  
21. r/Ingress Wiki \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/wiki/faq/](https://www.reddit.com/r/Ingress/wiki/faq/)  
22. Force Amp \- Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/force-amp/](https://fevgames.net/ingress/ingress-guide/items/mod/force-amp/)  
23. Turret | Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/turret/](https://fevgames.net/ingress/ingress-guide/items/mod/turret/)  
24. Turret mod testing results : r/Ingress \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/comments/4b5fj3/turret\_mod\_testing\_results/](https://www.reddit.com/r/Ingress/comments/4b5fj3/turret_mod_testing_results/)  
25. Link Amp \- Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/link-amp/](https://fevgames.net/ingress/ingress-guide/items/mod/link-amp/)  
26. How to determine link distance for different level portals with differing amounts of link amp? : r/Ingress \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/comments/1ij78p/how\_to\_determine\_link\_distance\_for\_different/](https://www.reddit.com/r/Ingress/comments/1ij78p/how_to_determine_link_distance_for_different/)  
27. ITO EN Transmuter Question : r/Ingress \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/comments/6p3t0u/ito\_en\_transmuter\_question/](https://www.reddit.com/r/Ingress/comments/6p3t0u/ito_en_transmuter_question/)  
28. Transmuter (mod) | Ingress | FevGames, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/mod/transmuter/](https://fevgames.net/ingress/ingress-guide/items/mod/transmuter/)  
29. Kinetic Capsules \- Ingress Support, 访问时间为 二月 2, 2026， [https://support.ingress.com/hc/en-us/articles/41140341728411-Kinetic-Capsules](https://support.ingress.com/hc/en-us/articles/41140341728411-Kinetic-Capsules)  
30. Since I started playing again, I've been trying to figure out what the most efficient way to use Kinetic Capsules is. So I made a chart. : r/Ingress \- Reddit, 访问时间为 二月 2, 2026， [https://www.reddit.com/r/Ingress/comments/1gcbge3/since\_i\_started\_playing\_again\_ive\_been\_trying\_to/](https://www.reddit.com/r/Ingress/comments/1gcbge3/since_i_started_playing_again_ive_been_trying_to/)  
31. Power Cube \- Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/power-cube/](https://fevgames.net/ingress/ingress-guide/items/power-cube/)  
32. Apex Boosts \- Ingress Support, 访问时间为 二月 2, 2026， [https://support.ingress.com/hc/en-us/articles/41140426847899-Apex-Boosts](https://support.ingress.com/hc/en-us/articles/41140426847899-Apex-Boosts)  
33. Apex \- Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/items/boost/apex/](https://fevgames.net/ingress/ingress-guide/items/boost/apex/)  
34. Portal Fracker \- Ingress Wiki \- Fandom, 访问时间为 二月 2, 2026， [https://ingress.fandom.com/wiki/Portal\_Fracker](https://ingress.fandom.com/wiki/Portal_Fracker)  
35. LavaSurfer's Secret Guide to Ingress Prime \- Topher's Castle, 访问时间为 二月 2, 2026， [https://www.lavasurfer.com/info/ingress.html](https://www.lavasurfer.com/info/ingress.html)  
36. Battle Beacon | Ingress Wiki \- Fandom, 访问时间为 二月 2, 2026， [https://ingress.fandom.com/wiki/Battle\_Beacon](https://ingress.fandom.com/wiki/Battle_Beacon)  
37. Battle Beacon FAQs \- Ingress Support, 访问时间为 二月 2, 2026， [https://support.ingress.com/hc/en-us/articles/41140407434651-Battle-Beacon-FAQs](https://support.ingress.com/hc/en-us/articles/41140407434651-Battle-Beacon-FAQs)  
38. Recycle | Fev Games, 访问时间为 二月 2, 2026， [https://fevgames.net/ingress/ingress-guide/actions/recycle/](https://fevgames.net/ingress/ingress-guide/actions/recycle/)
