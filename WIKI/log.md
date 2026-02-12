第一部分：日志操作分析 (5种核心核心操作)
根据你提供的日志片段，系统中主要发生了以下 5 类核心操作。这些操作之间存在严格的依赖关系和状态流转。

1. 占领与部署 (Deploy / Capture)
这是游戏的基础建设。

关键字: captured, deployed a Resonator

含义: 玩家将 resonators（谐振器）放入 Portal（据点）。

逻辑:

如果 Portal 是中立的，第一个放入 Resonator 的人触发 captured。

一个 Portal 最多有 8 个 Resonator 槽位。

示例:

deployed a Resonator on Ming Water Chalice captured Ming Water Chalice

2. 连线 (Link)
这是建立网络的过程。

关键字: linked from ... to ...

含义: 在两个 Portal 之间建立一条能量线。

依赖条件:

两个 Portal 都必须装满 8 个 Resonator。

玩家必须持有目标 Portal 的 Key（道具）。

这条线不能与任何现存的线交叉（判定线段相交）。

示例:

linked from 石墩遗迹 to 黄花阵南门柱

3. 建立控制域 (Create Control Field)
这是游戏的得分机制。

关键字: created a Control Field

含义: 当三条 Link 组成一个闭合三角形时，会形成一个 Field。

逻辑:

这是 Link 操作的副作用（Side Effect）。

系统需计算三角形覆盖的地理面积或人口密度（MU - Mind Units）。

示例:

created a Control Field @石墩遗迹 +3 MUs

4. 攻击与摧毁 (Attack / Destroy)
这是对抗机制，会触发连锁反应。

关键字: destroyed a Resonator, destroyed the Link, destroyed the Control Field

含义: 攻击敌方 Portal。

连锁逻辑 (Chain Reaction):

Resonator 血量归零 -> destroyed a Resonator.

如果 Resonator 数量少于特定值（通常是少于3个）或关键 Resonator 被毁 -> 依赖它的 Link 断裂 -> destroyed the Link.

如果组成三角形的任意一条 Link 断裂 -> Field 消失 -> destroyed the Control Field.

示例:

destroyed the Control Field @水磨社区 -52 MUs (因果链的末端)

5. 系统预警 (Alert)
关键字: under attack

含义: 当玩家的 Portal 受到攻击时，系统向拥有者发送的推送。

示例:

Your Portal 融科融智创新园东北门 is under attack by Cuplordbot


