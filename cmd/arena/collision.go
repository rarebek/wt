package main

// Collision detection for the arena game.

// CircleCollision checks if two circular entities overlap.
func CircleCollision(a, b Vec2, radiusA, radiusB float32) bool {
	return distance(a, b) < radiusA+radiusB
}

// AABBCollision checks axis-aligned bounding box collision.
type AABB struct {
	X, Y, W, H float32
}

func (a AABB) Overlaps(b AABB) bool {
	return a.X < b.X+b.W && a.X+a.W > b.X &&
		a.Y < b.Y+b.H && a.Y+a.H > b.Y
}

// PointInAABB checks if a point is inside a bounding box.
func PointInAABB(p Vec2, box AABB) bool {
	return p.X >= box.X && p.X <= box.X+box.W &&
		p.Y >= box.Y && p.Y <= box.Y+box.H
}

// FindPlayersInRadius returns player IDs within radius of a point.
func (m *Match) FindPlayersInRadius(center Vec2, radius float32) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []string
	for id, p := range m.players {
		if distance(center, p.Position) <= radius {
			result = append(result, id)
		}
	}
	return result
}

// FindClosestEnemy finds the nearest enemy player (different team).
func (m *Match) FindClosestEnemy(playerID string) (string, float32) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	player, ok := m.players[playerID]
	if !ok {
		return "", 0
	}

	var closestID string
	var closestDist float32 = 999999

	for id, p := range m.players {
		if id == playerID || p.Team == player.Team {
			continue
		}
		d := distance(player.Position, p.Position)
		if d < closestDist {
			closestDist = d
			closestID = id
		}
	}

	return closestID, closestDist
}
