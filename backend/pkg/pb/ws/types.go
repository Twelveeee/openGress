package ws

import "time"

type Message struct {
	Type      string      `json:"type"`
	Timestamp int64       `json:"timestamp"`
	ID        string      `json:"id"`
	Data      interface{} `json:"data"`
}

func NewMessage(msgType, id string, data interface{}) Message {
	return Message{
		Type:      msgType,
		Timestamp: time.Now().UnixMilli(),
		ID:        id,
		Data:      data,
	}
}

const (
	MessageConnect              = "CONNECT"
	MessageConnected            = "CONNECTED"
	MessagePlayerTargetUpdate   = "PLAYER_TARGET_UPDATE"
	MessagePlayerLocalUpdate    = "PLAYER_LOCAL_UPDATE"
	MessagePlayerViewUpdate     = "PLAYER_VIEW_UPDATE"
	MessageNearbyPlayers        = "NEARBY_PLAYERS"
	MessagePlayerState          = "PLAYER_STATE"
	MessagePlayerResourceUpdate = "PLAYER_RESOURCE_UPDATE"
	MessagePortalUpdate         = "PORTAL_UPDATE"
	MessageMapUpdate            = "MAP_UPDATE"
	MessageMapTick              = "MAP_TICK"
	MessageHackPortal           = "HACK_PORTAL"
	MessageHackResult           = "HACK_RESULT"
	MessageAutoHackResult       = "AUTO_HACK_RESULT"
	MessagePlayerDeployRes      = "PLAYER_DEPLOY_RESONATOR"
	MessagePlayerDeployMod      = "PLAYER_DEPLOY_MOD"
	MessagePlayerChargePortal   = "PLAYER_CHARGE_PORTAL"
	MessagePlayerCreateLink     = "PLAYER_CREATE_LINK"
	MessageLinkUpdate           = "LINK_UPDATE"
	MessageFieldCreated         = "FIELD_CREATED"
	MessagePlayerAttack         = "PLAYER_ATTACK"
	MessageAttackResult         = "ATTACK_RESULT"
	MessagePlayerToggleAutoHack = "PLAYER_TOGGLE_AUTO_HACK"
	MessageError                = "ERROR"
	MessagePing                 = "PING"
	MessagePong                 = "PONG"
)

type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ConnectPayload struct {
	PlayerID  string `json:"playerId"`
	AuthToken string `json:"authToken"`
	Version   string `json:"version"`
}

type PlayerTargetUpdatePayload struct {
	PlayerID  string  `json:"playerId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type ViewBoundsPayload struct {
	MinLat float64 `json:"minLat"`
	MaxLat float64 `json:"maxLat"`
	MinLon float64 `json:"minLon"`
	MaxLon float64 `json:"maxLon"`
}

type PlayerViewUpdatePayload struct {
	PlayerID string             `json:"playerId"`
	Bounds   *ViewBoundsPayload `json:"bounds"`
	Tiles    []string           `json:"tiles"`
}

type DeployResonatorPayload struct {
	PlayerID        string `json:"playerId"`
	PortalID        string `json:"portalId"`
	Slot            int    `json:"slot"`
	Level           int    `json:"level"`
	ExpectedVersion int    `json:"expectedVersion"`
}

type DeployModPayload struct {
	PlayerID        string `json:"playerId"`
	PortalID        string `json:"portalId"`
	Slot            int    `json:"slot"`
	ModType         string `json:"modType"`
	Rarity          string `json:"rarity"`
	ExpectedVersion int    `json:"expectedVersion"`
}

type ChargePortalPayload struct {
	PlayerID string `json:"playerId"`
	PortalID string `json:"portalId"`
	Amount   int    `json:"amount"`
}

type CreateLinkPayload struct {
	PlayerID     string `json:"playerId"`
	FromPortalID string `json:"fromPortalId"`
	ToPortalID   string `json:"toPortalId"`
}

type AttackPortalPayload struct {
	PlayerID    string  `json:"playerId"`
	PortalID    string  `json:"portalId,omitempty"`
	WeaponType  string  `json:"weaponType"`
	WeaponLevel int     `json:"weaponLevel"`
	ChargeBonus float64 `json:"chargeBonus,omitempty"`
}

type PortalUpdateResonator struct {
	Slot     int    `json:"slot"`
	Level    int    `json:"level"`
	Energy   int    `json:"energy"`
	PlayerID string `json:"playerId,omitempty"`
	Version  int    `json:"version,omitempty"`
}

type PortalUpdateMod struct {
	Slot     int    `json:"slot"`
	ModType  string `json:"modType"`
	Rarity   string `json:"rarity"`
	PlayerID string `json:"playerId,omitempty"`
	Version  int    `json:"version,omitempty"`
}

type PortalUpdatePayload struct {
	PortalID   string                        `json:"portalId"`
	Faction    string                        `json:"faction,omitempty"`
	Energy     int                           `json:"energy,omitempty"`
	Level      int                           `json:"level,omitempty"`
	Owner      string                        `json:"owner,omitempty"`
	Resonators map[int]PortalUpdateResonator `json:"resonators,omitempty"`
	Mods       map[int]PortalUpdateMod       `json:"mods,omitempty"`
}

type HackPortalPayload struct {
	PlayerID string `json:"playerId"`
	PortalID string `json:"portalId"`
	HackType string `json:"hackType"`
}

type MapPortal struct {
	ID        string  `json:"id"`
	Title     string  `json:"title"`
	CoverURL  string  `json:"cover_url"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Faction   string  `json:"faction"`
	Level     int     `json:"level"`
	Energy    int     `json:"energy"`
}

