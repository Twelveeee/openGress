package api

import "github.com/Twelveeee/openGress/backend/pkg/state"

type adminPlayerRequest struct {
	PlayerID string `json:"playerId"`
}

type adminAnnounceRequest struct {
	Message string `json:"message"`
}

type adminBanRequest struct {
	PlayerID string `json:"playerId"`
	Reason   string `json:"reason"`
}

type adminGrantItemRequest struct {
	ItemID string `json:"itemId"`
	Amount int    `json:"amount"`
}

type adminSetFactionRequest struct {
	Faction string `json:"faction"`
}

type adminSetLevelRequest struct {
	Level int `json:"level"`
}

type adminPortalRequest struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	CoverURL string  `json:"cover_url"`
	Lat      float64 `json:"lat"`
	Lon      float64 `json:"lon"`
	Faction  string  `json:"faction"`
}

type adminPortalPatchRequest struct {
	Lat      *float64 `json:"lat"`
	Lon      *float64 `json:"lon"`
	Title    *string  `json:"title"`
	CoverURL *string  `json:"cover_url"`
	Faction  *string  `json:"faction"`
	Level    *int     `json:"level"`
	Energy   *int     `json:"energy"`
}

type adminGCLogsRequest struct {
	Keep int `json:"keep"`
}

func appendLog(state *state.GameState, entry state.LogEntry) {
	if state == nil || state.Logs == nil {
		return
	}
	state.Logs.Add(entry)
}
