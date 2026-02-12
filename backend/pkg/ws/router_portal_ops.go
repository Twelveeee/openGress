package ws

import (
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/service/portal"
)

func handlePlayerDeployRes(hub *Hub, client *Client, msg Message) {
	var payload DeployResonatorPayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid deploy resonator payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" || payload.PortalID == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId or portalId"}))
		return
	}
	if hub.services == nil || hub.services.Portal == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	result, err := hub.services.Portal.DeployResonator(portal.DeployResonatorCmd{
		PlayerID:        playerID,
		PortalID:        payload.PortalID,
		Slot:            payload.Slot,
		Level:           payload.Level,
		ExpectedVersion: payload.ExpectedVersion,
	}, time.Now())
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
	slotEnergy := 0
	if result.Portal.Resonators != nil {
		if slot, ok := result.Portal.Resonators[payload.Slot]; ok {
			slotEnergy = slot.Energy
		}
	}
	client.Send(NewMessage(MessagePlayerResourceUpdate, msg.ID, PlayerResourceUpdatePayload{
		PlayerID: result.Player.ID,
		APGained: result.APGained,
		XMDelta:  -gameplay.DeployResonatorCost(hub.gameplay, payload.Level),
		InventoryDelta: map[string]int{
			gameplay.ItemIDResonator(payload.Level): -1,
		},
	}))
	hub.Broadcast(NewMessage(MessagePortalUpdate, "", map[string]interface{}{
		"portalId":    payload.PortalID,
		"playerId":    playerID,
		"slot":        payload.Slot,
		"level":       payload.Level,
		"energy":      result.Portal.Energy,
		"slotEnergy":  slotEnergy,
		"faction":     result.Portal.Faction,
		"portalLevel": result.Portal.Level,
		"version":     result.NewVersion,
	}))
}

func handlePlayerDeployMod(hub *Hub, client *Client, msg Message) {
	var payload DeployModPayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid deploy mod payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" || payload.PortalID == "" || payload.ModType == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId, portalId, or modType"}))
		return
	}
	if hub.services == nil || hub.services.Portal == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	result, err := hub.services.Portal.DeployMod(portal.DeployModCmd{
		PlayerID:        playerID,
		PortalID:        payload.PortalID,
		Slot:            payload.Slot,
		ModType:         payload.ModType,
		Rarity:          payload.Rarity,
		ExpectedVersion: payload.ExpectedVersion,
	}, time.Now())
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
	client.Send(NewMessage(MessagePlayerResourceUpdate, msg.ID, PlayerResourceUpdatePayload{
		PlayerID: result.Player.ID,
		APGained: result.APGained,
		XMDelta:  -result.XMCost,
		InventoryDelta: map[string]int{
			gameplay.ItemIDMod(result.ModType, result.Rarity): -1,
		},
	}))
	hub.Broadcast(NewMessage(MessagePortalUpdate, msg.ID, map[string]interface{}{
		"portalId": payload.PortalID,
		"playerId": playerID,
		"modSlot":  payload.Slot,
		"modType":  result.ModType,
		"rarity":   result.Rarity,
		"version":  result.NewVersion,
	}))
}

func handlePlayerChargePortal(hub *Hub, client *Client, msg Message) {
	var payload ChargePortalPayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid charge payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" || payload.PortalID == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId or portalId"}))
		return
	}
	if hub.services == nil || hub.services.Portal == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	result, err := hub.services.Portal.Charge(portal.ChargeCmd{PlayerID: playerID, PortalID: payload.PortalID, Amount: payload.Amount}, time.Now())
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
	client.Send(NewMessage(MessagePlayerResourceUpdate, msg.ID, PlayerResourceUpdatePayload{
		PlayerID: result.Player.ID,
		XMDelta:  -result.XMCost,
	}))
	hub.Broadcast(NewMessage(MessagePortalUpdate, msg.ID, map[string]interface{}{
		"portalId": payload.PortalID,
		"energy":   result.Portal.Energy,
	}))
}
