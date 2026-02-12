package ws

import (
	"log/slog"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	"github.com/Twelveeee/openGress/backend/pkg/service/common"
	playersvc "github.com/Twelveeee/openGress/backend/pkg/service/player"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/google/uuid"
)

func handleConnect(hub *Hub, client *Client, msg Message) {
	var payload ConnectPayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid connect payload"}))
		return
	}
	if payload.AuthToken == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeUnauthorized, Message: "missing authToken"}))
		return
	}
	if hub.auth == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "auth unavailable"}))
		return
	}
	user, ok := hub.auth.UserByAccessToken(payload.AuthToken)
	if !ok {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeUnauthorized, Message: "invalid authToken"}))
		return
	}
	if hub.state != nil && hub.state.Admin != nil {
		if _, banned := hub.state.Admin.Banned[user.PlayerID]; banned {
			client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeUnauthorized, Message: "banned"}))
			return
		}
	}
	client.SetPlayerID(user.PlayerID)
	client.SetSessionID(uuid.New().String())

	var initialPlayerState *PlayerStatePayload
	if hub.state != nil {
		player, exists := hub.state.Players.Get(user.PlayerID)
		if !exists {
			spawn := gameplay.RandomPositionInBounds(hub.gameplay.MapBounds, nil)
			player = newStatePlayer(user.PlayerID, spawn.Latitude, spawn.Longitude)
		}
		if playersvc.MapBoundsConfigured(hub.gameplay.MapBounds) {
			reason := ""
			switch {
			case isZeroPosition(player.Position):
				reason = "zero_position"
			case !gameplay.PositionWithinBounds(player.Position, hub.gameplay.MapBounds):
				reason = "out_of_bounds"
			}
			if reason != "" {
				oldPos := player.Position
				if reason == "zero_position" {
					player.Position = gameplay.RandomPositionInBounds(hub.gameplay.MapBounds, nil)
				} else {
					if repaired, changed := gameplay.EnsurePositionInBounds(player.Position, hub.gameplay.MapBounds, nil); changed {
						player.Position = repaired
					}
				}
				player.UpdatedAt = time.Now()
				slog.Info("repair player position",
					"playerId", player.ID,
					"oldPos", oldPos,
					"newPos", player.Position,
					"reason", reason,
				)
			}
		}
		player.Username = user.Username
		common.SyncPlayerProgression(&player)
		hub.state.Players.Upsert(player)
		payload := hub.services.Player.BuildPlayerState(player, hub.playerStatePushInterval())
		initialPlayerState = &payload
	}

	client.Send(NewMessage(MessageConnected, msg.ID, map[string]interface{}{
		"sessionId":  client.SessionID(),
		"serverTime": time.Now().UnixMilli(),
	}))
	if initialPlayerState != nil {
		// CONNECT 成功后立即下发一次权威位置，避免前端首帧等待 tick。
		client.Send(NewMessage(MessagePlayerState, "", *initialPlayerState))
	}
}

func isZeroPosition(pos state.Position) bool {
	return pos.Latitude == 0 && pos.Longitude == 0
}
