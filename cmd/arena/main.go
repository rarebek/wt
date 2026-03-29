// Arena — A production-grade multiplayer game server built on wt.
//
// Demonstrates every WebTransport advantage over WebSocket:
//
// 1. DATAGRAMS for player positions (60Hz, unreliable, no HOL blocking)
//    - Lost position? Who cares, next one arrives in 16ms
//    - WebSocket can't do this — every packet is guaranteed delivery
//
// 2. MULTIPLEXED STREAMS for game events (reliable, ordered, independent)
//    - Stream 1: Chat messages (reliable)
//    - Stream 2: Inventory/ability events (reliable)
//    - Stream 3: Match control (join/leave/ready)
//    - Each independent — a big chat message doesn't freeze game events
//
// 3. CONNECTION MIGRATION for mobile players
//    - Switch WiFi → cellular without disconnecting
//    - WebSocket drops the connection
//
// 4. ROOMS with presence for matchmaking
//    - Lobby, match rooms, spectator rooms
//    - Real-time presence (who's online, typing, ready)
//
// Architecture:
//   /lobby           — matchmaking, chat, presence
//   /match/{id}      — actual game (datagrams + streams)
//   /spectate/{id}   — watch a match (receive-only datagrams)
//   /leaderboard     — stats push via datagrams
//   /rpc             — account/settings via JSON-RPC

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"math"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rarebek/wt"
	"github.com/rarebek/wt/middleware"
)

// ═══════════════════════════════════════════════════
// GAME TYPES
// ═══════════════════════════════════════════════════

type Vec2 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
}

type PlayerState struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Position Vec2    `json:"pos"`
	Velocity Vec2    `json:"vel"`
	Rotation float32 `json:"rot"`
	Health   int     `json:"hp"`
	Score    int     `json:"score"`
	Tick     uint64  `json:"tick"`
	Team     int     `json:"team"` // 0=red, 1=blue
}

type GameEvent struct {
	Type    string `json:"type"` // chat, ability, damage, kill, join, leave, ready
	Player  string `json:"player"`
	Target  string `json:"target,omitempty"`
	Value   int    `json:"value,omitempty"`
	Message string `json:"message,omitempty"`
	Tick    uint64 `json:"tick"`
}

type MatchConfig struct {
	MaxPlayers int           `json:"max_players"`
	MapName    string        `json:"map_name"`
	Duration   time.Duration `json:"duration"`
	TickRate   int           `json:"tick_rate"` // Hz
}

type MatchState int

const (
	MatchWaiting  MatchState = iota // waiting for players
	MatchStarting                   // countdown
	MatchRunning                    // in progress
	MatchEnded                      // game over
)

// ═══════════════════════════════════════════════════
// MATCH ENGINE
// ═══════════════════════════════════════════════════

type Match struct {
	mu       sync.RWMutex
	id       string
	config   MatchConfig
	state    MatchState
	players  map[string]*PlayerState
	room     *wt.Room
	tick     atomic.Uint64
	started  time.Time
	events   *wt.RingBuffer[GameEvent]
}

func NewMatch(id string, config MatchConfig) *Match {
	return &Match{
		id:      id,
		config:  config,
		state:   MatchWaiting,
		players: make(map[string]*PlayerState),
		events:  wt.NewRingBuffer[GameEvent](100),
	}
}

func (m *Match) AddPlayer(id, name string, team int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.players) >= m.config.MaxPlayers {
		return false
	}

	// Random spawn position
	spawnX := float32(rand.IntN(800))
	spawnY := float32(rand.IntN(600))

	m.players[id] = &PlayerState{
		ID:       id,
		Name:     name,
		Position: Vec2{X: spawnX, Y: spawnY},
		Health:   100,
		Team:     team,
	}
	return true
}

func (m *Match) RemovePlayer(id string) {
	m.mu.Lock()
	delete(m.players, id)
	m.mu.Unlock()
}

