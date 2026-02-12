# OpenIngress RTS 技术设计文档（实现对齐版）

> 更新时间：2026-02  
> 本文档描述当前代码实现，不再使用历史设计稿中的旧消息名（如 `PLAYER_UPDATE`）。

## 1. 设计目标

- 后端以 Go 实现实时游戏主循环，内存态作为权威状态。
- HTTP 负责鉴权与管理接口；WebSocket 负责实时交互与广播。
- 持久化采用异步增量落库，优先保证在线链路吞吐与稳定性。
- 前端以 React + Leaflet 呈现地图，按 WS 协议进行状态渲染与动画预渲染。

---

## 2. 技术栈与模块

### 2.1 前端

- React（组件与状态管理）
- Leaflet（地图渲染）
- WebSocket（实时通信）

### 2.2 后端

- Go 1.24+
- Gin（HTTP）
- Gorilla WebSocket（WS）
- GORM + PostgreSQL（持久化）

### 2.3 核心目录

- `backend/pkg/api`: HTTP 与认证
- `backend/pkg/ws`: WS 协议、路由、Hub、广播
- `backend/pkg/service`: 游戏业务逻辑
- `backend/pkg/state`: 内存状态与仓库
- `backend/pkg/database`: DB 模型、快照映射、增量写入
- `frontend/src`: 地图 UI、弹窗、通知层

---

## 3. 运行时架构

### 3.1 关键组件

- `HTTPServer`: HTTP API 服务。
- `WSServer/Hub`: WS 连接管理、消息路由、tick 调度。
- `GameState`: 玩家/Portal/Link/Field/Log 的内存权威状态。
- `Service Container`: `player/mapsvc/portal/hack/link/field/combat` 业务服务。
- `PersistenceManager`: 异步持久化调度（队列容量 1，latest-wins）。
- `DeltaWriter`: 快照差异写入 PostgreSQL。

### 3.2 主流程

1. 客户端通过 HTTP 登录拿到 JWT。
2. WS `CONNECT` 时携带 `authToken`，服务端校验并绑定 `playerId`。
3. 连接成功返回 `CONNECTED`，随后立即下发 `PLAYER_STATE`（首帧权威位置）。
4. Hub 按 tick 执行：
   - 移动推进（movement tick）
   - 附近玩家与自身状态推送
   - 自动 Hack 尝试
   - 地图增量推送（map tick）
5. 持久化协程按周期入队最新快照并异步落库。

---

## 4. WebSocket 协议

### 4.1 统一消息结构

```json
{
  "type": "MESSAGE_TYPE",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {}
}
```

### 4.2 当前消息类型（实现中）

| 类型 | 方向 | 说明 |
|---|---|---|
| CONNECT | Client→Server | 建立会话（JWT） |
| CONNECTED | Server→Client | 连接成功 |
| PLAYER_TARGET_UPDATE | Client→Server | 设置移动目标点 |
| PLAYER_LOCAL_UPDATE | Client→Server | 已废弃，返回 ERROR |
| PLAYER_VIEW_UPDATE | Client→Server | 上报视野（bounds/tiles） |
| PLAYER_STATE | Server→Client | 玩家权威状态（含 render/preRender） |
| NEARBY_PLAYERS | Server→Client | 附近玩家快照 |
| MAP_UPDATE | Server→Client | 地图全量/批量更新 |
| MAP_TICK | Server→Client | 地图 tick 增量更新 |
| HACK_PORTAL | Client→Server | 手动 Hack |
| HACK_RESULT | Server→Client | 手动 Hack 结果 |
| PLAYER_TOGGLE_AUTO_HACK | Client→Server | 自动 Hack 开关 |
| AUTO_HACK_RESULT | Server→Client | 自动 Hack 结果 |
| PLAYER_DEPLOY_RESONATOR | Client→Server | 放置/升级脚 |
| PLAYER_DEPLOY_MOD | Client→Server | 安装 Mod |
| PLAYER_CHARGE_PORTAL | Client→Server | 充能 |
| PLAYER_CREATE_LINK | Client→Server | 建链 |
| PLAYER_ATTACK | Client→Server | 攻击 |
| PORTAL_UPDATE | Server→Client | 门户补丁更新（按操作携带不同字段） |
| LINK_UPDATE | Server→Client | Link 更新/移除 |
| FIELD_CREATED | Server→Client | Field 创建/移除 |
| ATTACK_RESULT | Server→Client | 攻击结果 |
| ERROR | Server→Client | 错误回包 |
| PING/PONG | 双向 | 心跳 |

### 4.3 关键 payload 约定

#### PLAYER_ATTACK / ATTACK_RESULT（2026-02）

