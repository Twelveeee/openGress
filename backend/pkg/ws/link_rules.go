package ws

import (
	"github.com/Twelveeee/openGress/backend/pkg/geo"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func broadcastFieldUpdate(hub *Hub, field state.Field, p1, p2, p3 state.Position, removed bool) {
	if hub == nil || hub.state == nil {
		return
	}
	for client := range hub.clients {
		playerID := client.PlayerID()
		if playerID == "" {
			continue
		}
		player, ok := hub.state.Players.Get(playerID)
		if !ok || player.View == nil {
			continue
		}
		if fieldVisibleInBounds(p1, p2, p3, *player.View) {
			client.Send(NewMessage(MessageFieldCreated, "", FieldCreatedPayload{
				FieldID:   field.ID,
				PortalIDs: []string{field.PortalIDs[0], field.PortalIDs[1], field.PortalIDs[2]},
				MU:        field.MU,
				Layer:     field.Layer,
				Faction:   field.Faction,
				Removed:   removed,
			}))
		}
	}
}

func fieldVisibleInBounds(a, b, c state.Position, bounds state.Bounds) bool {
	if pointInBounds(a, bounds) || pointInBounds(b, bounds) || pointInBounds(c, bounds) {
		return true
	}
	if linkVisibleInBounds(a, b, bounds) || linkVisibleInBounds(b, c, bounds) || linkVisibleInBounds(c, a, bounds) {
		return true
	}
	return false
}

func linkVisibleInBounds(from, to state.Position, bounds state.Bounds) bool {
	return geo.LinkVisibleInBounds(from, to, bounds)
}

func pointInBounds(pos state.Position, bounds state.Bounds) bool {
	return geo.PointInBounds(pos, bounds)
}
