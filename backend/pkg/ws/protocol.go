package ws

import pbws "github.com/Twelveeee/openGress/backend/pkg/pb/ws"

type Message = pbws.Message
type ErrorPayload = pbws.ErrorPayload
type ConnectPayload = pbws.ConnectPayload
type PlayerTargetUpdatePayload = pbws.PlayerTargetUpdatePayload
type ViewBoundsPayload = pbws.ViewBoundsPayload
type PlayerViewUpdatePayload = pbws.PlayerViewUpdatePayload
type DeployResonatorPayload = pbws.DeployResonatorPayload
type DeployModPayload = pbws.DeployModPayload
type ChargePortalPayload = pbws.ChargePortalPayload
type CreateLinkPayload = pbws.CreateLinkPayload
type AttackPortalPayload = pbws.AttackPortalPayload
type HackPortalPayload = pbws.HackPortalPayload
type AutoHackTogglePayload = pbws.AutoHackTogglePayload
type AutoHackResultPayload = pbws.AutoHackResultPayload
type HackResultPayload = pbws.HackResultPayload
type AttackResultPayload = pbws.AttackResultPayload
type AttackCounterattack = pbws.AttackCounterattack
type AttackDestroyedMod = pbws.AttackDestroyedMod
type AttackPortalDamage = pbws.AttackPortalDamage
type PortalUpdatePayload = pbws.PortalUpdatePayload
type PortalUpdateResonator = pbws.PortalUpdateResonator
type PortalUpdateMod = pbws.PortalUpdateMod
type PlayerStatePayload = pbws.PlayerStatePayload
type PlayerResourceUpdatePayload = pbws.PlayerResourceUpdatePayload
type NearbyPlayersPayload = pbws.NearbyPlayersPayload
type NearbyPlayer = pbws.NearbyPlayer
type FieldCreatedPayload = pbws.FieldCreatedPayload

func NewMessage(msgType, id string, data interface{}) Message {
	return pbws.NewMessage(msgType, id, data)
}

const (
	MessageConnect              = pbws.MessageConnect
	MessageConnected            = pbws.MessageConnected
	MessagePlayerTargetUpdate   = pbws.MessagePlayerTargetUpdate
	MessagePlayerLocalUpdate    = pbws.MessagePlayerLocalUpdate
	MessagePlayerViewUpdate     = pbws.MessagePlayerViewUpdate
	MessageNearbyPlayers        = pbws.MessageNearbyPlayers
	MessagePlayerState          = pbws.MessagePlayerState
	MessagePlayerResourceUpdate = pbws.MessagePlayerResourceUpdate
	MessagePortalUpdate         = pbws.MessagePortalUpdate
	MessageMapUpdate            = pbws.MessageMapUpdate
	MessageMapTick              = pbws.MessageMapTick
	MessageHackPortal           = pbws.MessageHackPortal
	MessageHackResult           = pbws.MessageHackResult
	MessageAutoHackResult       = pbws.MessageAutoHackResult
	MessagePlayerDeployRes      = pbws.MessagePlayerDeployRes
	MessagePlayerDeployMod      = pbws.MessagePlayerDeployMod
	MessagePlayerChargePortal   = pbws.MessagePlayerChargePortal
	MessagePlayerCreateLink     = pbws.MessagePlayerCreateLink
	MessageLinkUpdate           = pbws.MessageLinkUpdate
	MessageFieldCreated         = pbws.MessageFieldCreated
	MessagePlayerAttack         = pbws.MessagePlayerAttack
	MessageAttackResult         = pbws.MessageAttackResult
	MessagePlayerToggleAutoHack = pbws.MessagePlayerToggleAutoHack
	MessageError                = pbws.MessageError
	MessagePing                 = pbws.MessagePing
	MessagePong                 = pbws.MessagePong
)
