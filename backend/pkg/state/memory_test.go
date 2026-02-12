package state

import (
	"testing"
	"time"
)

func TestInMemoryPlayerRepoReturnsDetachedCopies(t *testing.T) {
	repo := NewInMemoryPlayerRepo()
	now := time.Now()
	repo.Upsert(Player{
		ID:            "p1",
		Inventory:     map[string]int{"RESO_L1": 1},
		HackCooldowns: map[string]time.Time{"portal-1": now},
		ViewTiles:     []string{"10/20/30"},
		Target:        &Position{Latitude: 1, Longitude: 2},
		View:          &Bounds{MinLat: 1, MaxLat: 2, MinLon: 3, MaxLon: 4},
	})

	player, ok := repo.Get("p1")
	if !ok {
		t.Fatalf("expected player exists")
	}
	player.Inventory["RESO_L1"] = 99
	player.HackCooldowns["portal-1"] = now.Add(time.Hour)
	player.ViewTiles[0] = "changed"
	player.Target.Latitude = 9
	player.View.MaxLat = 99

	again, ok := repo.Get("p1")
	if !ok {
		t.Fatalf("expected player exists")
	}
	if again.Inventory["RESO_L1"] != 1 {
		t.Fatalf("expected stored inventory unchanged, got %d", again.Inventory["RESO_L1"])
	}
	if !again.HackCooldowns["portal-1"].Equal(now) {
		t.Fatalf("expected stored cooldown unchanged")
	}
	if again.ViewTiles[0] != "10/20/30" {
		t.Fatalf("expected stored view tiles unchanged")
	}
	if again.Target == nil || again.Target.Latitude != 1 {
		t.Fatalf("expected stored target unchanged")
	}
	if again.View == nil || again.View.MaxLat != 2 {
		t.Fatalf("expected stored view unchanged")
	}

	list := repo.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 player in list, got %d", len(list))
	}
	list[0].Inventory["RESO_L1"] = 7
	last, _ := repo.Get("p1")
	if last.Inventory["RESO_L1"] != 1 {
		t.Fatalf("expected list mutation not affect store")
	}
}

func TestInMemoryPortalRepoReturnsDetachedCopies(t *testing.T) {
	repo := NewInMemoryPortalRepo()
	repo.Upsert(Portal{
		ID:    "portal-1",
		Title: "Portal 1",
		Resonators: map[int]ResonatorSlot{
			1: {Slot: 1, Level: 1, Energy: 1000},
		},
		Mods: map[int]ModSlot{
			1: {Slot: 1, ModType: "SHIELD", Rarity: "COMMON"},
		},
	})

	portal, ok := repo.Get("portal-1")
	if !ok {
		t.Fatalf("expected portal exists")
	}
	portal.Resonators[1] = ResonatorSlot{Slot: 1, Level: 8, Energy: 6000}
	portal.Mods[1] = ModSlot{Slot: 1, ModType: "LINK_AMP", Rarity: "RARE"}

	again, ok := repo.Get("portal-1")
	if !ok {
		t.Fatalf("expected portal exists")
	}
	if again.Resonators[1].Level != 1 {
		t.Fatalf("expected stored resonator unchanged, got %d", again.Resonators[1].Level)
	}
	if again.Mods[1].ModType != "SHIELD" {
		t.Fatalf("expected stored mod unchanged, got %s", again.Mods[1].ModType)
	}
}
