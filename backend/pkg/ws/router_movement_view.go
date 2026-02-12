package ws

import (
	"fmt"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func handlePlayerTargetUpdate(hub *Hub, client *Client, msg Message) {
	var payload PlayerTargetUpdatePayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid player target payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId"}))
		return
	}
	if hub.services == nil || hub.services.Player == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	target := state.Position{Latitude: payload.Latitude, Longitude: payload.Longitude}
	if err := hub.services.Player.SetTarget(playerID, target); err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
}

func handlePlayerViewUpdate(hub *Hub, client *Client, msg Message) {
	var payload PlayerViewUpdatePayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid view update payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId"}))
		return
	}
	if hub.services == nil || hub.services.Player == nil || hub.services.Map == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	bounds, err := toStateBounds(payload.Bounds)
	if err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid bounds"}))
		return
	}
	normalized, err := hub.services.Player.UpdateView(playerID, bounds, payload.Tiles)
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
	if normalized != nil {
		portals := hub.services.Map.CollectPortals(*normalized)
		sendMapUpdates(client, toMapPortals(portals), true, MessageMapUpdate)
		sendLinkUpdatesForView(client, hub.services.Map.CollectLinks(*normalized), *normalized)
		sendFieldUpdatesForView(client, hub.services.Map.CollectFields(*normalized), hub.state.Portals, *normalized)
	}
}

func handlePlayerLocalUpdateDeprecated(client *Client, msg Message) {
	client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "PLAYER_LOCAL_UPDATE deprecated"}))
}

func toStateBounds(bounds *ViewBoundsPayload) (*state.Bounds, error) {
	if bounds == nil {
		return nil, nil
	}
	if bounds.MinLat > bounds.MaxLat || bounds.MinLon > bounds.MaxLon {
		return nil, fmt.Errorf("invalid bounds")
	}
	return &state.Bounds{
		MinLat: bounds.MinLat,
		MaxLat: bounds.MaxLat,
		MinLon: bounds.MinLon,
		MaxLon: bounds.MaxLon,
	}, nil
}

func sendLinkUpdatesForView(client *Client, links []state.Link, bounds state.Bounds) {
	for _, link := range links {
		if linkVisibleInBounds(link.FromPosition, link.ToPosition, bounds) {
			client.Send(NewMessage(MessageLinkUpdate, "", LinkUpdatePayload{
				LinkID:       link.ID,
				FromPortalID: link.FromPortalID,
				ToPortalID:   link.ToPortalID,
				FromLat:      link.FromPosition.Latitude,
				FromLon:      link.FromPosition.Longitude,
				ToLat:        link.ToPosition.Latitude,
				ToLon:        link.ToPosition.Longitude,
			}))
		}
	}
}

func sendFieldUpdatesForView(client *Client, fields []state.Field, portals state.PortalRepository, bounds state.Bounds) {
	for _, field := range fields {
		p1, ok1 := portals.Get(field.PortalIDs[0])
		p2, ok2 := portals.Get(field.PortalIDs[1])
		p3, ok3 := portals.Get(field.PortalIDs[2])
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		if fieldVisibleInBounds(p1.Position, p2.Position, p3.Position, bounds) {
			client.Send(NewMessage(MessageFieldCreated, "", FieldCreatedPayload{
				FieldID:   field.ID,
				PortalIDs: []string{field.PortalIDs[0], field.PortalIDs[1], field.PortalIDs[2]},
				MU:        field.MU,
				Layer:     field.Layer,
				Faction:   field.Faction,
			}))
		}
	}
}