func (m *Match) UpdatePosition(id string, pos, vel Vec2, rot float32) {
	m.mu.RLock()
	p, ok := m.players[id]
	m.mu.RUnlock()
	if ok {
		p.Position = pos
		p.Velocity = vel
		p.Rotation = rot
	}
}

func (m *Match) ApplyDamage(attackerID, targetID string, damage int) *GameEvent {
	m.mu.Lock()
	defer m.mu.Unlock()

	target, ok := m.players[targetID]
	if !ok {
		return nil
	}

	target.Health -= damage
	if target.Health < 0 {
		target.Health = 0
	}

	event := GameEvent{
		Type:   "damage",
		Player: attackerID,
		Target: targetID,
		Value:  damage,
		Tick:   m.tick.Load(),
	}
	m.events.Push(event)

	if target.Health == 0 {
		killEvent := GameEvent{
			Type:   "kill",
			Player: attackerID,
			Target: targetID,
			Tick:   m.tick.Load(),
		}
		m.events.Push(killEvent)

		// Respawn
		target.Health = 100
		target.Position = Vec2{X: float32(rand.IntN(800)), Y: float32(rand.IntN(600))}

		if attacker, ok := m.players[attackerID]; ok {
			attacker.Score += 10
		}

		return &killEvent
	}

	return &event
}

func (m *Match) GetWorldState() []PlayerState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	tick := m.tick.Add(1)
	states := make([]PlayerState, 0, len(m.players))
	for _, p := range m.players {
		p.Tick = tick
		states = append(states, *p)
	}
	return states
}

func (m *Match) PlayerCount() int {
	m.mu.RLock()
	n := len(m.players)
	m.mu.RUnlock()
	return n
}

func (m *Match) IsRunning() bool {
	return m.state == MatchRunning
}

// ═══════════════════════════════════════════════════
// MATCHMAKER
// ═══════════════════════════════════════════════════

type Matchmaker struct {
	mu      sync.Mutex
	matches map[string]*Match
	queue   []*queueEntry
}

type queueEntry struct {
	sessionID string
	name      string
	joinedAt  time.Time
}

func NewMatchmaker() *Matchmaker {
	return &Matchmaker{
		matches: make(map[string]*Match),
	}
}

func (mm *Matchmaker) CreateMatch(config MatchConfig) *Match {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	id := fmt.Sprintf("match-%d", len(mm.matches)+1)
	m := NewMatch(id, config)
	mm.matches[id] = m
	return m
}

func (mm *Matchmaker) GetMatch(id string) *Match {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	return mm.matches[id]
}

func (mm *Matchmaker) ActiveMatches() int {
	mm.mu.Lock()
	defer mm.mu.Unlock()
	count := 0
	for _, m := range mm.matches {
		if m.IsRunning() {
			count++
		}
	}
	return count
}

func (mm *Matchmaker) JoinQueue(sessionID, name string) {
	mm.mu.Lock()
	mm.queue = append(mm.queue, &queueEntry{
		sessionID: sessionID,
		name:      name,
		joinedAt:  time.Now(),
	})
	mm.mu.Unlock()
}

func (mm *Matchmaker) QueueSize() int {
	mm.mu.Lock()
	n := len(mm.queue)
	mm.mu.Unlock()
	return n
}

// ═══════════════════════════════════════════════════
// LEADERBOARD
// ═══════════════════════════════════════════════════

type LeaderboardEntry struct {
	Name   string `json:"name"`
	Score  int    `json:"score"`
	Kills  int    `json:"kills"`
	Deaths int    `json:"deaths"`
	KD     float64 `json:"kd"`
}

type Leaderboard struct {
	mu      sync.RWMutex
	entries map[string]*LeaderboardEntry
}

func NewLeaderboard() *Leaderboard {
	return &Leaderboard{entries: make(map[string]*LeaderboardEntry)}
}

