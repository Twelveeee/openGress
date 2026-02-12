package state

import "time"

type Position struct {
	// 统一的地理坐标结构。
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Bounds struct {
	// 视野范围（经纬度矩形）。
	MinLat float64 `json:"minLat"`
	MaxLat float64 `json:"maxLat"`
	MinLon float64 `json:"minLon"`
	MaxLon float64 `json:"maxLon"`
}

type Player struct {
	// 玩家核心状态。
	ID             string               `json:"id"`
	Username       string               `json:"username"`
	Faction        string               `json:"faction"`
	Level          int                  `json:"level"`
	AP             int                  `json:"ap"`
	XM             int                  `json:"xm"`
	MaxXM          int                  `json:"maxXm"`
	PortalCaptures int                  `json:"portalCaptures"`
	LinksCreated   int                  `json:"linksCreated"`
	FieldsCreated  int                  `json:"fieldsCreated"`
	MUTotal        int                  `json:"muTotal"`
	Inventory      map[string]int       `json:"inventory,omitempty"`
	Position       Position             `json:"position"`
	Target         *Position            `json:"target,omitempty"`
	View           *Bounds              `json:"view,omitempty"`
	ViewTiles      []string             `json:"viewTiles,omitempty"`
	AutoHack       bool                 `json:"autoHack"`
	HackCooldowns  map[string]time.Time `json:"hackCooldowns,omitempty"`
	UpdatedAt      time.Time            `json:"updatedAt"`
}

type Portal struct {
	// Portal 状态与能量信息。
	ID         string                `json:"id"`
	Title      string                `json:"title"`
	CoverURL   string                `json:"cover_url"`
	Position   Position              `json:"position"`
	Faction    string                `json:"faction"`
	Level      int                   `json:"level"`
	Energy     int                   `json:"energy"`
	Resonators map[int]ResonatorSlot `json:"resonators,omitempty"`
	Mods       map[int]ModSlot       `json:"mods,omitempty"`
	UpdatedAt  time.Time             `json:"updatedAt"`
}

type ResonatorSlot struct {
	// Resonator 槽位状态，包含乐观锁版本号。
	Slot      int       `json:"slot"`
	Level     int       `json:"level"`
	Energy    int       `json:"energy"`
	PlayerID  string    `json:"playerId"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ModSlot struct {
	// Mod 槽位状态，包含乐观锁版本号。
	Slot      int       `json:"slot"`
	ModType   string    `json:"modType"`
	Rarity    string    `json:"rarity"`
	PlayerID  string    `json:"playerId"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Field struct {
	// Field（三角场）由三座 Portal 组成。
	ID        string    `json:"id"`
	PortalIDs [3]string `json:"portalIds"`
	Faction   string    `json:"faction"`
	MU        int       `json:"mu"`
	Layer     int       `json:"layer"`
	CreatedAt time.Time `json:"createdAt"`
}

type Link struct {
	// Link 连接信息（包含端点坐标）。
	ID           string    `json:"id"`
	FromPortalID string    `json:"fromPortalId"`
	ToPortalID   string    `json:"toPortalId"`
	FromPosition Position  `json:"fromPosition"`
	ToPosition   Position  `json:"toPosition"`
	CreatedAt    time.Time `json:"createdAt"`
}
