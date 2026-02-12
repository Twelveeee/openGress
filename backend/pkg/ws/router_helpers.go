package ws

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/gameplay"
	serr "github.com/Twelveeee/openGress/backend/pkg/service/errors"
	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/google/uuid"
)

func decodePayload(input interface{}, output interface{}) error {
	raw, err := json.Marshal(input)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, output)
}

func newStatePlayer(playerID string, lat, lon float64) state.Player {
	return state.Player{
		ID:            playerID,
		Username:      playerID,
		Faction:       "NEUTRAL",
		Level:         1,
		XM:            gameplay.MaxXMForLevel(1),
		MaxXM:         gameplay.MaxXMForLevel(1),
		Inventory:     make(map[string]int),
		Position:      state.Position{Latitude: lat, Longitude: lon},
		HackCooldowns: make(map[string]time.Time),
		UpdatedAt:     time.Now(),
	}
}

func resolvePlayerID(payloadID string, client *Client) string {
	if payloadID != "" {
		return payloadID
	}
	return client.PlayerID()
}

func appendLog(gameState *state.GameState, entry state.LogEntry) {
	if gameState == nil || gameState.Logs == nil {
		return
	}
	if entry.ID == "" {
		entry.ID = uuid.New().String()
	}
	gameState.Logs.Add(entry)
}

func serviceErrCode(err error) int {
	if err == nil {
		return api.ErrCodeInternal
	}
	var se *serr.ServiceError
	if !errors.As(err, &se) {
		return api.ErrCodeInternal
	}
	switch se.Kind {
	case serr.KindBadRequest:
		return api.ErrCodeBadRequest
	case serr.KindUnauthorized:
		return api.ErrCodeUnauthorized
	case serr.KindNotFound:
		return api.ErrCodeNotFound
	case serr.KindConflict:
		return api.ErrCodeConflict
	default:
		return api.ErrCodeInternal
	}
}

func sendServiceError(client *Client, msgID string, err error) {
	if err == nil {
		return
	}
	client.Send(NewMessage(MessageError, msgID, ErrorPayload{Code: serviceErrCode(err), Message: err.Error()}))
}
