# OpenIngress RTS API 文档（请求值 / 返回值 / Demo）

> 目标：提供可直接对照实现的接口文档。覆盖 HTTP + WebSocket，包含请求字段、响应字段与示例。
> 文档状态标注：已实现 / 规划中 / 已废弃。

## 1. 通用约定

### 1.1 HTTP 响应结构（统一）

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {}
}
```

### 1.2 WebSocket 消息结构（统一）

```json
{
  "type": "MESSAGE_TYPE",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {}
}
```

### 1.3 错误码表

| ErrCode | 含义 |
|---|---|
| 0 | success |
| 1001 | bad request |
| 1002 | internal error |
| 1003 | not found |
| 1004 | unauthorized |
| 1005 | conflict |

HTTP 错误示例：

```json
{
  "errno": 1004,
  "errmsg": "invalid token",
  "data": null
}
```

WS 错误示例：

```json
{
  "type": "ERROR",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "code": 1001,
    "message": "invalid request"
  }
}
```

### 1.4 通用数据结构

**Position**

```json
{
  "latitude": 39.9123,
  "longitude": 116.3812
}
```

**Bounds**

```json
{
  "minLat": 39.90,
  "maxLat": 39.92,
  "minLon": 116.37,
  "maxLon": 116.39
}
```

**Player**

```json
{
  "id": "player-uuid",
  "username": "agent",
  "faction": "RESISTANCE",
  "level": 3,
  "ap": 1200,
  "xm": 2500,
  "maxXm": 3000,
  "inventory": {
    "XMP_1": 10,
    "CUBE": 3
  },
  "position": {"latitude": 39.91, "longitude": 116.38},
  "target": {"latitude": 39.915, "longitude": 116.385},
  "view": {"minLat": 39.90, "maxLat": 39.92, "minLon": 116.37, "maxLon": 116.39},
  "viewTiles": ["tile-1", "tile-2"],
  "autoHack": false,
  "hackCooldowns": {"portal-uuid": "2026-02-03T10:00:00Z"},
  "updatedAt": "2026-02-03T10:00:00Z"
}
```

**Portal**

```json
{
  "id": "portal-uuid",
  "position": {"latitude": 39.91, "longitude": 116.38},
  "faction": "RESISTANCE",
  "level": 3,
  "energy": 4500,
  "resonators": {
    "1": {"slot": 1, "level": 3, "energy": 1500, "playerId": "player-uuid", "version": 2, "updatedAt": "2026-02-03T10:00:00Z"}
  },
  "mods": {
    "1": {"slot": 1, "modType": "LINK_AMP", "rarity": "RARE", "playerId": "player-uuid", "version": 1, "updatedAt": "2026-02-03T10:00:00Z"}
  },
  "updatedAt": "2026-02-03T10:00:00Z"
}
```

**Link**

```json
{
  "id": "link-uuid",
  "fromPortalId": "portal-a",
  "toPortalId": "portal-b",
  "fromPosition": {"latitude": 39.91, "longitude": 116.38},
  "toPosition": {"latitude": 39.92, "longitude": 116.39},
  "createdAt": "2026-02-03T10:00:00Z"
}
```

**Field**

```json
{
  "id": "field-uuid",
  "portalIds": ["portal-a", "portal-b", "portal-c"],
  "faction": "RESISTANCE",
  "mu": 1200,
  "layer": 1,
  "createdAt": "2026-02-03T10:00:00Z"
}
```

---

## 2. WebSocket 接口（实时游戏）

连接地址：`ws://<host>:<port>/ws`

### 2.1 CONNECT

- 方向：Client → Server
- 状态：已实现
- 请求字段：
  - `authToken`（必填，JWT）
  - `playerId`（可选，实际以 JWT `sub` 为准）
  - `version`（可选）
- 响应/推送：
  - 成功：`CONNECTED`
  - 失败：`ERROR`

请求 Demo：

```json
{
  "type": "CONNECT",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "authToken": "<jwt>",
    "version": "1.0.0"
  }
}
```

响应 Demo：

