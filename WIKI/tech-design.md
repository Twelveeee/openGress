# OpenIngress RTS 技术设计文档

## 技术栈推荐

### 前端

**推荐框架**: React + Leaflet

**理由**:
- React 优秀的组件化架构，状态管理清晰
- Leaflet 提供 OpenStreetMap 无缝集成
- 生态丰富，文档完善
- WebSocket 集成方便

### 后端

**推荐语言**: Go (Golang)

**理由**:
- 优秀的并发支持 (goroutines)
- 低内存占用
- 原生 HTTP/2 支持
- 内置 WebSocket 库，性能优异
- 快速编译，单文件部署

### 数据库

**推荐方案**: PostgreSQL + PostGIS

**理由**:
- PostGIS 提供地理空间查询能力
- GIST 索引支持高效位置查询
- 事务支持确保数据一致性
- ACID 合规，适合游戏状态

### 地图边界

**当前限制**: 北京 5 环以内

**北京 5 环坐标范围**:
- 1 环: 天安门周边
- 2 环: 二环路
- 3 环: 三环路
- 4 环: 四环路
- 5 环: 五环路/六环路
- 约: 39.7°N ~ 40.1°N, 115.7°E ~ 117.0°E

**注意**: 坐标范围可根据实际情况调整

**边界验证逻辑**:
```go
type MapBounds struct {
    MinLat float64
    MaxLat float64
    MinLon float64
    MaxLon float64
}

var GameMapBounds = MapBounds{
    MinLat: 39.7,
    MaxLat: 40.1,
    MinLon: 115.7,
    MaxLon: 117.0,
}

func ValidateMapBounds(lat, lon float64) error {
    if lat < GameMapBounds.MinLat || lat > GameMapBounds.MaxLat {
        return fmt.Errorf("latitude out of bounds")
    }
    if lon < GameMapBounds.MinLon || lon > GameMapBounds.MaxLon {
        return fmt.Errorf("longitude out of bounds")
    }
    return nil
}

// 玩家移动验证
func ValidatePlayerMovement(newPos Position) error {
    if err := ValidateMapBounds(newPos.Latitude, newPos.Longitude); err != nil {
        return fmt.Errorf("move position out of map bounds: %w", err)
    }
    return nil
}

// Portal 创建验证
func ValidatePortalPosition(lat, lon float64) error {
    return ValidateMapBounds(lat, lon)
}
```

---

## WebSocket 消息协议

### 消息结构

```json
{
  "type": "MESSAGE_TYPE",
  "timestamp": 1706500800,
  "id": "msg_uuid",
  "data": { }
}
```

### 消息类型

| 类型 | 方向 | 描述 |
|------|--------|------|
| CONNECT | Client→Server | 玩家连接 |
| CONNECTED | Server→Client | 连接成功 |
| PLAYER_UPDATE | Client→Server | 更新玩家位置 |
| NEARBY_PLAYERS | Server→Client | 附近玩家列表 |
| PORTAL_UPDATE | Server→Client | 门户状态更新 |
| MAP_UPDATE | Server→Client | 地图批量更新 |
| HACK_PORTAL | Client→Server | 入侵门户 |
| HACK_RESULT | Server→Client | 入侵结果 |
| DEPLOY_RESONATOR | Client→Server | 部署谐振器 |
| CREATE_LINK | Client→Server | 创建连接 |
| ATTACK_PORTAL | Client→Server | 攻击门户 |
| FIELD_CREATED | Server→Client | 区域创建 |
| LINK_UPDATE | Server→Client | 连接状态更新 |
| ERROR | Server→Client | 错误信息 |
| PING/PONG | 双向 | 心跳保活 |

### 连接消息

```json
// Client → Server
{
  "type": "CONNECT",
  "data": {
    "playerId": "uuid",
    "authToken": "jwt-token",
    "version": "1.0.0"
  }
}

// Server → Client
{
  "type": "CONNECTED",
  "data": {
    "sessionId": "session-uuid",
    "serverTime": 1706500800000
  }
}
```

### 玩家位置更新

```json
// Client → Server
{
  "type": "PLAYER_UPDATE",
  "data": {
    "latitude": 37.7749,
    "longitude": -122.4194
  }
}

// Server → Client (附近玩家)
{
  "type": "NEARBY_PLAYERS",
  "data": {
    "players": [
      {
        "id": "player-uuid",
        "username": "AgentName",
        "faction": "RESISTANCE",
        "level": 8
      }
    ]
  }
}
```

### 门户更新

