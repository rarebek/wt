// Example: WebTransport game server.
//
// Demonstrates the key WebTransport advantage over WebSocket:
// - Unreliable datagrams for position updates (60 Hz, fire-and-forget)
// - Reliable streams for game events (chat, inventory, abilities)
// - No head-of-line blocking between channels
package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"sync"
	"time"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

// --- Game types ---

type Vec2 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type PlayerState struct {
	ID       string  `json:"id"`
	Position Vec2    `json:"pos"`
	Velocity Vec2    `json:"vel"`
	Health   int     `json:"hp"`
	Tick     uint64  `json:"tick"`
}

type GameEvent struct {
	Type    string `json:"type"` // "chat", "ability", "damage", "join", "leave"
	Player  string `json:"player"`
	Payload string `json:"payload"`
	Tick    uint64 `json:"tick"`
}

// --- Game state ---

type GameRoom struct {
	mu      sync.RWMutex
	players map[string]*PlayerState
	room    *wt.Room
	tick    uint64
}

func NewGameRoom(room *wt.Room) *GameRoom {
	return &GameRoom{
		players: make(map[string]*PlayerState),
		room:    room,
	}
}

func (g *GameRoom) AddPlayer(id string) {
	g.mu.Lock()
	g.players[id] = &PlayerState{
		ID:     id,
		Health: 100,
	}
	g.mu.Unlock()
}

func (g *GameRoom) RemovePlayer(id string) {
	g.mu.Lock()
	delete(g.players, id)
	g.mu.Unlock()
}

func (g *GameRoom) UpdatePosition(id string, pos, vel Vec2) {
	g.mu.Lock()
	if p, ok := g.players[id]; ok {
		p.Position = pos
		p.Velocity = vel
	}
	g.mu.Unlock()
}

func (g *GameRoom) GetAllStates() []PlayerState {
	g.mu.RLock()
	defer g.mu.RUnlock()
	states := make([]PlayerState, 0, len(g.players))
	g.tick++
	for _, p := range g.players {
		p.Tick = g.tick
		states = append(states, *p)
	}
	return states
}

// --- Main ---

var rooms = wt.NewRoomManager()
var games = sync.Map{} // roomName -> *GameRoom

func main() {
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
	)

	metrics := middleware.NewMetrics()

	log.Printf("Certificate hash: %s", server.CertHash())

	server.Use(middleware.DefaultLogger())
	server.Use(middleware.Recover(nil))
	server.Use(metrics.Middleware())

	server.Handle("/game/{id}", handleGame)

	// Log metrics every 5 seconds
	go func() {
		for {
			time.Sleep(5 * time.Second)
			snap := metrics.Snapshot()
			slog.Info("server metrics",
				"active_sessions", snap.ActiveSessions,
				"total_sessions", snap.TotalSessions,
			)
		}
	}()

	log.Printf("Game server listening on %s", server.Addr())
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func handleGame(c *wt.Context) {
	gameID := c.Param("id")
	playerID := c.ID()

	// Get or create game room
	room := rooms.GetOrCreate(gameID)
	room.Join(c)
	defer room.Leave(c)

	gameVal, _ := games.LoadOrStore(gameID, NewGameRoom(room))
	game := gameVal.(*GameRoom)
	game.AddPlayer(playerID)
	defer game.RemovePlayer(playerID)

	slog.Info("player joined",
		"game", gameID,
		"player", playerID,
		"total_players", room.Count(),
	)

	// Broadcast join event via reliable stream to all other players
	broadcastEvent(room, GameEvent{
		Type:   "join",
		Player: playerID,
	}, playerID)

	// Goroutine 1: Receive position datagrams from this player
	// (unreliable — if a position update is lost, who cares, next one comes in 16ms)
	go func() {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			var pos Vec2
			if err := json.Unmarshal(data, &pos); err != nil {
				continue
			}
			game.UpdatePosition(playerID, pos, Vec2{})
		}
	}()

	// Goroutine 2: Send world state to this player via datagrams at 20 Hz
	// (server-authoritative position broadcast)
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond) // 20 Hz
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				states := game.GetAllStates()
				data, _ := json.Marshal(states)
				_ = c.SendDatagram(data) // unreliable, no head-of-line blocking
			case <-c.Context().Done():
				return
			}
		}
	}()

	// Main loop: Accept reliable streams for game events (chat, abilities, etc.)
	for {
		stream, err := c.AcceptStream()
		if err != nil {
			broadcastEvent(room, GameEvent{
				Type:   "leave",
				Player: playerID,
			}, playerID)
			return
		}
		go handleGameEvent(stream, room, game, playerID)
	}
}

func handleGameEvent(s *wt.Stream, room *wt.Room, _ *GameRoom, playerID string) {
	defer s.Close()

	msg, err := s.ReadMessage()
	if err != nil {
		return
	}

	var event GameEvent
	if err := json.Unmarshal(msg, &event); err != nil {
		return
	}
	event.Player = playerID

	slog.Info("game event",
		"type", event.Type,
		"player", playerID,
	)

	// Broadcast event to all room members via reliable streams
	broadcastEvent(room, event, playerID)
}

func broadcastEvent(room *wt.Room, event GameEvent, excludeID string) {
	data, _ := json.Marshal(event)
	for _, member := range room.Members() {
		if member.ID() == excludeID {
			continue
		}
		go func(m *wt.Context) {
			s, err := m.OpenStream()
			if err != nil {
				return
			}
			_ = s.WriteMessage(data)
			s.Close()
		}(member)
	}
}