```json
{
  "type": "CONNECTED",
  "timestamp": 1706500801123,
  "id": "msg-uuid",
  "data": {
    "sessionId": "session-uuid",
    "serverTime": 1706500801123
  }
}
```

### 2.2 PLAYER_TARGET_UPDATE

- 方向：Client → Server
- 状态：已实现
- 请求字段：
  - `playerId`（可选）
  - `latitude`（必填）
  - `longitude`（必填）
- 响应/推送：
  - 无直接回包，服务器在 tick 中推送 `PLAYER_STATE` 与 `NEARBY_PLAYERS`

请求 Demo：

```json
{
  "type": "PLAYER_TARGET_UPDATE",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "latitude": 39.9123,
    "longitude": 116.3812
  }
}
```

### 2.3 PLAYER_LOCAL_UPDATE

- 方向：Client → Server
- 状态：已废弃（Deprecated）
- 行为：服务器返回 `ERROR`

请求 Demo：

```json
{
  "type": "PLAYER_LOCAL_UPDATE",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "latitude": 39.9123,
    "longitude": 116.3812
  }
}
```

响应 Demo：

```json
{
  "type": "ERROR",
  "timestamp": 1706500801123,
  "id": "msg-uuid",
  "data": {
    "code": 1001,
    "message": "PLAYER_LOCAL_UPDATE deprecated"
  }
}
```

### 2.4 PLAYER_VIEW_UPDATE

- 方向：Client → Server
- 状态：已实现
- 请求字段：
  - `playerId`（可选）
  - `bounds`（可选，优先）
  - `tiles`（可选，预留）
- 响应/推送：
  - `MAP_UPDATE`（full=true）

请求 Demo：

```json
{
  "type": "PLAYER_VIEW_UPDATE",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "bounds": {
      "minLat": 39.90,
      "maxLat": 39.92,
      "minLon": 116.37,
      "maxLon": 116.39
    }
  }
}
```

推送 Demo：

