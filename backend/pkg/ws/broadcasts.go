package ws

import (
	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func broadcastPortalChanges(hub *Hub, portals []state.Portal) {
	if hub.state == nil || len(portals) == 0 {
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
		visibleStates := collectPortalStatesInBounds(portals, *player.View)
		if len(visibleStates) == 0 {
			continue
		}
		visible := toMapPortals(visibleStates)
		if len(visible) > 1 {
			sendMapUpdates(client, visible, false, MessageMapUpdate)
		}
		for _, portalState := range visibleStates {
			client.Send(NewMessage(MessagePortalUpdate, "", buildPortalUpdatePayload(portalState)))
		}
	}
}

func collectPortalStatesInBounds(portals []state.Portal, bounds state.Bounds) []state.Portal {
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
	return filtered
}

func buildPortalUpdatePayload(portal state.Portal) PortalUpdatePayload {
	resonators := make(map[int]PortalUpdateResonator, len(portal.Resonators))
	for slot, entry := range portal.Resonators {
		resonators[slot] = PortalUpdateResonator{
			Slot:     entry.Slot,
			Level:    entry.Level,
			Energy:   entry.Energy,
			PlayerID: entry.PlayerID,
			Version:  entry.Version,
		}
	}
	mods := make(map[int]PortalUpdateMod, len(portal.Mods))
	for slot, entry := range portal.Mods {
		mods[slot] = PortalUpdateMod{
			Slot:     entry.Slot,
			ModType:  entry.ModType,
			Rarity:   entry.Rarity,
			PlayerID: entry.PlayerID,
			Version:  entry.Version,
		}
	}
	owner := ""
	for _, resonator := range portal.Resonators {
		if resonator.PlayerID != "" {
			owner = resonator.PlayerID
			break
		}
	}
	return PortalUpdatePayload{
		PortalID:   portal.ID,
		Faction:    portal.Faction,
		Energy:     portal.Energy,
		Level:      portal.Level,
		Owner:      owner,
		Resonators: resonators,
		Mods:       mods,
	}
}

func broadcastLinkRemovals(hub *Hub, links []state.Link) {
	for _, link := range links {
		broadcastLinkUpdate(hub, LinkUpdatePayload{
			LinkID:       link.ID,
			FromPortalID: link.FromPortalID,
			ToPortalID:   link.ToPortalID,
			FromLat:      link.FromPosition.Latitude,
			FromLon:      link.FromPosition.Longitude,
			ToLat:        link.ToPosition.Latitude,
			ToLon:        link.ToPosition.Longitude,
			Removed:      true,
		})
	}
}

func broadcastFieldRemovals(hub *Hub, fields []state.Field) {
	if hub == nil || hub.state == nil {
		return
	}
	for _, field := range fields {
		p1, ok1 := hub.state.Portals.Get(field.PortalIDs[0])
		p2, ok2 := hub.state.Portals.Get(field.PortalIDs[1])
		p3, ok3 := hub.state.Portals.Get(field.PortalIDs[2])
		if !ok1 || !ok2 || !ok3 {
			continue
		}
		broadcastFieldUpdate(hub, field, p1.Position, p2.Position, p3.Position, true)
	}
}

func broadcastTopologySnapshot(hub *Hub) {
	if hub == nil || hub.state == nil {
		return
	}
	portals := hub.state.Portals.List()
	links := hub.state.Links.List()
	fields := hub.state.Fields.List()

	for client := range hub.clients {
		playerID := client.PlayerID()
		if playerID == "" {
			continue
		}
		player, ok := hub.state.Players.Get(playerID)
		if !ok || player.View == nil {
			continue
		}
		sendMapUpdates(client, collectPortalsInBounds(portals, *player.View), true, MessageMapUpdate)
		sendLinkUpdatesForView(client, links, *player.View)
		sendFieldUpdatesForView(client, fields, hub.state.Portals, *player.View)
	}
}