func (lb *Leaderboard) Update(name string, score, kills, deaths int) {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	e, ok := lb.entries[name]
	if !ok {
		e = &LeaderboardEntry{Name: name}
		lb.entries[name] = e
	}
	e.Score += score
	e.Kills += kills
	e.Deaths += deaths
	if e.Deaths > 0 {
		e.KD = float64(e.Kills) / float64(e.Deaths)
	} else {
		e.KD = float64(e.Kills)
	}
}

func (lb *Leaderboard) Top(n int) []LeaderboardEntry {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	all := make([]LeaderboardEntry, 0, len(lb.entries))
	for _, e := range lb.entries {
		all = append(all, *e)
	}

	// Simple sort by score (descending)
	for i := range len(all) {
		for j := i + 1; j < len(all); j++ {
			if all[j].Score > all[i].Score {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// ═══════════════════════════════════════════════════
// PHYSICS (simplified 2D)
// ═══════════════════════════════════════════════════

const (
	MapWidth  = 800
	MapHeight = 600
	MaxSpeed  = 200.0 // units per second
)

func clampPosition(pos *Vec2) {
	if pos.X < 0 { pos.X = 0 }
	if pos.Y < 0 { pos.Y = 0 }
	if pos.X > MapWidth { pos.X = MapWidth }
	if pos.Y > MapHeight { pos.Y = MapHeight }
}

func distance(a, b Vec2) float32 {
	dx := a.X - b.X
	dy := a.Y - b.Y
	return float32(math.Sqrt(float64(dx*dx + dy*dy)))
}

// ═══════════════════════════════════════════════════
// STREAM MUX TYPES
// ═══════════════════════════════════════════════════

const (
	StreamChat     uint16 = 1
	StreamAbility  uint16 = 2
	StreamControl  uint16 = 3
)

// ═══════════════════════════════════════════════════
// GLOBALS
// ═══════════════════════════════════════════════════

var (
	rooms       = wt.NewRoomManager()
	presence    = wt.NewPresenceTracker()
	matchmaker  = NewMatchmaker()
	leaderboard = NewLeaderboard()
	pubsub      = wt.NewPubSub()
	tags        = wt.NewTags()
)

// ═══════════════════════════════════════════════════
// HANDLERS
// ═══════════════════════════════════════════════════

func handleLobby(c *wt.Context) {
	room := rooms.GetOrCreate("lobby")
	room.Join(c)
	defer room.Leave(c)

	playerName := c.GetString("name")
	if playerName == "" {
		playerName = "Player-" + c.ID()[:6]
	}

	presence.Join("lobby", c)
	defer presence.Leave("lobby", c)
	tags.Tag(c.ID(), "lobby")
	defer tags.Untag(c.ID(), "lobby")

	slog.Info("player joined lobby", "name", playerName, "id", c.ID())

	// Broadcast presence update
	broadcastPresence(room, "lobby")

	// Send lobby status via datagrams every 2 seconds
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				status := map[string]any{
					"type":           "lobby_status",
					"players_online": presence.Count("lobby"),
					"queue_size":     matchmaker.QueueSize(),
					"active_matches": matchmaker.ActiveMatches(),
				}
				data, _ := json.Marshal(status)
				_ = c.SendDatagram(data)
			case <-c.Context().Done():
				return
			}
		}
	}()

	// Handle lobby streams (chat + match control)
	mux := wt.NewStreamMux()

	mux.Handle(StreamChat, func(s *wt.Stream, ctx *wt.Context) {
		defer s.Close()
		for {
			msg, err := s.ReadMessage()
			if err != nil {
				return
			}
			var chatMsg GameEvent
			json.Unmarshal(msg, &chatMsg)
			chatMsg.Type = "chat"
			chatMsg.Player = playerName

			encoded, _ := json.Marshal(chatMsg)
			room.BroadcastExcept(encoded, c.ID())
		}
	})

	mux.Handle(StreamControl, func(s *wt.Stream, ctx *wt.Context) {
		defer s.Close()
		msg, err := s.ReadMessage()
		if err != nil {
			return
		}

		var cmd struct {
			Action string `json:"action"` // queue, cancel_queue, create_match
		}
		json.Unmarshal(msg, &cmd)

		switch cmd.Action {
		case "queue":
			matchmaker.JoinQueue(c.ID(), playerName)
			s.WriteJSON(map[string]string{"status": "queued"})
		case "create_match":
			match := matchmaker.CreateMatch(MatchConfig{
				MaxPlayers: 8,
				MapName:    "arena_01",
				Duration:   5 * time.Minute,
				TickRate:   20,
			})
			s.WriteJSON(map[string]string{
				"status":   "created",
				"match_id": match.id,
			})
		}
	})

	mux.Serve(c)
}

func handleMatch(c *wt.Context) {
	matchID := c.Param("id")
	match := matchmaker.GetMatch(matchID)
	if match == nil {
		c.CloseWithError(404, "match not found")
		return
	}

	playerName := c.GetString("name")
	if playerName == "" {
		playerName = "Player-" + c.ID()[:6]
	}

	team := 0
	if match.PlayerCount()%2 == 1 {
		team = 1
	}

	if !match.AddPlayer(c.ID(), playerName, team) {
		c.CloseWithError(503, "match full")
		return
	}
	defer match.RemovePlayer(c.ID())

	if match.room == nil {
		match.room = rooms.GetOrCreate("match-" + matchID)
	}
	match.room.Join(c)
	defer match.room.Leave(c)

	presence.Join("match-"+matchID, c)
	defer presence.Leave("match-"+matchID, c)
	tags.Tag(c.ID(), "in-match")
	defer tags.Untag(c.ID(), "in-match")

	slog.Info("player joined match",
		"match", matchID,
		"player", playerName,
		"team", team,
		"players", match.PlayerCount(),
	)

	// Broadcast join event
	joinEvent := GameEvent{
		Type:    "join",
		Player:  playerName,
		Message: fmt.Sprintf("%s joined team %d", playerName, team),
	}
	eventData, _ := json.Marshal(joinEvent)
	match.room.BroadcastExcept(eventData, c.ID())

	// ── DATAGRAM LOOP: receive player input (unreliable, 60Hz) ──
	// This is THE WebTransport advantage. Position updates at 60Hz
	// as fire-and-forget datagrams. Lost packet? Next one in 16ms.
	// WebSocket would guarantee delivery of every position, causing
	// stalls when any packet is lost.
	go func() {
		for {
			data, err := c.ReceiveDatagram()
			if err != nil {
				return
			}
			var input struct {
				Pos Vec2    `json:"pos"`
				Vel Vec2    `json:"vel"`
				Rot float32 `json:"rot"`
			}
			if json.Unmarshal(data, &input) == nil {
				clampPosition(&input.Pos)
				match.UpdatePosition(c.ID(), input.Pos, input.Vel, input.Rot)
			}
		}
	}()

	// ── WORLD STATE PUSH: send game state to all players (20Hz datagrams) ──
	// Every 50ms, serialize all player positions and send as a datagram.
	// If one datagram is lost, the next one has the full world state.
	// With WebSocket, a lost TCP segment would freeze ALL data until retransmit.
	go func() {
		tickRate := match.config.TickRate
		if tickRate <= 0 {
			tickRate = 20
		}
		ticker := time.NewTicker(time.Second / time.Duration(tickRate))
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				states := match.GetWorldState()
				data, _ := json.Marshal(map[string]any{
					"type":    "world_state",
					"players": states,
					"tick":    match.tick.Load(),
				})
				_ = c.SendDatagram(data)
			case <-c.Context().Done():
				return
			}
		}
	}()

	// ── STREAM MUX: reliable game events (chat, abilities, damage) ──
	// Each event type gets its own independent stream.
	// A large chat message doesn't block a damage event.
	mux := wt.NewStreamMux()

	// Chat stream — reliable, ordered (within this stream only)
	mux.Handle(StreamChat, func(s *wt.Stream, _ *wt.Context) {
		defer s.Close()
		for {
			msg, err := s.ReadMessage()
			if err != nil {
				return
			}
			var chat GameEvent
			json.Unmarshal(msg, &chat)
			chat.Type = "chat"
			chat.Player = playerName
			chat.Tick = match.tick.Load()

			encoded, _ := json.Marshal(chat)
			match.room.BroadcastExcept(encoded, c.ID())
		}
	})

	// Ability stream — reliable, ordered
	mux.Handle(StreamAbility, func(s *wt.Stream, _ *wt.Context) {
		defer s.Close()
		for {
			msg, err := s.ReadMessage()
			if err != nil {
				return
			}
			var ability struct {
				Type   string `json:"ability"`
				Target string `json:"target"`
			}
			if json.Unmarshal(msg, &ability) != nil {
				continue
			}

			switch ability.Type {
			case "attack":
				// Find closest enemy
				if ability.Target != "" {
					event := match.ApplyDamage(c.ID(), ability.Target, 25)
					if event != nil {
						encoded, _ := json.Marshal(event)
						match.room.Broadcast(encoded)

						if event.Type == "kill" {
							leaderboard.Update(playerName, 10, 1, 0)
							match.mu.RLock()
							if target, ok := match.players[ability.Target]; ok {
								leaderboard.Update(target.Name, 0, 0, 1)
							}
							match.mu.RUnlock()
						}
					}
				}
			case "heal":
				match.mu.Lock()
				if p, ok := match.players[c.ID()]; ok {
					p.Health = min(100, p.Health+30)
				}
				match.mu.Unlock()
			}
		}
	})

	// Control stream — match lifecycle
	mux.Handle(StreamControl, func(s *wt.Stream, _ *wt.Context) {
		defer s.Close()
		msg, _ := s.ReadMessage()
		var cmd struct{ Action string `json:"action"` }
		json.Unmarshal(msg, &cmd)

		if cmd.Action == "ready" {
			s.WriteJSON(map[string]string{"status": "ready"})
		}
	})

	mux.Serve(c)

	// Player disconnected — broadcast leave
	leaveEvent := GameEvent{
		Type:    "leave",
		Player:  playerName,
		Message: fmt.Sprintf("%s left the match", playerName),
		Tick:    match.tick.Load(),
	}
	leaveData, _ := json.Marshal(leaveEvent)
	match.room.Broadcast(leaveData)
}