```json
{
  "type": "MAP_UPDATE",
  "timestamp": 1706500801123,
  "id": "",
  "data": {
    "full": true,
    "batchIndex": 1,
    "batchTotal": 1,
    "portals": [
      {
        "id": "portal-uuid",
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

### 2.5 PLAYER_STATE

- 方向：Server → Client
- 状态：已实现
- 字段：
  - `playerId`
  - `latitude`
  - `longitude`
  - `speedMps`
  - `headingDeg`

推送 Demo：

```json
{
  "type": "PLAYER_STATE",
  "timestamp": 1706500801123,
  "id": "",
  "data": {
    "playerId": "player-uuid",
    "latitude": 39.9123,
    "longitude": 116.3812,
    "speedMps": 33.3,
    "headingDeg": 90.0
  }
}
```

### 2.6 NEARBY_PLAYERS

- 方向：Server → Client
- 状态：已实现
- 字段：
  - `players[]`: `id`, `latitude`, `longitude`
- 距离阈值：与视野半径一致（默认 400m）

推送 Demo：

```json
{
  "type": "NEARBY_PLAYERS",
  "timestamp": 1706500801123,
  "id": "",
  "data": {
    "players": [
      {"id": "player-1", "latitude": 39.91, "longitude": 116.38}
    ]
  }
}
```

### 2.7 PORTAL_UPDATE

- 方向：Server → Client
- 状态：已实现
- 用途：部署/充能/Mod/战斗后的增量更新

Deploy Resonator Demo：

```json
{
  "type": "PORTAL_UPDATE",
  "timestamp": 1706500801123,
  "id": "msg-uuid",
  "data": {
    "portalId": "portal-uuid",
    "slot": 1,
    "level": 3,
    "energy": 4500,
    "faction": "RESISTANCE",
    "portalLevel": 3,
    "version": 2
  }
}
```

Deploy Mod Demo：

```json
{
  "type": "PORTAL_UPDATE",
  "timestamp": 1706500801123,
  "id": "msg-uuid",
  "data": {
    "portalId": "portal-uuid",
    "modSlot": 1,
    "modType": "LINK_AMP",
    "rarity": "RARE",
    "version": 2
  }
}
```

Charge Portal Demo：

```json
{
  "type": "PORTAL_UPDATE",
  "timestamp": 1706500801123,
  "id": "msg-uuid",
  "data": {
    "portalId": "portal-uuid",
    "energy": 5200
  }
}
```

### 2.8 MAP_UPDATE / MAP_TICK

- 方向：Server → Client
- 状态：已实现
- 字段：
  - `full`（MAP_UPDATE=true, MAP_TICK=false）
  - `batchIndex`, `batchTotal`
  - `portals[]`: `id`, `latitude`, `longitude`, `faction`, `level`, `energy`

MAP_TICK Demo：

```json
{
  "type": "MAP_TICK",
  "timestamp": 1706500801123,
  "id": "",
  "data": {
    "full": false,
    "batchIndex": 1,
    "batchTotal": 1,
    "portals": []
  }
}
```

### 2.9 PLAYER_DEPLOY_RESONATOR

- 方向：Client → Server
- 状态：已实现（校验库存/距离为规划）
- 请求字段：
  - `portalId`（必填）
  - `slot`（1-8）
  - `level`（1-8）
  - `expectedVersion`（乐观锁）
- 响应/推送：
  - 成功：`PORTAL_UPDATE`
  - 冲突：`ERROR`（ErrCodeConflict）

请求 Demo：

```json
{
  "type": "PLAYER_DEPLOY_RESONATOR",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "portalId": "portal-uuid",
    "slot": 1,
    "level": 3,
    "expectedVersion": 1
  }
}
```

### 2.10 PLAYER_DEPLOY_MOD

- 方向：Client → Server
- 状态：已实现（校验库存/距离为规划）
- 请求字段：
  - `portalId`
  - `slot`（1-4）
  - `modType`
  - `rarity`
  - `expectedVersion`
- 响应/推送：`PORTAL_UPDATE`

请求 Demo：

```json
{
  "type": "PLAYER_DEPLOY_MOD",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "portalId": "portal-uuid",
    "slot": 1,
    "modType": "LINK_AMP",
    "rarity": "RARE",
    "expectedVersion": 0
  }
}
```

### 2.11 PLAYER_CHARGE_PORTAL

- 方向：Client → Server
- 状态：已实现（XM 消耗规则为规划）
- 请求字段：
  - `portalId`
  - `amount`
- 响应/推送：`PORTAL_UPDATE`

请求 Demo：

```json
{
  "type": "PLAYER_CHARGE_PORTAL",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "portalId": "portal-uuid",
    "amount": 500
  }
}
```

### 2.12 PLAYER_CREATE_LINK

- 方向：Client → Server
- 状态：已实现
- 请求字段：
  - `fromPortalId`
  - `toPortalId`
- 响应/推送：
  - 成功：`LINK_UPDATE`
  - 失败：`ERROR`

请求 Demo：

```json
{
  "type": "PLAYER_CREATE_LINK",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "fromPortalId": "portal-a",
    "toPortalId": "portal-b"
  }
}
```

### 2.13 LINK_UPDATE

- 方向：Server → Client
- 状态：已实现
- 字段：
  - `linkId`
  - `fromPortalId`, `toPortalId`
  - `fromLat`, `fromLon`, `toLat`, `toLon`

推送 Demo：

```json
{
  "type": "LINK_UPDATE",
  "timestamp": 1706500801123,
  "id": "",
  "data": {
    "linkId": "link-uuid",
    "fromPortalId": "portal-a",
    "toPortalId": "portal-b",
    "fromLat": 39.91,
    "fromLon": 116.38,
    "toLat": 39.92,
    "toLon": 116.39
  }
}
```

### 2.14 FIELD_CREATED

- 方向：Server → Client
- 状态：已实现
- 字段：
  - `fieldId`
  - `portalIds[]`
  - `mu`
  - `layer`
  - `faction`

推送 Demo：

```json
{
  "type": "FIELD_CREATED",
  "timestamp": 1706500801123,
  "id": "",
  "data": {
    "fieldId": "field-uuid",
    "portalIds": ["portal-a", "portal-b", "portal-c"],
    "mu": 1200,
    "layer": 1,
    "faction": "RESISTANCE"
  }
}
```

### 2.15 PLAYER_ATTACK

- 方向：Client → Server
- 状态：已实现
- 请求字段：
  - `portalId`
  - `weaponType`（XMP / US）
  - `weaponLevel`（1-8）
- 响应/推送：
  - `ATTACK_RESULT`
  - `MAP_UPDATE`（多 Portal 受影响）或 `PORTAL_UPDATE`

请求 Demo：

```json
{
  "type": "PLAYER_ATTACK",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "portalId": "portal-uuid",
    "weaponType": "XMP",
    "weaponLevel": 4
  }
}
```

### 2.16 ATTACK_RESULT

- 方向：Server → Client
- 状态：已实现
- 字段：
  - `portalId`
  - `weaponType`, `weaponLevel`
  - `damage`
  - `resonatorsDestroyed`
  - `portalEnergy`
  - `portalNeutral`
  - `itemsDropped[]`（可选）

推送 Demo：

```json
{
  "type": "ATTACK_RESULT",
  "timestamp": 1706500801123,
  "id": "msg-uuid",
  "data": {
    "portalId": "portal-uuid",
    "weaponType": "XMP",
    "weaponLevel": 4,
    "damage": 1200,
    "resonatorsDestroyed": 2,
    "portalEnergy": 3000,
    "portalNeutral": false,
    "itemsDropped": ["KEY:portal-uuid"]
  }
}
```

### 2.17 PLAYER_TOGGLE_AUTO_HACK

- 方向：Client → Server
- 状态：已实现
- 请求字段：
  - `playerId`（可选）
  - `enabled`（true/false）
- 响应/推送：
  - 无直接回包

请求 Demo：

```json
{
  "type": "PLAYER_TOGGLE_AUTO_HACK",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {
    "enabled": true
  }
}
```

### 2.18 AUTO_HACK_RESULT

- 方向：Server → Client
- 状态：已实现
- 字段：
  - `portalId`
  - `success`
  - `cooldownSeconds`

推送 Demo：

```json
{
  "type": "AUTO_HACK_RESULT",
  "timestamp": 1706500801123,
  "id": "",
  "data": {
    "portalId": "portal-uuid",
    "success": true,
    "cooldownSeconds": 300
  }
}
```

### 2.19 PING / PONG

- 方向：双向
- 状态：已实现

PING Demo：

```json
{
  "type": "PING",
  "timestamp": 1706500800123,
  "id": "msg-uuid",
  "data": {}
}
```

PONG Demo：

```json
{
  "type": "PONG",
  "timestamp": 1706500801123,
  "id": "msg-uuid",
  "data": {}
}
```

---

## 3. HTTP 接口（拉取型 / 低频）

Base Path：`/api/v1`

### 3.1 Auth

#### POST /api/v1/auth/register

- Auth：无需
- Request (JSON)：`username`、`password`、`faction`(可选)
- Response data：`access_token`、`refresh_token`、`player_id`

请求 Demo：

```json
{
  "username": "agent",
  "password": "pass",
  "faction": "RESISTANCE"
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "refresh-uuid",
    "player_id": "player-uuid"
  }
}
```

失败示例（用户名已存在）：

```json
{
  "errno": 1001,
  "errmsg": "username exists",
  "data": null
}
```

#### POST /api/v1/auth/login

- Auth：无需
- Request：`username`、`password`
- Response data：`access_token`、`refresh_token`、`player_id`

请求 Demo：

```json
{
  "username": "agent",
  "password": "pass"
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "access_token": "<jwt>",
    "refresh_token": "refresh-uuid",
    "player_id": "player-uuid"
  }
}
```

失败示例（账号或密码错误）：

```json
{
  "errno": 1004,
  "errmsg": "invalid credentials",
  "data": null
}
```

#### POST /api/v1/auth/refresh

- Auth：无需
- Request：`refresh_token`
- Response data：`access_token`

#### POST /api/v1/auth/logout

- Auth：可选（有则撤销 refresh 逻辑）
- Response data：`{"status":"ok"}`

---

### 3.2 Players

#### GET /api/v1/players/me

- Auth：需要
- Response data：`Player`

#### PATCH /api/v1/players/me

- Auth：需要
- Request：`autoHack`（可选）
- Response data：更新后的 `Player`

#### GET /api/v1/players/:id

- Auth：需要
- Response data：`Player`

#### GET /api/v1/players/:id/stats

- Auth：需要
- Response data：`portalCaptures`、`linksCreated`、`fieldsCreated`、`muTotal`

#### GET /api/v1/players/:id/visibility

- Auth：需要
- Response data：`online`、`lastActive`、`position`

---

### 3.3 Map / Portals

#### GET /api/v1/map/entities

- Auth：需要
- Query：`minLat`、`maxLat`、`minLon`、`maxLon`
- Response data：`portals[]`、`links[]`、`fields[]`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "portals": [{"id": "portal-uuid", "position": {"latitude": 39.91, "longitude": 116.38}, "faction": "RESISTANCE", "level": 3, "energy": 4500, "resonators": {}, "mods": {}, "updatedAt": "2026-02-03T10:00:00Z"}],
    "links": [{"id": "link-uuid", "fromPortalId": "portal-a", "toPortalId": "portal-b", "fromPosition": {"latitude": 39.91, "longitude": 116.38}, "toPosition": {"latitude": 39.92, "longitude": 116.39}, "createdAt": "2026-02-03T10:00:00Z"}],
    "fields": [{"id": "field-uuid", "portalIds": ["portal-a", "portal-b", "portal-c"], "faction": "RESISTANCE", "mu": 1200, "layer": 1, "createdAt": "2026-02-03T10:00:00Z"}]
  }
}
```

