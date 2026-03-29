package main

import "testing"

func TestCircleCollision(t *testing.T) {
	if !CircleCollision(Vec2{0, 0}, Vec2{5, 0}, 3, 3) {
		t.Error("circles should overlap (distance 5 < radius 6)")
	}
	if CircleCollision(Vec2{0, 0}, Vec2{10, 0}, 3, 3) {
		t.Error("circles should not overlap (distance 10 > radius 6)")
	}
}

func TestAABBOverlaps(t *testing.T) {
	a := AABB{0, 0, 10, 10}
	b := AABB{5, 5, 10, 10}
	c := AABB{20, 20, 10, 10}

	if !a.Overlaps(b) {
		t.Error("a and b should overlap")
	}
	if a.Overlaps(c) {
		t.Error("a and c should not overlap")
	}
}

func TestPointInAABB(t *testing.T) {
	box := AABB{10, 10, 20, 20}

	if !PointInAABB(Vec2{15, 15}, box) {
		t.Error("point inside should return true")
	}
	if PointInAABB(Vec2{5, 5}, box) {
		t.Error("point outside should return false")
	}
}

func TestFindPlayersInRadius(t *testing.T) {
	m := NewMatch("test", MatchConfig{MaxPlayers: 8})
	m.AddPlayer("p1", "A", 0)
	m.AddPlayer("p2", "B", 1)

	m.mu.Lock()
	m.players["p1"].Position = Vec2{100, 100}
	m.players["p2"].Position = Vec2{105, 100}
	m.mu.Unlock()

	nearby := m.FindPlayersInRadius(Vec2{100, 100}, 10)
	if len(nearby) != 2 {
		t.Errorf("expected 2 nearby, got %d", len(nearby))
	}

	far := m.FindPlayersInRadius(Vec2{500, 500}, 10)
	if len(far) != 0 {
		t.Errorf("expected 0 far, got %d", len(far))
	}
}

func TestFindClosestEnemy(t *testing.T) {
	m := NewMatch("test", MatchConfig{MaxPlayers: 8})
	m.AddPlayer("p1", "A", 0)
	m.AddPlayer("p2", "B", 1)
	m.AddPlayer("p3", "C", 1)

	m.mu.Lock()
	m.players["p1"].Position = Vec2{100, 100}
	m.players["p2"].Position = Vec2{110, 100} // 10 away
	m.players["p3"].Position = Vec2{200, 100} // 100 away
	m.mu.Unlock()

	closest, dist := m.FindClosestEnemy("p1")
	if closest != "p2" {
		t.Errorf("expected p2, got %s", closest)
	}
	if dist > 11 {
		t.Errorf("expected ~10, got %f", dist)
	}
}
