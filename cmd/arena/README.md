# Arena — Multiplayer Game Server

A production-grade multiplayer game server built on the `wt` WebTransport framework.

## Why WebTransport, not WebSocket?

| Feature | WebSocket | Arena (WebTransport) |
|---------|-----------|---------------------|
| Position updates | Reliable (wasteful) | **Datagrams** (fire-and-forget at 60Hz) |
| Lost packet | Freezes ALL data | **Only affects that one datagram** |
| Chat + game events | One pipe (blocks each other) | **Independent streams** (chat can't freeze combat) |
| Network switch | Connection drops | **QUIC migration** (seamless) |

## Endpoints

| Path | Transport | Purpose |
|------|-----------|---------|
| `/lobby` | Streams + Datagrams | Matchmaking, chat, presence |
| `/match/{id}` | Streams + Datagrams | Game play (20Hz state, abilities, chat) |
| `/spectate/{id}` | Datagrams only | Watch a match (receive-only) |
| `/leaderboard` | Datagrams | Top players push every 5s |
| `/rpc` | Streams | JSON-RPC for account/settings/stats |

## Run

```bash
go run ./cmd/arena
```

## Architecture

```
Client connects to /lobby
  ├─ Datagrams: receive lobby status (online count, queue, matches)
  ├─ Stream type 1: chat messages
  └─ Stream type 3: match control (queue, create)

Client joins /match/{id}
  ├─ Datagrams IN:  player input (position, velocity) at 60Hz
  ├─ Datagrams OUT: world state (all players) at 20Hz
  ├─ Stream type 1: in-match chat
  ├─ Stream type 2: abilities (attack, heal)
  └─ Stream type 3: match lifecycle (ready, leave)

Spectator connects to /spectate/{id}
  └─ Datagrams OUT: world state at 20Hz (read-only)
```
