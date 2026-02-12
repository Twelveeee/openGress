package database

import "testing"

func TestDiffModelIDsOnlyChangedAndDeleted(t *testing.T) {
	prev := map[string]string{
		"a": "h1",
		"b": "h2",
		"d": "h4",
	}
	next := map[string]string{
		"a": "h1",
		"b": "h3",
		"c": "h5",
	}

	upserts, deletes := diffModelIDs(prev, next)
	if len(upserts) != 2 || upserts[0] != "b" || upserts[1] != "c" {
		t.Fatalf("unexpected upserts: %v", upserts)
	}
	if len(deletes) != 1 || deletes[0] != "d" {
		t.Fatalf("unexpected deletes: %v", deletes)
	}
}

func TestChunkIDs(t *testing.T) {
	ids := []string{"1", "2", "3", "4", "5"}
	chunks := chunkIDs(ids, 2)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 2 || chunks[0][0] != "1" || chunks[0][1] != "2" {
		t.Fatalf("unexpected first chunk: %v", chunks[0])
	}
	if len(chunks[1]) != 2 || chunks[1][0] != "3" || chunks[1][1] != "4" {
		t.Fatalf("unexpected second chunk: %v", chunks[1])
	}
	if len(chunks[2]) != 1 || chunks[2][0] != "5" {
		t.Fatalf("unexpected third chunk: %v", chunks[2])
	}
}

func TestBuildModelStateDetectsChangedRowHash(t *testing.T) {
	origin := modelSnapshot{
		players: []PlayerModel{
			{ID: "p1", Username: "u1"},
		},
	}
	changed := modelSnapshot{
		players: []PlayerModel{
			{ID: "p1", Username: "u2"},
		},
	}

	originState, _, err := buildModelState(origin)
	if err != nil {
		t.Fatalf("buildModelState origin failed: %v", err)
	}
	changedState, _, err := buildModelState(changed)
	if err != nil {
		t.Fatalf("buildModelState changed failed: %v", err)
	}
	if originState.players["p1"] == changedState.players["p1"] {
		t.Fatalf("expected hash to change for modified row")
	}
}

func TestBuildModelStateDetectsChangedInventoryRowHash(t *testing.T) {
	origin := modelSnapshot{
		inventories: []PlayerInventoryModel{
			{PlayerID: "p1", ItemID: "XMP_L1", Amount: 1},
		},
	}
	changed := modelSnapshot{
		inventories: []PlayerInventoryModel{
			{PlayerID: "p1", ItemID: "XMP_L1", Amount: 2},
		},
	}

	originState, _, err := buildModelState(origin)
	if err != nil {
		t.Fatalf("buildModelState origin failed: %v", err)
	}
	changedState, _, err := buildModelState(changed)
	if err != nil {
		t.Fatalf("buildModelState changed failed: %v", err)
	}

	key := inventoryModelKey("p1", "XMP_L1")
	if originState.inventories[key] == changedState.inventories[key] {
		t.Fatalf("expected inventory hash to change for modified row")
	}
}