#### GET /api/v1/portals/:id

- Auth：需要
- Response data：`Portal`

#### GET /api/v1/portals/:id?format=legacy

- Auth：需要
- Response data：`result`（legacy 数组结构）

Legacy 响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "result": [
      "p",
      "N",
      39916543,
      116392546,
      1,
      0,
      0,
      "https://lh3.googleusercontent.com/...",
      "PortalName",
      [],
      false,
      false,
      null,
      1768102573047,
      [null, null, null, null],
      [],
      "",
      ["", "", []]
    ]
  }
}
```

---

### 3.4 Inventory

#### GET /api/v1/inventory

- Auth：需要
- Response data：`{ itemType: amount }`

#### POST /api/v1/inventory/use

- Auth：需要
- Request：`itemType`、`amount`
- Response data：更新后的库存

#### POST /api/v1/inventory/recycle

- Auth：需要
- Request：`itemType`、`amount`
- Response data：更新后的库存

---

### 3.5 Leaderboard

#### GET /api/v1/leaderboard

- Auth：无需
- Response data：`Player[]`

---

### 3.6 Logs

#### GET /api/v1/logs/global

- Auth：无需
- Response data：`{ logs: [] }`

---

### 3.7 System

#### GET /api/v1/health

- Auth：无需
- Response data：`{ status: "ok" }`

#### GET /health

- Auth：无需
- Response data：`{ status: "ok" }`

#### GET /api/v1/config

- Auth：无需
- Response data：`{ viewRadiusMeters, mapTickMs }`

---

### 3.8 Admin（管理接口）

> Auth：需要 `X-Admin-Token` 或 `Authorization: Bearer <adminToken>`

#### GET /api/v1/admin/status

- Request：无
- Response data：`players`、`portals`、`links`、`fields`、`logs`、`banned`、`time`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "players": 12,
    "portals": 120,
    "links": 30,
    "fields": 5,
    "logs": 80,
    "banned": 1,
    "time": 1706500801123
  }
}
```

