package state

import "testing"

func TestSnapshotClonesAdminState(t *testing.T) {
	gameState := NewGameState()
	gameState.Admin.Banned["p1"] = "reason-1"

	snapshot := gameState.Snapshot()
	if snapshot.Admin == nil {
		t.Fatalf("expected admin snapshot not nil")
	}

	snapshot.Admin.Banned["p1"] = "changed"
	if gameState.Admin.Banned["p1"] != "reason-1" {
		t.Fatalf("expected snapshot mutation not affect game state")
	}
}
