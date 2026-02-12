package database

import (
	"encoding/json"
	"sort"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/state"
	"github.com/google/uuid"
)

type modelSnapshot struct {
	players     []PlayerModel
	inventories []PlayerInventoryModel
	portals     []PortalModel
	links       []LinkModel
	fields      []FieldModel
	logs        []LogModel
	adminBans   []AdminBanModel
}

func snapshotToModels(snapshot state.Snapshot) (modelSnapshot, error) {
	out := modelSnapshot{}

	out.players = make([]PlayerModel, 0, len(snapshot.Players))
	out.inventories = make([]PlayerInventoryModel, 0)
	for _, player := range snapshot.Players {
		model, err := toPlayerModel(player)
		if err != nil {
			return modelSnapshot{}, err
		}
		out.players = append(out.players, model)
		out.inventories = append(out.inventories, toPlayerInventoryModels(player)...)
	}

	out.portals = make([]PortalModel, 0, len(snapshot.Portals))
	for _, portal := range snapshot.Portals {
		model, err := toPortalModel(portal)
		if err != nil {
			return modelSnapshot{}, err
		}
		out.portals = append(out.portals, model)
	}

	out.links = make([]LinkModel, 0, len(snapshot.Links))
	for _, link := range snapshot.Links {
		out.links = append(out.links, LinkModel{
			ID:           link.ID,
			FromPortalID: link.FromPortalID,
			ToPortalID:   link.ToPortalID,
			FromLat:      link.FromPosition.Latitude,
			FromLon:      link.FromPosition.Longitude,
			ToLat:        link.ToPosition.Latitude,
			ToLon:        link.ToPosition.Longitude,
			CreatedAt:    link.CreatedAt,
		})
	}

	out.fields = make([]FieldModel, 0, len(snapshot.Fields))
	for _, field := range snapshot.Fields {
		out.fields = append(out.fields, FieldModel{
			ID:        field.ID,
			Portal1ID: field.PortalIDs[0],
			Portal2ID: field.PortalIDs[1],
			Portal3ID: field.PortalIDs[2],
			Faction:   field.Faction,
			MU:        field.MU,
			Layer:     field.Layer,
			CreatedAt: field.CreatedAt,
		})
	}

	out.logs = make([]LogModel, 0, len(snapshot.Logs))
	for _, entry := range snapshot.Logs {
		id := entry.ID
		if id == "" {
			id = uuid.NewString()
		}
		out.logs = append(out.logs, LogModel{
			ID:        id,
			Type:      entry.Type,
			PlayerID:  entry.PlayerID,
			PortalID:  entry.PortalID,
			Faction:   entry.Faction,
			MU:        entry.MU,
			Message:   entry.Message,
			Timestamp: entry.Timestamp,
		})
	}

	if snapshot.Admin != nil {
		out.adminBans = make([]AdminBanModel, 0, len(snapshot.Admin.Banned))
		for playerID, reason := range snapshot.Admin.Banned {
			out.adminBans = append(out.adminBans, AdminBanModel{
				PlayerID:  playerID,
				Reason:    reason,
				UpdatedAt: snapshot.Timestamp,
			})
		}
	}

	return out, nil
}

func modelsToSnapshot(models modelSnapshot) (state.Snapshot, error) {
	snapshot := state.Snapshot{
		Version:   2,
		Timestamp: time.Now(),
		Players:   make([]state.Player, 0, len(models.players)),
		Portals:   make([]state.Portal, 0, len(models.portals)),
		Links:     make([]state.Link, 0, len(models.links)),
		Fields:    make([]state.Field, 0, len(models.fields)),
		Logs:      make([]state.LogEntry, 0, len(models.logs)),
	}

	for _, model := range models.players {
		player, err := fromPlayerModel(model)
		if err != nil {
			return state.Snapshot{}, err
		}
		snapshot.Players = append(snapshot.Players, player)
	}
	playerIndex := make(map[string]int, len(snapshot.Players))
	for idx := range snapshot.Players {
		playerIndex[snapshot.Players[idx].ID] = idx
	}
	for _, inventory := range models.inventories {
		idx, ok := playerIndex[inventory.PlayerID]
		if !ok {
			continue
		}
		if snapshot.Players[idx].Inventory == nil {
			snapshot.Players[idx].Inventory = make(map[string]int)
		}
		snapshot.Players[idx].Inventory[inventory.ItemID] = inventory.Amount
	}

	for _, model := range models.portals {
		portal, err := fromPortalModel(model)
		if err != nil {
			return state.Snapshot{}, err
		}
		snapshot.Portals = append(snapshot.Portals, portal)
	}

	for _, model := range models.links {
		snapshot.Links = append(snapshot.Links, state.Link{
			ID:           model.ID,
			FromPortalID: model.FromPortalID,
			ToPortalID:   model.ToPortalID,
			FromPosition: state.Position{Latitude: model.FromLat, Longitude: model.FromLon},
			ToPosition:   state.Position{Latitude: model.ToLat, Longitude: model.ToLon},
			CreatedAt:    model.CreatedAt,
		})
	}

	for _, model := range models.fields {
		snapshot.Fields = append(snapshot.Fields, state.Field{
			ID:        model.ID,
			PortalIDs: [3]string{model.Portal1ID, model.Portal2ID, model.Portal3ID},
			Faction:   model.Faction,
			MU:        model.MU,
			Layer:     model.Layer,
			CreatedAt: model.CreatedAt,
		})
	}

	for _, model := range models.logs {
		snapshot.Logs = append(snapshot.Logs, state.LogEntry{
			ID:        model.ID,
			Type:      model.Type,
			PlayerID:  model.PlayerID,
			PortalID:  model.PortalID,
			Faction:   model.Faction,
			MU:        model.MU,
			Message:   model.Message,
			Timestamp: model.Timestamp,
		})
	}

	if len(models.adminBans) > 0 {
		admin := state.NewAdminState()
		for _, model := range models.adminBans {
			admin.Banned[model.PlayerID] = model.Reason
		}
		snapshot.Admin = admin
	}

	return snapshot, nil
}