#### POST /api/v1/admin/announce

- Request：`message`
- Response data：`{ status: "ok" }`

请求 Demo：

```json
{
  "message": "server maintenance in 10 minutes"
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### POST /api/v1/admin/kick

- Request：`playerId`
- Response data：`{ kicked: bool }`

请求 Demo：

```json
{
  "playerId": "player-uuid"
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "kicked": true
  }
}
```

#### POST /api/v1/admin/ban

- Request：`playerId`、`reason`(可选)
- Response data：`{ status: "ok" }`

请求 Demo：

```json
{
  "playerId": "player-uuid",
  "reason": "cheating"
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### POST /api/v1/admin/unban

- Request：`playerId`
- Response data：`{ status: "ok" }`

请求 Demo：

```json
{
  "playerId": "player-uuid"
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### POST /api/v1/admin/players/:id/faction

- Request：`faction`
- Response data：`Player`

请求 Demo：

```json
{
  "faction": "ENLIGHTENED"
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "id": "player-uuid",
    "username": "agent",
    "faction": "ENLIGHTENED",
    "level": 3,
    "ap": 1200,
    "xm": 2500,
    "maxXm": 3000,
    "inventory": {},
    "position": {"latitude": 39.91, "longitude": 116.38},
    "updatedAt": "2026-02-03T10:00:00Z"
  }
}
```

#### POST /api/v1/admin/players/:id/level

- Request：`level`
- Response data：`Player`

请求 Demo：

```json
{
  "level": 5
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "id": "player-uuid",
    "username": "agent",
    "faction": "RESISTANCE",
    "level": 5,
    "ap": 1200,
    "xm": 2500,
    "maxXm": 5000,
    "inventory": {},
    "position": {"latitude": 39.91, "longitude": 116.38},
    "updatedAt": "2026-02-03T10:00:00Z"
  }
}
```

#### POST /api/v1/admin/players/:id/grant-item

- Request：`itemId`、`amount`
- Response data：`inventory`

请求 Demo：

```json
{
  "itemId": "XMP_L3",
  "amount": 5
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "XMP_L3": 5
  }
}
```

#### POST /api/v1/admin/portals

- Request：`id`、`lat`、`lon`、`faction`(可选)
- Response data：`Portal`

请求 Demo：

```json
{
  "id": "portal-uuid",
  "lat": 39.91,
  "lon": 116.38,
  "faction": "NEUTRAL"
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "id": "portal-uuid",
    "position": {"latitude": 39.91, "longitude": 116.38},
    "faction": "NEUTRAL",
    "level": 0,
    "energy": 0,
    "resonators": {},
    "mods": {},
    "updatedAt": "2026-02-03T10:00:00Z"
  }
}
```

#### PATCH /api/v1/admin/portals/:id

- Request：`lat`、`lon`、`faction`、`level`、`energy`（可选）
- Response data：`Portal`

请求 Demo：

```json
{
  "faction": "RESISTANCE",
  "level": 3
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "id": "portal-uuid",
    "position": {"latitude": 39.91, "longitude": 116.38},
    "faction": "RESISTANCE",
    "level": 3,
    "energy": 4500,
    "resonators": {},
    "mods": {},
    "updatedAt": "2026-02-03T10:00:00Z"
  }
}
```

#### DELETE /api/v1/admin/portals/:id

- Request：无
- Response data：`{ status: "ok" }`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### DELETE /api/v1/admin/links/:id

- Request：无
- Response data：`{ status: "ok" }`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### DELETE /api/v1/admin/fields/:id

- Request：无
- Response data：`{ status: "ok" }`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### GET /api/v1/admin/logs?limit=100

- Request：`limit`(可选)
- Response data：`{ logs: LogEntry[] }`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "logs": [
      {
        "id": "log-uuid",
        "type": "BAN",
        "playerId": "player-uuid",
        "portalId": "",
        "mu": 0,
        "message": "player banned",
        "timestamp": "2026-02-03T10:00:00Z"
      }
    ]
  }
}
```

