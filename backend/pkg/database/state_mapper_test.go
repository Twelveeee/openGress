package database

import (
	"reflect"
	"testing"
	"time"

	"github.com/Twelveeee/openGress/backend/pkg/state"
)

func TestSnapshotMapperRoundTrip(t *testing.T) {
	ts := time.Date(2026, 2, 6, 10, 0, 0, 0, time.UTC)
	snapshot := state.Snapshot{
		Version:   2,
		Timestamp: ts,
		Players: []state.Player{
			{
				ID:             "p1",
				Username:       "u1",
				Faction:        "RESISTANCE",
				Level:          3,
				AP:             1200,
				XM:             1500,
				MaxXM:          3000,
				PortalCaptures: 1,
				LinksCreated:   2,
				FieldsCreated:  3,
				MUTotal:        400,
				Inventory:      map[string]int{"XMP_L1": 2},
				Position:       state.Position{Latitude: 39.9, Longitude: 116.4},
				Target:         &state.Position{Latitude: 39.91, Longitude: 116.41},
				View:           &state.Bounds{MinLat: 39.8, MaxLat: 40.0, MinLon: 116.3, MaxLon: 116.5},
				ViewTiles:      []string{"t1", "t2"},
				AutoHack:       true,
				HackCooldowns:  map[string]time.Time{"portal-1": ts},
				UpdatedAt:      ts,
			},
		},
		Portals: []state.Portal{
			{
				ID:       "portal-1",
				Title:    "Portal One",
				CoverURL: "https://example.com/portal-1.jpg",
				Position: state.Position{Latitude: 39.9, Longitude: 116.4},
				Faction:  "RESISTANCE",
				Level:    4,
				Energy:   2300,
				Resonators: map[int]state.ResonatorSlot{
					1: {Slot: 1, Level: 4, Energy: 1000, PlayerID: "p1", Version: 2, UpdatedAt: ts},
				},
				Mods: map[int]state.ModSlot{
					1: {Slot: 1, ModType: "SHIELD", Rarity: "COMMON", PlayerID: "p1", Version: 1, UpdatedAt: ts},
				},
				UpdatedAt: ts,
			},
		},
		Links: []state.Link{
			{
				ID:           "l1",
				FromPortalID: "portal-1",
				ToPortalID:   "portal-2",
				FromPosition: state.Position{Latitude: 39.9, Longitude: 116.4},
				ToPosition:   state.Position{Latitude: 39.91, Longitude: 116.41},
				CreatedAt:    ts,
			},
		},
		Fields: []state.Field{
			{
				ID:        "f1",
				PortalIDs: [3]string{"portal-1", "portal-2", "portal-3"},
				Faction:   "RESISTANCE",
				MU:        800,
				Layer:     1,
				CreatedAt: ts,
			},
		},
		Logs: []state.LogEntry{
			{
				ID:        "log-1",
				Type:      "CAPTURE",
				PlayerID:  "p1",
				PortalID:  "portal-1",
				Faction:   "RESISTANCE",
				MU:        0,
				Message:   "captured",
				Timestamp: ts,
			},
		},
		Admin: &state.AdminState{Banned: map[string]string{"p2": "spam"}},
	}

	models, err := snapshotToModels(snapshot)
	if err != nil {
		t.Fatalf("snapshotToModels: %v", err)
	}

	roundTrip, err := modelsToSnapshot(models)
	if err != nil {
		t.Fatalf("modelsToSnapshot: %v", err)
	}

	if len(roundTrip.Players) != 1 || len(roundTrip.Portals) != 1 || len(roundTrip.Links) != 1 || len(roundTrip.Fields) != 1 || len(roundTrip.Logs) != 1 {
		t.Fatalf("unexpected counts: players=%d portals=%d links=%d fields=%d logs=%d", len(roundTrip.Players), len(roundTrip.Portals), len(roundTrip.Links), len(roundTrip.Fields), len(roundTrip.Logs))
	}
	if !reflect.DeepEqual(snapshot.Players[0].Inventory, roundTrip.Players[0].Inventory) {
		t.Fatalf("inventory mismatch")
	}
	if !reflect.DeepEqual(snapshot.Players[0].HackCooldowns, roundTrip.Players[0].HackCooldowns) {
		t.Fatalf("cooldowns mismatch")
	}
	if !reflect.DeepEqual(snapshot.Portals[0].Resonators, roundTrip.Portals[0].Resonators) {
		t.Fatalf("resonators mismatch")
	}
	if !reflect.DeepEqual(snapshot.Portals[0].Mods, roundTrip.Portals[0].Mods) {
		t.Fatalf("mods mismatch")
	}
	if snapshot.Portals[0].Title != roundTrip.Portals[0].Title {
		t.Fatalf("portal title mismatch")
	}
	if snapshot.Portals[0].CoverURL != roundTrip.Portals[0].CoverURL {
		t.Fatalf("portal cover url mismatch")
	}
	if roundTrip.Admin == nil || roundTrip.Admin.Banned["p2"] != "spam" {
		t.Fatalf("admin bans mismatch")
	}
}