func toPlayerModel(player state.Player) (PlayerModel, error) {
	cooldowns, err := marshalJSON(player.HackCooldowns)
	if err != nil {
		return PlayerModel{}, err
	}
	view, err := marshalJSON(player.View)
	if err != nil {
		return PlayerModel{}, err
	}
	viewTiles, err := marshalJSON(player.ViewTiles)
	if err != nil {
		return PlayerModel{}, err
	}
	target, err := marshalJSON(player.Target)
	if err != nil {
		return PlayerModel{}, err
	}

	return PlayerModel{
		ID:             player.ID,
		Username:       player.Username,
		Faction:        player.Faction,
		Level:          player.Level,
		AP:             player.AP,
		XM:             player.XM,
		MaxXM:          player.MaxXM,
		PortalCaptures: player.PortalCaptures,
		LinksCreated:   player.LinksCreated,
		FieldsCreated:  player.FieldsCreated,
		MUTotal:        player.MUTotal,
		PosLat:         player.Position.Latitude,
		PosLon:         player.Position.Longitude,
		AutoHack:       player.AutoHack,
		CooldownsJSON:  cooldowns,
		ViewJSON:       view,
		ViewTilesJSON:  viewTiles,
		TargetJSON:     target,
		UpdatedAt:      player.UpdatedAt,
	}, nil
}

func toPlayerInventoryModels(player state.Player) []PlayerInventoryModel {
	if len(player.Inventory) == 0 {
		return nil
	}
	itemIDs := make([]string, 0, len(player.Inventory))
	for itemID := range player.Inventory {
		itemIDs = append(itemIDs, itemID)
	}
	sort.Strings(itemIDs)

	out := make([]PlayerInventoryModel, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		amount := player.Inventory[itemID]
		if amount <= 0 {
			continue
		}
		out = append(out, PlayerInventoryModel{
			PlayerID: player.ID,
			ItemID:   itemID,
			Amount:   amount,
		})
	}
	return out
}

func fromPlayerModel(model PlayerModel) (state.Player, error) {
	player := state.Player{
		ID:             model.ID,
		Username:       model.Username,
		Faction:        model.Faction,
		Level:          model.Level,
		AP:             model.AP,
		XM:             model.XM,
		MaxXM:          model.MaxXM,
		PortalCaptures: model.PortalCaptures,
		LinksCreated:   model.LinksCreated,
		FieldsCreated:  model.FieldsCreated,
		MUTotal:        model.MUTotal,
		Position:       state.Position{Latitude: model.PosLat, Longitude: model.PosLon},
		AutoHack:       model.AutoHack,
		UpdatedAt:      model.UpdatedAt,
	}

	if err := unmarshalJSON(model.LegacyInventoryJSON, &player.Inventory); err != nil {
		return state.Player{}, err
	}
	if err := unmarshalJSON(model.CooldownsJSON, &player.HackCooldowns); err != nil {
		return state.Player{}, err
	}
	if err := unmarshalJSON(model.ViewJSON, &player.View); err != nil {
		return state.Player{}, err
	}
	if err := unmarshalJSON(model.ViewTilesJSON, &player.ViewTiles); err != nil {
		return state.Player{}, err
	}
	if err := unmarshalJSON(model.TargetJSON, &player.Target); err != nil {
		return state.Player{}, err
	}

	return player, nil
}

func toPortalModel(portal state.Portal) (PortalModel, error) {
	resonators, err := marshalJSON(portal.Resonators)
	if err != nil {
		return PortalModel{}, err
	}
	mods, err := marshalJSON(portal.Mods)
	if err != nil {
		return PortalModel{}, err
	}
	title := portal.Title
	if title == "" {
		title = portal.ID
	}

	return PortalModel{
		ID:             portal.ID,
		Title:          title,
		CoverURL:       portal.CoverURL,
		Latitude:       portal.Position.Latitude,
		Longitude:      portal.Position.Longitude,
		Faction:        portal.Faction,
		Level:          portal.Level,
		Energy:         portal.Energy,
		ResonatorsJSON: resonators,
		ModsJSON:       mods,
		UpdatedAt:      portal.UpdatedAt,
	}, nil
}

func fromPortalModel(model PortalModel) (state.Portal, error) {
	portal := state.Portal{
		ID:        model.ID,
		Title:     model.Title,
		CoverURL:  model.CoverURL,
		Position:  state.Position{Latitude: model.Latitude, Longitude: model.Longitude},
		Faction:   model.Faction,
		Level:     model.Level,
		Energy:    model.Energy,
		UpdatedAt: model.UpdatedAt,
	}
	if portal.Title == "" {
		portal.Title = portal.ID
	}
	if err := unmarshalJSON(model.ResonatorsJSON, &portal.Resonators); err != nil {
		return state.Portal{}, err
	}
	if err := unmarshalJSON(model.ModsJSON, &portal.Mods); err != nil {
		return state.Portal{}, err
	}
	return portal, nil
}

func marshalJSON(value interface{}) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if string(data) == "null" {
		return nil, nil
	}
	return data, nil
}

func unmarshalJSON(data []byte, value interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, value)
}