type MapUpdatePayload struct {
	Full       bool        `json:"full"`
	BatchIndex int         `json:"batchIndex"`
	BatchTotal int         `json:"batchTotal"`
	Portals    []MapPortal `json:"portals"`
}

type LinkUpdatePayload struct {
	LinkID       string  `json:"linkId"`
	FromPortalID string  `json:"fromPortalId"`
	ToPortalID   string  `json:"toPortalId"`
	FromLat      float64 `json:"fromLat"`
	FromLon      float64 `json:"fromLon"`
	ToLat        float64 `json:"toLat"`
	ToLon        float64 `json:"toLon"`
	Removed      bool    `json:"removed,omitempty"`
}

type FieldCreatedPayload struct {
	FieldID   string   `json:"fieldId"`
	PortalIDs []string `json:"portalIds"`
	MU        int      `json:"mu"`
	Layer     int      `json:"layer"`
	Faction   string   `json:"faction"`
	Removed   bool     `json:"removed,omitempty"`
}

type PlayerStatePayload struct {
	PlayerID           string  `json:"playerId"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	RenderLatitude     float64 `json:"renderLatitude"`
	RenderLongitude    float64 `json:"renderLongitude"`
	PreRenderLatitude  float64 `json:"preRenderLatitude"`
	PreRenderLongitude float64 `json:"preRenderLongitude"`
	SpeedMps           float64 `json:"speedMps"`
	HeadingDeg         float64 `json:"headingDeg"`
}

type PlayerResourceUpdatePayload struct {
	PlayerID       string         `json:"playerId"`
	APGained       int            `json:"apGained,omitempty"`
	XMDelta        int            `json:"xmDelta,omitempty"`
	InventoryDelta map[string]int `json:"inventoryDelta,omitempty"`
}

type NearbyPlayersPayload struct {
	Players []NearbyPlayer `json:"players"`
}

type NearbyPlayer struct {
	ID                 string  `json:"id"`
	Latitude           float64 `json:"latitude"`
	Longitude          float64 `json:"longitude"`
	RenderLatitude     float64 `json:"renderLatitude"`
	RenderLongitude    float64 `json:"renderLongitude"`
	PreRenderLatitude  float64 `json:"preRenderLatitude"`
	PreRenderLongitude float64 `json:"preRenderLongitude"`
}

type AutoHackTogglePayload struct {
	PlayerID string `json:"playerId"`
	Enabled  bool   `json:"enabled"`
}

type AutoHackResultPayload struct {
	PortalID        string   `json:"portalId"`
	Success         bool     `json:"success"`
	CooldownSeconds int      `json:"cooldownSeconds"`
	ItemsGained     []string `json:"itemsGained,omitempty"`
	APGained        int      `json:"apGained,omitempty"`
}

type HackResultPayload struct {
	PortalID        string   `json:"portalId"`
	Success         bool     `json:"success"`
	ItemsGained     []string `json:"itemsGained,omitempty"`
	APGained        int      `json:"apGained,omitempty"`
	CooldownSeconds int      `json:"cooldownSeconds"`
}

type AttackResultPayload struct {
	PortalID               string                `json:"portalId"`
	WeaponType             string                `json:"weaponType"`
	WeaponLevel            int                   `json:"weaponLevel"`
	ChargeBonus            float64               `json:"chargeBonus,omitempty"`
	DamageDealt            int                   `json:"damage"`
	ResonatorsDestroyed    int                   `json:"resonatorsDestroyed"`
	PortalEnergy           int                   `json:"portalEnergy"`
	PortalNeutral          bool                  `json:"portalNeutral"`
	MitigationApplied      float64               `json:"mitigationApplied,omitempty"`
	ModsDestroyed          []AttackDestroyedMod  `json:"modsDestroyed,omitempty"`
	CounterattackTriggered bool                  `json:"counterattackTriggered,omitempty"`
	CounterattackDamage    int                   `json:"counterattackDamage,omitempty"`
	Counterattacks         []AttackCounterattack `json:"counterattacks,omitempty"`
	PortalDamages          []AttackPortalDamage  `json:"portalDamages,omitempty"`
	ItemsDropped           []string              `json:"itemsDropped,omitempty"`
}

type AttackDestroyedMod struct {
	PortalID string `json:"portalId"`
	Slot     int    `json:"slot"`
	ModType  string `json:"modType"`
	Rarity   string `json:"rarity"`
}

type AttackPortalDamage struct {
	PortalID            string `json:"portalId"`
	Damage              int    `json:"damage"`
	ResonatorsDestroyed int    `json:"resonatorsDestroyed"`
	PortalNeutral       bool   `json:"portalNeutral,omitempty"`
}

type AttackCounterattack struct {
	PortalID  string `json:"portalId"`
	Triggered bool   `json:"triggered"`
	Damage    int    `json:"damage,omitempty"`
	Critical  bool   `json:"critical,omitempty"`
}
