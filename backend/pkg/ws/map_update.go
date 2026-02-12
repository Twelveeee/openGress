package ws

import (
	pbws "github.com/Twelveeee/openGress/backend/pkg/pb/ws"
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

const mapUpdateBatchSize = 200

type MapPortal = pbws.MapPortal
type MapUpdatePayload = pbws.MapUpdatePayload

func toMapPortals(portals []state.Portal) []MapPortal {
	result := make([]MapPortal, 0, len(portals))
	for _, portal := range portals {
		title := portal.Title
		if title == "" {
			title = portal.ID
		}
		result = append(result, MapPortal{
			ID:        portal.ID,
			Title:     title,
			CoverURL:  portal.CoverURL,
			Latitude:  portal.Position.Latitude,
			Longitude: portal.Position.Longitude,
			Faction:   portal.Faction,
			Level:     portal.Level,
			Energy:    portal.Energy,
		})
	}
	return result
}

func collectPortalsInBounds(portals []state.Portal, bounds state.Bounds) []MapPortal {
	filtered := make([]state.Portal, 0, len(portals))
	for _, portal := range portals {
		if portal.Position.Latitude < bounds.MinLat || portal.Position.Latitude > bounds.MaxLat {
			continue
		}
		if portal.Position.Longitude < bounds.MinLon || portal.Position.Longitude > bounds.MaxLon {
			continue
		}
		filtered = append(filtered, portal)
	}
	return toMapPortals(filtered)
}

func sendMapUpdates(client *Client, portals []MapPortal, full bool, msgType string) {
	if len(portals) == 0 {
		client.Send(NewMessage(msgType, "", MapUpdatePayload{
			Full:       full,
			BatchIndex: 1,
			BatchTotal: 1,
			Portals:    []MapPortal{},
		}))
		return
	}
	total := (len(portals) + mapUpdateBatchSize - 1) / mapUpdateBatchSize
	for i := 0; i < total; i++ {
		start := i * mapUpdateBatchSize
		end := start + mapUpdateBatchSize
		if end > len(portals) {
			end = len(portals)
		}
		client.Send(NewMessage(msgType, "", MapUpdatePayload{
			Full:       full,
			BatchIndex: i + 1,
			BatchTotal: total,
			Portals:    portals[start:end],
		}))
	}
}