```json
// Server → Client
{
  "type": "PORTAL_UPDATE",
  "data": {
    "portal": {
      "id": "portal-uuid",
      "name": "Portal Name",
      "level": 8,
      "faction": "RESISTANCE",
      "resonators": [
        { "position": 1, "level": 8, "energy": 6000 }
      ]
    },
    "changeType": "DEPLOY"
  }
}
```

### 入侵消息

```json
// Client → Server
{
  "type": "HACK_PORTAL",
  "data": {
    "portalId": "portal-uuid",
    "hackType": "NORMAL"
  }
}

// Server → Client
{
  "type": "HACK_RESULT",
  "data": {
    "success": true,
    "itemsGained": [
      { "type": "XMP", "level": 8, "rarity": "COMMON" }
    ],
    "apGained": 500,
    "cooldownSeconds": 300
  }
}
```

### 攻击消息

```json
// Client → Server
{
  "type": "ATTACK_PORTAL",
  "data": {
    "portalId": "portal-uuid",
    "weaponType": "XMP",
    "weaponLevel": 8
  }
}

// Server → Client (攻击结果)
{
  "type": "ATTACK_RESULT",
  "data": {
    "damageToResonators": 4500,
    "resonatorsDestroyed": [
      { "position": 1, "level": 8 }
    ],
    "portalCaptured": false,
    "itemsDropped": [
      { "type": "PORTAL_KEY", "portalId": "target-uuid" }
    ]
  }
}
```

---

## 数据库 Schema

### 玩家表 (players)

```sql
CREATE TABLE players (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    faction VARCHAR(10) NOT NULL CHECK (faction IN ('ENLIGHTENED', 'RESISTANCE')),
    level INTEGER NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 16),
    ap INTEGER NOT NULL DEFAULT 0,
    current_xm INTEGER NOT NULL DEFAULT 3000,
    max_xm INTEGER NOT NULL DEFAULT 3000,
    latitude DECIMAL(10, 7),
    longitude DECIMAL(10, 7),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_active TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### 门户表 (portals)

```sql
CREATE TABLE portals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    latitude DECIMAL(10, 7) NOT NULL,
    longitude DECIMAL(10, 7) NOT NULL,
    location GEOGRAPHY(POINT, 4326) NOT NULL,
    level INTEGER DEFAULT 0 CHECK (level BETWEEN 0 AND 8),
    faction VARCHAR(10) CHECK (faction IN ('ENLIGHTENED', 'RESISTANCE', 'NEUTRAL')),
    owner_id UUID REFERENCES players(id),
    energy INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_modified TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_portals_location ON portals USING GIST (location);
