package gameplay

import "testing"

func TestCanAddKeyRespectsLimit(t *testing.T) {
	inv := map[string]int{
		ItemIDKey("portal-1"): 2,
	}
	if CanAddKey(inv, "portal-1", 1, 1) {
		t.Fatalf("expected key limit to block adding")
	}
}

func TestCapacityForLevel(t *testing.T) {
	general, keys := CapacityForLevel(1)
	if general != 500 || keys != 500 {
		t.Fatalf("expected L1 capacity 500/500, got %d/%d", general, keys)
	}
	general, keys = CapacityForLevel(9)
	if general != 950 || keys != 950 {
		t.Fatalf("expected L9 capacity 950/950, got %d/%d", general, keys)
	}
}
