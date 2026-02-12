package ws

import (
	pbws "github.com/Twelveeee/openGress/backend/pkg/pb/ws"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

type LinkUpdatePayload = pbws.LinkUpdatePayload

func broadcastLinkUpdate(hub *Hub, payload LinkUpdatePayload) {
	if hub.state == nil {
		return
	}
	fromPos := state.Position{Latitude: payload.FromLat, Longitude: payload.FromLon}
	toPos := state.Position{Latitude: payload.ToLat, Longitude: payload.ToLon}

	for client := range hub.clients {
		playerID := client.PlayerID()
		if playerID == "" {
			continue
		}
		player, ok := hub.state.Players.Get(playerID)
		if !ok || player.View == nil {
			continue
		}
		if linkVisibleInBounds(fromPos, toPos, *player.View) {
			client.Send(NewMessage(MessageLinkUpdate, "", payload))
		}
	}
}
