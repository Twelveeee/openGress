package gameplay

import "testing"

func TestLevelForAP(t *testing.T) {
	if got := LevelForAP(0); got != 1 {
		t.Fatalf("expected L1 for AP 0, got L%d", got)
	}
	if got := LevelForAP(2500); got != 2 {
		t.Fatalf("expected L2 for AP 2500, got L%d", got)
	}
	if got := LevelForAP(1200000); got != 8 {
		t.Fatalf("expected L8 for AP 1200000, got L%d", got)
	}
	if got := LevelForAP(40000000); got != 16 {
		t.Fatalf("expected L16 for AP 40000000, got L%d", got)
	}
}

func TestMaxXMForLevel(t *testing.T) {
	if got := MaxXMForLevel(1); got != 3000 {
		t.Fatalf("expected L1 max xm 3000, got %d", got)
	}
	if got := MaxXMForLevel(8); got != 10000 {
		t.Fatalf("expected L8 max xm 10000, got %d", got)
	}
	if got := MaxXMForLevel(16); got != 22000 {
		t.Fatalf("expected L16 max xm 22000, got %d", got)
	}
}
