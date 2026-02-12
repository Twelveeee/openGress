package ws

import (
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/api"
	linksvc "github.com/Twelveeee/openGress/backend/pkg/service/link"
)

func handlePlayerCreateLink(hub *Hub, client *Client, msg Message) {
	var payload CreateLinkPayload
	if err := decodePayload(msg.Data, &payload); err != nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "invalid create link payload"}))
		return
	}
	playerID := resolvePlayerID(payload.PlayerID, client)
	if playerID == "" || payload.FromPortalID == "" || payload.ToPortalID == "" {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeBadRequest, Message: "missing playerId or portal ids"}))
		return
	}
	if hub.services == nil || hub.services.Link == nil || hub.services.Field == nil {
		client.Send(NewMessage(MessageError, msg.ID, ErrorPayload{Code: api.ErrCodeInternal, Message: "state unavailable"}))
		return
	}
	result, err := hub.services.Link.CreateLink(linksvc.CreateLinkCmd{
		PlayerID:     playerID,
		FromPortalID: payload.FromPortalID,
		ToPortalID:   payload.ToPortalID,
	}, time.Now())
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}

	linkState := result.Link
	broadcastLinkUpdate(hub, LinkUpdatePayload{
		LinkID:       linkState.ID,
		FromPortalID: linkState.FromPortalID,
		ToPortalID:   linkState.ToPortalID,
		FromLat:      linkState.FromPosition.Latitude,
		FromLon:      linkState.FromPosition.Longitude,
		ToLat:        linkState.ToPosition.Latitude,
		ToLon:        linkState.ToPosition.Longitude,
	})

	fields, err := hub.services.Field.CreateFromNewLink(linkState, result.Player.ID, time.Now())
	if err != nil {
		sendServiceError(client, msg.ID, err)
		return
	}
	for _, fieldState := range fields {
		p1, ok1 := hub.state.Portals.Get(fieldState.PortalIDs[0])
		p2, ok2 := hub.state.Portals.Get(fieldState.PortalIDs[1])
		p3, ok3 := hub.state.Portals.Get(fieldState.PortalIDs[2])
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		broadcastFieldUpdate(hub, fieldState, p1.Position, p2.Position, p3.Position, false)
	}
}