- `PLAYER_ATTACK.data` 支持可选 `chargeBonus`（0.0~0.2，服务端 clamp）。
- `ATTACK_RESULT.data` 在保留旧字段基础上新增可选：
  - `mitigationApplied`
  - `modsDestroyed[]`
  - `counterattackTriggered` / `counterattackDamage`
  - `portalDamages[]`
- 攻击后同 `id` 还会返回 `PLAYER_RESOURCE_UPDATE`（`xmDelta`、`inventoryDelta`、`apGained`）。

#### CONNECT 成功后首帧

```json
{
  "type": "PLAYER_STATE",
  "data": {
    "playerId": "player-uuid",
    "latitude": 39.9123,
    "longitude": 116.3812,
    "renderLatitude": 39.9123,
    "renderLongitude": 116.3812,
    "preRenderLatitude": 39.9126,
    "preRenderLongitude": 116.3815,
    "speedMps": 33.3333,
    "headingDeg": 74.2
  }
}
```

- `render*` 用于当前渲染位置。
- `preRender*` 用于前端预渲染平滑移动。

#### MAP_UPDATE / MAP_TICK

```json
{
  "type": "MAP_UPDATE",
  "data": {
    "full": true,
    "batchIndex": 1,
    "batchTotal": 1,
    "portals": [
      {
        "id": "portal-uuid",
        "title": "Portal Name",
        "cover_url": "https://example.com/portal-cover.jpg",
        "latitude": 39.91,
        "longitude": 116.38,
        "faction": "RESISTANCE",
        "level": 3,
        "energy": 4500
      }
    ]
  }
}
```

- 地图更新按批次发送，当前批大小 `200`。
- `MAP_TICK` 与 `MAP_UPDATE` 使用同结构，语义不同（定时增量 vs 视野触发）。

#### PORTAL_UPDATE（放置脚）

```json
{
  "type": "PORTAL_UPDATE",
  "data": {
    "portalId": "portal-uuid",
    "playerId": "player-uuid",
    "slot": 1,
    "level": 1,
    "energy": 1000,
    "slotEnergy": 1000,
    "faction": "RESISTANCE",
    "portalLevel": 1,
    "version": 1,
    "apGained": 625
  }
}
```

- `apGained` 为可选字段，仅在本次操作产生 AP 奖励时出现。

#### HACK_RESULT / AUTO_HACK_RESULT

```json
{
  "type": "AUTO_HACK_RESULT",
  "data": {
    "portalId": "portal-uuid",
    "success": true,
    "cooldownSeconds": 300,
    "itemsGained": ["XMP_L1", "RESO_L1", "KEY:portal-uuid"],
    "apGained": 50
  }
}
```

---

## 5. 前端实时渲染约定

- 新开局后端会先推送 `PLAYER_STATE`，前端据此居中与放大地图。
- 前端接收 `PLAYER_STATE.preRenderLatitude/preRenderLongitude` 做移动预渲染，减少“顿挫跳点”。
- `AUTO_HACK_RESULT` 用于通知层（右上角物品 + 顶部 AP），不写入前端 `Global Log`。
- 任意 WS 消息只要 `data.apGained > 0`，前端可触发 `+xxx AP` 飘字。

---

## 6. 数据模型

### 6.1 领域模型（内存态）

- `Player`: 位置、目标点、视野、库存、冷却、自动 Hack。
- `Portal`: `id/title/cover_url`、等级、阵营、总能量、`resonators`、`mods`。
- `Link`: 起终点 Portal 与坐标。
- `Field`: 三点 Portal、MU、层级、阵营。
- `LogEntry`: 全局日志项。

### 6.2 并发安全

- `InMemoryPlayerRepo` 与 `InMemoryPortalRepo` 在 `Get/List/Upsert` 做深拷贝，避免 map/slice/pointer 别名造成并发读写 panic。
- `GameState.Snapshot()` 与 `NewGameStateFromSnapshot()` 对 `AdminState` 做克隆，避免共享 map。

---

## 7. 数据库 Schema（当前实现）

> 以下为当前 GORM 模型对齐字段，来源：`backend/pkg/database/models.go`。

### 7.1 players

```sql
CREATE TABLE players (
  id VARCHAR(64) PRIMARY KEY,
  username VARCHAR(128) NOT NULL DEFAULT '',
  faction VARCHAR(32) NOT NULL DEFAULT 'NEUTRAL',
  level INT NOT NULL DEFAULT 1,
  ap INT NOT NULL DEFAULT 0,
  xm INT NOT NULL DEFAULT 0,
  max_xm INT NOT NULL DEFAULT 0,
  portal_captures INT NOT NULL DEFAULT 0,
  links_created INT NOT NULL DEFAULT 0,
  fields_created INT NOT NULL DEFAULT 0,
  mu_total INT NOT NULL DEFAULT 0,
  pos_lat DOUBLE PRECISION NOT NULL DEFAULT 0,
  pos_lon DOUBLE PRECISION NOT NULL DEFAULT 0,
  auto_hack BOOLEAN NOT NULL DEFAULT FALSE,
  inventory_json JSONB,
  cooldowns_json JSONB,
  view_json JSONB,
  view_tiles_json JSONB,
  target_json JSONB,
  updated_at TIMESTAMP NOT NULL
);
```

