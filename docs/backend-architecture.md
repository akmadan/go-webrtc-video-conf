# Go Backend Architecture

## Purpose

The Go backend is a signaling and coordination layer for WebRTC peers.  
It does **not** relay media in the current architecture (mesh P2P).

## Package Layout

- `cmd/server` - app entrypoint, bootstrap, graceful shutdown
- `internal/config` - env-driven config
- `internal/server` - HTTP router + middleware
- `internal/websocket` - signaling protocol, room hub, peer lifecycle
- `internal/signaling` - event bus abstraction + Redis pub/sub implementation
- `internal/observability` - structured logs and Prometheus metrics

## Runtime Components

1. HTTP server boots with middleware stack.
2. WebSocket clients connect to `/ws`.
3. Clients send signaling messages (`join`, `offer`, `answer`, `ice-candidate`, `leave`).
4. Hub manages rooms and forwards messages to target peers.
5. Optional Redis bus propagates events across multiple backend instances.

## Room Model

- Room is created on first peer join.
- Peer joins and receives existing peer summaries (including names).
- Peer leave triggers notification and room cleanup when empty.
- Room size is limited via config.

## Observability

- JSON logs via `slog`.
- Metrics on `/metrics`:
  - HTTP requests, latency
  - WebSocket active connections
  - WebSocket message in/out counters

## Security and Limits

- Optional signaling token auth
- Origin allowlist checks
- HTTP and WS message rate limits
- Message schema validation

