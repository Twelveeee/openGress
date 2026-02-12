package database

import "time"

type PlayerModel struct {
	ID                  string    `gorm:"primaryKey;size:64"`
	Username            string    `gorm:"size:128;not null;default:''"`
	Faction             string    `gorm:"size:32;not null;default:'NEUTRAL'"`
	Level               int       `gorm:"not null;default:1"`
	AP                  int       `gorm:"not null;default:0"`
	XM                  int       `gorm:"not null;default:0"`
	MaxXM               int       `gorm:"column:max_xm;not null;default:0"`
	PortalCaptures      int       `gorm:"column:portal_captures;not null;default:0"`
	LinksCreated        int       `gorm:"column:links_created;not null;default:0"`
	FieldsCreated       int       `gorm:"column:fields_created;not null;default:0"`
	MUTotal             int       `gorm:"column:mu_total;not null;default:0"`
	PosLat              float64   `gorm:"column:pos_lat;not null;default:0"`
	PosLon              float64   `gorm:"column:pos_lon;not null;default:0"`
	AutoHack            bool      `gorm:"column:auto_hack;not null;default:false"`
	LegacyInventoryJSON []byte    `gorm:"column:inventory_json;type:jsonb"`
	CooldownsJSON       []byte    `gorm:"column:cooldowns_json;type:jsonb"`
	ViewJSON            []byte    `gorm:"column:view_json;type:jsonb"`
	ViewTilesJSON       []byte    `gorm:"column:view_tiles_json;type:jsonb"`
	TargetJSON          []byte    `gorm:"column:target_json;type:jsonb"`
	UpdatedAt           time.Time `gorm:"not null"`
}

func (PlayerModel) TableName() string {
	return "players"
}

type PlayerInventoryModel struct {
	PlayerID string `gorm:"column:player_id;size:64;primaryKey"`
	ItemID   string `gorm:"column:item_id;size:64;primaryKey"`
	Amount   int    `gorm:"column:amount;not null;default:0"`
}

func (PlayerInventoryModel) TableName() string {
	return "player_inventories"
}

type PortalModel struct {
	ID             string    `gorm:"primaryKey;size:64"`
	Title          string    `gorm:"column:title;size:255;not null;default:''"`
	CoverURL       string    `gorm:"column:cover_url;type:text;not null;default:''"`
	Latitude       float64   `gorm:"not null"`
	Longitude      float64   `gorm:"not null"`
	Faction        string    `gorm:"size:32;not null;default:'NEUTRAL'"`
	Level          int       `gorm:"not null;default:0"`
	Energy         int       `gorm:"not null;default:0"`
	ResonatorsJSON []byte    `gorm:"column:resonators_json;type:jsonb"`
	ModsJSON       []byte    `gorm:"column:mods_json;type:jsonb"`
	UpdatedAt      time.Time `gorm:"not null"`
}

func (PortalModel) TableName() string {
	return "portals"
}

type LinkModel struct {
	ID           string    `gorm:"primaryKey;size:64"`
	FromPortalID string    `gorm:"column:from_portal_id;size:64;not null;index"`
	ToPortalID   string    `gorm:"column:to_portal_id;size:64;not null;index"`
	FromLat      float64   `gorm:"column:from_lat;not null"`
	FromLon      float64   `gorm:"column:from_lon;not null"`
	ToLat        float64   `gorm:"column:to_lat;not null"`
	ToLon        float64   `gorm:"column:to_lon;not null"`
	CreatedAt    time.Time `gorm:"not null"`
}

func (LinkModel) TableName() string {
	return "links"
}

type FieldModel struct {
	ID        string    `gorm:"primaryKey;size:64"`
	Portal1ID string    `gorm:"column:portal_1_id;size:64;not null;index"`
	Portal2ID string    `gorm:"column:portal_2_id;size:64;not null;index"`
	Portal3ID string    `gorm:"column:portal_3_id;size:64;not null;index"`
	Faction   string    `gorm:"size:32;not null;default:'NEUTRAL'"`
	MU        int       `gorm:"not null;default:0"`
	Layer     int       `gorm:"not null;default:0"`
	CreatedAt time.Time `gorm:"not null"`
}

func (FieldModel) TableName() string {
	return "fields"
}

type LogModel struct {
	ID        string    `gorm:"primaryKey;size:64"`
	Type      string    `gorm:"size:32;not null"`
	PlayerID  string    `gorm:"column:player_id;size:64;index"`
	PortalID  string    `gorm:"column:portal_id;size:64;index"`
	Faction   string    `gorm:"size:32"`
	MU        int       `gorm:"not null;default:0"`
	Message   string    `gorm:"type:text;not null;default:''"`
	Timestamp time.Time `gorm:"not null;index"`
}

func (LogModel) TableName() string {
	return "logs"
}

type AdminBanModel struct {
	PlayerID  string    `gorm:"column:player_id;primaryKey;size:64"`
	Reason    string    `gorm:"type:text;not null;default:''"`
	UpdatedAt time.Time `gorm:"not null"`
}

func (AdminBanModel) TableName() string {
	return "admin_bans"
}

type AuthUserModel struct {
	UsernameNorm string    `gorm:"column:username_norm;primaryKey;size:128"`
	Username     string    `gorm:"column:username;size:128;not null"`
	PasswordHash string    `gorm:"column:password_hash;type:text;not null"`
	PlayerID     string    `gorm:"column:player_id;size:64;not null;uniqueIndex"`
	CreatedAt    time.Time `gorm:"column:created_at;not null"`
	UpdatedAt    time.Time `gorm:"column:updated_at;not null"`
}

func (AuthUserModel) TableName() string {
	return "auth_users"
}