#### POST /api/v1/admin/gc-logs

- Request：`keep`
- Response data：`{ status: "ok" }`

请求 Demo：

```json
{
  "keep": 100
}
```

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### POST /api/v1/admin/recalc-stats

- Request：无
- Response data：`{ status: "ok" }`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### POST /api/v1/admin/reload-config

- Request：无
- Response data：`{ status: "ok" }`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

#### POST /api/v1/admin/rotate-logs

- Request：无
- Response data：`{ status: "ok" }`

响应 Demo：

```json
{
  "errno": 0,
  "errmsg": "success",
  "data": {
    "status": "ok"
  }
}
```

管理鉴权失败示例：

```json
{
  "errno": 1004,
  "errmsg": "invalid admin token",
  "data": null
}
```

---

## 4. 规则与备注

- 地图边界：默认北京五环（`minLat=39.7,maxLat=40.1,minLon=115.7,maxLon=117.0`），超出返回错误。
- 视野最大范围：400m（超出以玩家位置为中心截断）。
- 交互距离：Deploy/Mod/Charge/Link 需在 40m 内；Attack 需在武器半径内。
- Link 可见性：线段与视野相交即推送，端点坐标必须完整。
- Resonator 固定圆环半径：10m（slot 角度 0/45/.../315）。
- AoE 伤害：`damage = maxDamage * (1 - distance / radius)`。
- JWT：HS256，`sub`=playerId，`exp`=24h（MVP）。