func handleSpectate(c *wt.Context) {
	matchID := c.Param("id")
	match := matchmaker.GetMatch(matchID)
	if match == nil {
		c.CloseWithError(404, "match not found")
		return
	}

	tags.Tag(c.ID(), "spectator")
	defer tags.Untag(c.ID(), "spectator")

	slog.Info("spectator joined", "match", matchID, "id", c.ID())

	// Spectators only receive world state — no input accepted
	// This is a unidirectional push pattern
	tickRate := match.config.TickRate
	if tickRate <= 0 {
		tickRate = 20
	}
	ticker := time.NewTicker(time.Second / time.Duration(tickRate))
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			states := match.GetWorldState()
			data, _ := json.Marshal(map[string]any{
				"type":       "spectate_state",
				"players":    states,
				"tick":       match.tick.Load(),
				"player_cnt": match.PlayerCount(),
			})
			_ = c.SendDatagram(data)
		case <-c.Context().Done():
			return
		}
	}
}

func handleLeaderboard(c *wt.Context) {
	// Push leaderboard every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			top := leaderboard.Top(20)
			data, _ := json.Marshal(map[string]any{
				"type":        "leaderboard",
				"top_players": top,
			})
			_ = c.SendDatagram(data)
		case <-c.Context().Done():
			return
		}
	}
}