CREATE INDEX idx_portals_level ON portals (level);
CREATE INDEX idx_portals_faction ON portals (faction);
```

### 谐振器表 (resonators)

```sql
CREATE TABLE resonators (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portal_id UUID NOT NULL REFERENCES portals(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    position INTEGER NOT NULL CHECK (position BETWEEN 1 AND 8),
    level INTEGER NOT NULL CHECK (level BETWEEN 1 AND 8),
    energy INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (portal_id, position)
);
```

### 连接表 (links)

```sql
CREATE TABLE links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portal_from_id UUID NOT NULL REFERENCES portals(id) ON DELETE CASCADE,
    portal_to_id UUID NOT NULL REFERENCES portals(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    level INTEGER NOT NULL CHECK (level BETWEEN 1 AND 8),
    length_km DECIMAL(10, 2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CHECK (portal_from_id != portal_to_id),
    UNIQUE (portal_from_id, portal_to_id)
);

CREATE INDEX idx_links_from ON links (portal_from_id);
CREATE INDEX idx_links_to ON links (portal_to_id);
```

### 区域表 (fields)

```sql
CREATE TABLE fields (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portal_1_id UUID NOT NULL REFERENCES portals(id) ON DELETE CASCADE,
    portal_2_id UUID NOT NULL REFERENCES portals(id) ON DELETE CASCADE,
    portal_3_id UUID NOT NULL REFERENCES portals(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    faction VARCHAR(10) NOT NULL,
    mu INTEGER NOT NULL,
    layer INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE (portal_1_id, portal_2_id, portal_3_id)
);
```

### 物品表 (inventory)

```sql
CREATE TABLE inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    player_id UUID NOT NULL REFERENCES players(id) ON DELETE CASCADE,
    item_type VARCHAR(50) NOT NULL,
    level INTEGER CHECK (level BETWEEN 1 AND 8),
    rarity VARCHAR(20) CHECK (rarity IN ('COMMON', 'RARE', 'VERY_RARE')),
    quantity INTEGER NOT NULL DEFAULT 1,
    portal_id UUID REFERENCES portals(id), -- For keys
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### 模组表 (portal_mods)

```sql
CREATE TABLE portal_mods (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    portal_id UUID NOT NULL REFERENCES portals(id) ON DELETE CASCADE,
    player_id UUID NOT NULL REFERENCES players(id),
    slot INTEGER NOT NULL CHECK (slot BETWEEN 1 AND 4),
    mod_type VARCHAR(50) NOT NULL,
    level INTEGER CHECK (level BETWEEN 1 AND 8),
    rarity VARCHAR(20),
    UNIQUE (portal_id, slot)
);
```

---

## 游戏逻辑

### 玩家移动系统

```go
const MaxSpeedKmH = 120.0

func ValidateMovement(oldPos, newPos Position, timeDelta time.Duration) error {
    distance := CalculateDistance(oldPos, newPos)
    hours := timeDelta.Hours()
    speedKmH := (distance / 1000) / hours

    if speedKmH > MaxSpeedKmH {
        return fmt.Errorf("movement speed exceeds maximum")
    }
    return nil
}

// Haversine 公式计算距离
func CalculateDistance(p1, p2 Position) float64 {
    const earthRadius = 6371000.0
    // ... 实现
    return distance
}
```

### 战斗系统

```go
// 伤害计算
func CalculateDamage(weapon Weapon, target Resonator, portal Portal) float64 {
    baseDamage := float64(weapon.Level) * 300

    // 距离衰减
    distance := CalculateDistance(weapon.Player.Position, target.Position)
    distanceFactor := math.Max(0, 1.0 - (distance / 40.0))

    // 护盾减免
    shieldMitigation := CalculateShieldMitigation(portal)

    damage := baseDamage * distanceFactor * (1.0 - shieldMitigation)
    return math.Max(0, damage)
}

// 护盾计算
func CalculateShieldMitigation(portal Portal) float64 {
    var totalMitigation float64
    for _, mod := range portal.Mods {
        if mod.Type == "SHIELD" {
            switch mod.Rarity {
            case "COMMON":
                totalMitigation += 0.10
            case "RARE":
                totalMitigation += 0.15
            case "VERY_RARE":
                totalMitigation += 0.20
            }
        }
    }
    return math.Min(totalMitigation, 0.60) // 最多 60%
}
```

### 连接验证

```go
func ValidateLinkCreation(fromPortal, toPortal Portal) (bool, string) {
    // 检查阵营
    if fromPortal.Faction != toPortal.Faction {
        return false, "different factions"
    }

    // 检查距离
    distance := CalculateDistance(fromPortal.Position, toPortal.Position)
    maxDistance := CalculateLinkDistance(fromPortal.Level)

    if distance > maxDistance {
        return false, "distance exceeded"
    }

    // 检查交叉
    if LinksWouldCross(fromPortal, toPortal) {
        return false, "links would cross"
    }

    // 检查 Out Link 数量
    if len(fromPortal.OutgoingLinks) >= 8 {
        return false, "max outgoing links"
    }

    return true, ""
}
```

### 区域创建

```go
func CreateField(p1, p2, p3 Portal) (Field, error) {
    // 验证三个门户已连接
    if !LinkExists(p1, p2) || !LinkExists(p2, p3) || !LinkExists(p3, p1) {
        return Field{}, errors.New("links must exist")
    }

    // 计算 MU
    mu := CalculateFieldMU(p1, p2, p3)

    // 确定层级
    layer := DetermineFieldLayer(p1, p2, p3)

    return Field{
        Portal1: p1,
        Portal2: p2,
        Portal3: p3,
        MU: mu,
        Layer: layer,
    }, nil
}
```

---

## 常量定义

```go
const (
    // 距离限制
    PortalHackRange    = 40.0   // meters
    PortalDeployRange   = 40.0   // meters
    PortalRechargeRange = 500.0  // meters

    // 数量限制
    MaxLinksPerPortal = 8
    MaxModsPerPortal = 4
    ResonatorsPerPortal = 8

    // 衰减
    ResonatorDecayRate = 0.15 // 15% per 24 hours

    // 玩家
    MaxPlayerSpeedKmH = 120.0
)
```

---

## 性能优化

### 前端

- 使用 `React.memo` 避免不必要的重渲染
- 地图瓦片缓存
- WebSocket 消息批量处理
- 虚拟列表用于大量玩家

### 后端

- 使用连接池管理数据库连接
- Goroutine 池处理高并发
- 内存状态减少数据库查询
- 地理空间索引 (GIST) 优化位置查询

---

*注: 本文档基于 Ingress 机制和实时游戏开发最佳实践整理。*