func TestSnapshotMapperRoundTripNilFields(t *testing.T) {
	ts := time.Date(2026, 2, 6, 10, 0, 0, 0, time.UTC)
	snapshot := state.Snapshot{
		Version:   2,
		Timestamp: ts,
		Players: []state.Player{
			{
				ID:        "p-nil",
				Username:  "nil-user",
				Faction:   "NEUTRAL",
				Level:     1,
				XM:        1000,
				MaxXM:     1000,
				Position:  state.Position{Latitude: 0, Longitude: 0},
				UpdatedAt: ts,
			},
		},
		Portals: []state.Portal{
			{
				ID:        "portal-nil",
				Position:  state.Position{Latitude: 0, Longitude: 0},
				Faction:   "NEUTRAL",
				UpdatedAt: ts,
			},
		},
	}

	models, err := snapshotToModels(snapshot)
	if err != nil {
		t.Fatalf("snapshotToModels: %v", err)
	}

	roundTrip, err := modelsToSnapshot(models)
	if err != nil {
		t.Fatalf("modelsToSnapshot: %v", err)
	}

	if len(roundTrip.Players) != 1 || len(roundTrip.Portals) != 1 {
		t.Fatalf("unexpected counts: players=%d portals=%d", len(roundTrip.Players), len(roundTrip.Portals))
	}
	player := roundTrip.Players[0]
	if player.Inventory != nil {
		t.Fatalf("expected nil inventory, got %v", player.Inventory)
	}
	if player.HackCooldowns != nil {
		t.Fatalf("expected nil hack cooldowns, got %v", player.HackCooldowns)
	}
	if player.View != nil || player.Target != nil || player.ViewTiles != nil {
		t.Fatalf("expected nil view/target/tiles")
	}
	portal := roundTrip.Portals[0]
	if portal.Resonators != nil {
		t.Fatalf("expected nil resonators, got %v", portal.Resonators)
	}
	if portal.Mods != nil {
		t.Fatalf("expected nil mods, got %v", portal.Mods)
	}
}

func TestSnapshotMapperExtractsInventoryRows(t *testing.T) {
	snapshot := state.Snapshot{
		Players: []state.Player{
			{
				ID:        "p1",
				Username:  "u1",
				Inventory: map[string]int{"XMP_L1": 2, "RESO_L1": 4},
			},
		},
	}

	models, err := snapshotToModels(snapshot)
	if err != nil {
		t.Fatalf("snapshotToModels: %v", err)
	}
	if len(models.players) != 1 {
		t.Fatalf("expected 1 player model, got %d", len(models.players))
	}
	if len(models.inventories) != 2 {
		t.Fatalf("expected 2 inventory rows, got %d", len(models.inventories))
	}
	if models.inventories[0].PlayerID != "p1" || models.inventories[1].PlayerID != "p1" {
		t.Fatalf("expected inventory rows to belong to player p1")
	}
}