func handleRPC(c *wt.Context) {
	rpc := wt.NewRPCServer()

	rpc.Register("set_name", func(params json.RawMessage) (any, error) {
		var name string
		json.Unmarshal(params, &name)
		if name == "" {
			return nil, fmt.Errorf("name required")
		}
		c.Set("name", name)
		return map[string]string{"name": name}, nil
	})

	rpc.Register("get_stats", func(_ json.RawMessage) (any, error) {
		return map[string]any{
			"online":         presence.Count("lobby"),
			"in_match":       tags.Count("in-match"),
			"spectators":     tags.Count("spectator"),
			"active_matches": matchmaker.ActiveMatches(),
			"queue_size":     matchmaker.QueueSize(),
		}, nil
	})

	rpc.Register("get_leaderboard", func(params json.RawMessage) (any, error) {
		var n int
		json.Unmarshal(params, &n)
		if n <= 0 { n = 10 }
		return leaderboard.Top(n), nil
	})

	rpc.Register("get_presence", func(params json.RawMessage) (any, error) {
		var room string
		json.Unmarshal(params, &room)
		if room == "" { room = "lobby" }
		return presence.GetPresence(room), nil
	})

	// Handle RPC streams
	for stream := range wt.Streams(c) {
		go rpc.Serve(stream)
	}
}