### 7.2 player_inventories

```sql
CREATE TABLE player_inventories (
  player_id VARCHAR(64) NOT NULL,
  item_id VARCHAR(64) NOT NULL,
  amount INT NOT NULL DEFAULT 0,
  PRIMARY KEY (player_id, item_id)
);
```

### 7.3 portals

```sql
CREATE TABLE portals (
  id VARCHAR(64) PRIMARY KEY,
  title VARCHAR(255) NOT NULL DEFAULT '',
  cover_url TEXT NOT NULL DEFAULT '',
  latitude DOUBLE PRECISION NOT NULL,
  longitude DOUBLE PRECISION NOT NULL,
  faction VARCHAR(32) NOT NULL DEFAULT 'NEUTRAL',
  level INT NOT NULL DEFAULT 0,
  energy INT NOT NULL DEFAULT 0,
  resonators_json JSONB,
  mods_json JSONB,
  updated_at TIMESTAMP NOT NULL
);
```

### 7.4 links / fields / logs

```sql
CREATE TABLE links (
  id VARCHAR(64) PRIMARY KEY,
  from_portal_id VARCHAR(64) NOT NULL,
  to_portal_id VARCHAR(64) NOT NULL,
  from_lat DOUBLE PRECISION NOT NULL,
  from_lon DOUBLE PRECISION NOT NULL,
  to_lat DOUBLE PRECISION NOT NULL,
  to_lon DOUBLE PRECISION NOT NULL,
  created_at TIMESTAMP NOT NULL
);

CREATE TABLE fields (
  id VARCHAR(64) PRIMARY KEY,
  portal_1_id VARCHAR(64) NOT NULL,
  portal_2_id VARCHAR(64) NOT NULL,
  portal_3_id VARCHAR(64) NOT NULL,
  faction VARCHAR(32) NOT NULL DEFAULT 'NEUTRAL',
  mu INT NOT NULL DEFAULT 0,
  layer INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL
);

CREATE TABLE logs (
  id VARCHAR(64) PRIMARY KEY,
  type VARCHAR(32) NOT NULL,
  player_id VARCHAR(64),
  portal_id VARCHAR(64),
  faction VARCHAR(32),
  mu INT NOT NULL DEFAULT 0,
  message TEXT NOT NULL DEFAULT '',
  timestamp TIMESTAMP NOT NULL
);
```

### 7.5 admin_bans / auth_users

```sql
CREATE TABLE admin_bans (
  player_id VARCHAR(64) PRIMARY KEY,
  reason TEXT NOT NULL DEFAULT '',
  updated_at TIMESTAMP NOT NULL
);

CREATE TABLE auth_users (
  username_norm VARCHAR(128) PRIMARY KEY,
  username VARCHAR(128) NOT NULL,
  password_hash TEXT NOT NULL,
  player_id VARCHAR(64) NOT NULL UNIQUE,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL
);
```

---

## 8. 持久化策略

- 运行中：`PersistenceManager.EnqueueLatest(snapshot)` 周期入队。
- 队列容量为 `1`，满时丢弃旧任务保留最新快照（latest-wins）。
- `DeltaWriter.SaveSnapshot` 对 `players/player_inventories/portals/links/fields/logs/admin_bans/auth_users` 做增量 upsert/delete。
- 关机：`FlushBestEffort` 在超时内尝试最后一次落库。

---

## 9. 规则与常量（当前实现）

- 玩家移动速度：`120 km/h`（`player.MoveSpeedKmH`）。
- 默认交互范围：`40m`（`player.InteractRange` fallback）。
- Hack 距离：`40m`；Hack 冷却：`300s`。
- WS 默认节奏：
  - movement tick: `100ms`
  - map tick: `1000ms`
  - player state push: `300ms`
  - nearby players push: `600ms`
- 部署脚 AP：
  - 空槽部署：`+125`
  - 首占中立 Portal：额外 `+500`
  - 升级已有脚：`+65`
- Hack 成功 AP：`+50`。

---

## 10. 文档一致性要求

- 协议字段改动时，必须同步更新：
  - `WIKI/api.md`
  - `backend/pkg/ws/router_test.go`
  - 本文档（`WIKI/tech-design.md`）
- 本文档用于“架构与约定”，字段级别细节以 `WIKI/api.md` 为准。
