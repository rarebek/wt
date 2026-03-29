package main

import "testing"

func TestMatchAddRemovePlayer(t *testing.T) {
	m := NewMatch("test", MatchConfig{MaxPlayers: 4})

	if !m.AddPlayer("p1", "Alice", 0) {
		t.Error("should add player")
	}
	if !m.AddPlayer("p2", "Bob", 1) {
		t.Error("should add player")
	}
	if m.PlayerCount() != 2 {
		t.Errorf("expected 2, got %d", m.PlayerCount())
	}

	m.RemovePlayer("p1")
	if m.PlayerCount() != 1 {
		t.Errorf("expected 1, got %d", m.PlayerCount())
	}
}

func TestMatchFull(t *testing.T) {
	m := NewMatch("test", MatchConfig{MaxPlayers: 2})

	m.AddPlayer("p1", "A", 0)
	m.AddPlayer("p2", "B", 1)
	if m.AddPlayer("p3", "C", 0) {
		t.Error("should not add to full match")
	}
}

func TestMatchDamage(t *testing.T) {
	m := NewMatch("test", MatchConfig{MaxPlayers: 4})
	m.AddPlayer("attacker", "Attacker", 0)
	m.AddPlayer("target", "Target", 1)

	event := m.ApplyDamage("attacker", "target", 30)
	if event == nil {
		t.Fatal("expected damage event")
	}
	if event.Type != "damage" {
		t.Errorf("expected 'damage', got %q", event.Type)
	}

	m.mu.RLock()
	hp := m.players["target"].Health
	m.mu.RUnlock()
	if hp != 70 {
		t.Errorf("expected 70 HP, got %d", hp)
	}
}

func TestMatchKill(t *testing.T) {
	m := NewMatch("test", MatchConfig{MaxPlayers: 4})
	m.AddPlayer("killer", "Killer", 0)
	m.AddPlayer("victim", "Victim", 1)

	// 4 hits of 25 = 100 damage = kill
	for range 3 {
		m.ApplyDamage("killer", "victim", 25)
	}
	event := m.ApplyDamage("killer", "victim", 25)

	if event == nil || event.Type != "kill" {
		t.Error("expected kill event")
	}

	// Victim should respawn with full HP
	m.mu.RLock()
	hp := m.players["victim"].Health
	m.mu.RUnlock()
	if hp != 100 {
		t.Errorf("expected 100 HP after respawn, got %d", hp)
	}
}

func TestMatchWorldState(t *testing.T) {
	m := NewMatch("test", MatchConfig{MaxPlayers: 4})
	m.AddPlayer("p1", "A", 0)
	m.AddPlayer("p2", "B", 1)

	states := m.GetWorldState()
	if len(states) != 2 {
		t.Errorf("expected 2 players in world state, got %d", len(states))
	}
}

func TestMatchmaker(t *testing.T) {
	mm := NewMatchmaker()
	m := mm.CreateMatch(MatchConfig{MaxPlayers: 8, MapName: "test"})

	if m.id == "" {
		t.Error("expected non-empty match ID")
	}

	got := mm.GetMatch(m.id)
	if got != m {
		t.Error("expected same match")
	}

	if mm.GetMatch("nonexistent") != nil {
		t.Error("expected nil for nonexistent match")
	}
}

func TestLeaderboard(t *testing.T) {
	lb := NewLeaderboard()
	lb.Update("Alice", 100, 10, 2)
	lb.Update("Bob", 80, 8, 3)
	lb.Update("Charlie", 120, 12, 1)

	top := lb.Top(2)
	if len(top) != 2 {
		t.Fatalf("expected 2, got %d", len(top))
	}
	if top[0].Name != "Charlie" {
		t.Errorf("expected Charlie first, got %s", top[0].Name)
	}
}

func TestClampPosition(t *testing.T) {
	pos := Vec2{X: -10, Y: 700}
	clampPosition(&pos)
	if pos.X != 0 || pos.Y != MapHeight {
		t.Errorf("expected clamped, got %+v", pos)
	}
}

func TestDistance(t *testing.T) {
	d := distance(Vec2{0, 0}, Vec2{3, 4})
	if d < 4.9 || d > 5.1 {
		t.Errorf("expected ~5, got %f", d)
	}
}

func TestMatchQueue(t *testing.T) {
	mm := NewMatchmaker()
	mm.JoinQueue("s1", "Alice")
	mm.JoinQueue("s2", "Bob")

	if mm.QueueSize() != 2 {
		t.Errorf("expected 2, got %d", mm.QueueSize())
	}
}