func broadcastPresence(room *wt.Room, roomName string) {
	info := presence.GetPresenceJSON(roomName)
	room.Broadcast(info)
}

// ═══════════════════════════════════════════════════
// MAIN
// ═══════════════════════════════════════════════════

func main() {
	server := wt.New(
		wt.WithAddr(":4433"),
		wt.WithSelfSignedTLS(),
		wt.WithIdleTimeout(10*time.Minute),
		wt.WithQUICConfig(wt.GameServerQUICConfig()),
	)

	log.Printf("Arena game server")
	log.Printf("Certificate hash: %s", server.CertHash())

	// Middleware stack
	pm := middleware.NewPrometheusMetrics()
	server.Use(pm.Middleware())
	server.Use(middleware.Recover(nil))
	server.Use(middleware.DefaultLogger())
	server.Use(middleware.RequestID())
	server.Use(middleware.RateLimit(50))

	// Create a default match
	matchmaker.CreateMatch(MatchConfig{
		MaxPlayers: 8,
		MapName:    "arena_01",
		Duration:   5 * time.Minute,
		TickRate:   20,
	})

	// Routes
	server.Handle("/lobby", handleLobby)
	server.Handle("/match/{id}", handleMatch)
	server.Handle("/spectate/{id}", handleSpectate)
	server.Handle("/leaderboard", handleLeaderboard)
	server.Handle("/rpc", handleRPC)

	// Lifecycle hooks
	server.OnConnect(func(c *wt.Context) {
		slog.Info("client connected", "id", c.ID(), "remote", c.RemoteAddr())
	})
	server.OnDisconnect(func(c *wt.Context) {
		presence.Leave("lobby", c)
		tags.UntagAll(c.ID())
		slog.Info("client disconnected", "id", c.ID())
	})

	// HTTP debug/metrics server
	go func() {
		mux := wt.DebugMux(server)
		mux.Handle("/metrics", pm.Handler())
		mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write(server.StatsJSON())
		})
		log.Println("Debug: http://localhost:6060")
		http.ListenAndServe(":6060", mux)
	}()

	log.Printf("Listening on %s", server.Addr())
	log.Printf("Routes:")
	log.Printf("  /lobby           — matchmaking + chat")
	log.Printf("  /match/{id}      — game (datagrams 20Hz + streams)")
	log.Printf("  /spectate/{id}   — watch (receive-only)")
	log.Printf("  /leaderboard     — stats push")
	log.Printf("  /rpc             — JSON-RPC (account, stats)")

	wt.ListenAndServeWithGracefulShutdown(server, 30*time.Second)
}
