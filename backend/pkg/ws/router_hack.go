package ws

import (
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
)

func handlePlayerToggleAutoHack(hub *Hub, client *Client, msg Message) {
	var payload AutoHackTogglePayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid auto hack payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId"}))
		return
	}
	if hub.services == nil || hub.services.Hack == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	if err := hub.services.Hack.ToggleAutoHack(playerID, payload.Enabled, time.Now()); err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
}

func handleHackPortal(hub *Hub, client *Client, msg Message) {
	var payload HackPortalPayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid hack payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" || payload.PortalID == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId or portalId"}))
		return
	}
	if hub.services == nil || hub.services.Hack == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	result, err := hub.services.Hack.HackPortal(playerID, payload.PortalID, time.Now())
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
	client.Send(NewMessage(MessageHackResult, msg.ID, HackResultPayload{
		PortalID:        result.PortalID,
		Success:         result.Success,
		ItemsGained:     result.ItemsGained,
		APGained:        result.APGained,
		CooldownSeconds: result.CooldownSeconds,
	}))
}