**物品 ID 约定**：
- Resonator：`RESO_L1`~`RESO_L8`
- XMP：`XMP_L1`~`XMP_L8`
- Ultra Strike：`US_L1`~`US_L8`
- Power Cube：`CUBE_L1`~`CUBE_L8`（兼容旧 `CUBE` 视为 L1）
- Mod：`MOD:<modType>:<rarity>`（如 `MOD:LINK_AMP:RARE`）
- Key：`KEY:<portalId>`
- Key 上限：每 Portal 最多 2 把；超出不再掉落。
- 容量规则：Key 槽优先占用，Key 满后占 General 槽（容量按 `WIKI/player.md`）。

**XM 消耗（配置化）**：
- `deploy_resonator_cost = deploy_resonator_base + deploy_resonator_per_level * (level-1)`
- `attack_xmp_cost = attack_xmp_base + attack_xmp_per_level * (level-1)`
- `attack_us_cost = attack_us_base + attack_us_per_level * (level-1)`
- `deploy_mod_cost = deploy_mod_flat`
- `link_cost = link_flat`
- `charge_cost = amount * charge_per_xm`

**MVP 开发补充规则（2026-02-06）**：
- Hack 统一规则：手动 `HACK_PORTAL/HACK_RESULT` 与自动 `AUTO_HACK_RESULT` 使用同一套距离、冷却、掉落、容量校验与 AP 奖励逻辑。
- 掉落/发放容量校验：所有新增物品路径（Hack、攻击掉落、管理发放）必须统一经过容量与 Key 上限校验（`CanAddItem/CanAddKey` 语义）。
- Portal 中立清理：当 Portal 能量归零且变为 `NEUTRAL` 时，必须清理相关 `Links/Fields`，并通过 `MAP_UPDATE`/`LINK_UPDATE`/`FIELD_CREATED` 的反向更新及时广播可见玩家。
- 等级与属性同步：玩家 AP 变化后需立即重算等级（`AP -> Level`），并同步 `MaxXM` 与背包容量上限；所有等级限制操作按最新等级校验。
- Link 距离计算：Link 最大距离按 8 个 Resonator 组合与 Link Amp/SBUL 倍率计算，不仅按 Portal 显示等级估算（对齐 `WIKI/portal.md`）。
- 文档一致性：接口或规则实现状态变化时，必须同步更新本文件“状态说明”与对应条目状态（已实现/规划中/已废弃）。

---

## 5. 状态说明

- 已实现：与 `backend/pkg/ws` 和 `backend/pkg/api` 现有行为一致
- 规划中：字段/校验未落地，已在条目中标注
- 已废弃：如 `PLAYER_LOCAL_UPDATE`
