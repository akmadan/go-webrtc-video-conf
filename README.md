# Go WebRTC Video Conf

Peer-to-peer video conferencing app built with:

- Go signaling backend (`cmd/`, `internal/`)
- Next.js frontend (`ui/`)
- WebRTC for media transport

## Project Structure

```text
go-webrtc-video-conf/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── config/
│   ├── observability/
│   ├── server/
│   ├── signaling/
│   └── websocket/
├── ui/
│   ├── src/
│   └── ...
├── config.example.env
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── go.mod
├── go.sum
├── docs/
│   ├── README.md
│   ├── backend-architecture.md
│   ├── webrtc-basics.md
│   ├── turn-stun-signaling.md
│   ├── client-communication-flow.md
│   └── WEBRTC_OVERVIEW.md
└── README.md
```

## Prerequisites

- Go `1.21+`
- Node.js `20+`
- npm `10+`

## 1) Start Backend (No TURN, Local-Only)

From repository root:

```bash
cp config.example.env .env
```

Use these values in `.env`:

```env
TURN_ENABLED=false
REDIS_ENABLED=false
SIGNALING_TOKEN=
```

Run backend:

```bash
go mod tidy
go run cmd/server/main.go
```

Backend endpoints:

- `GET http://localhost:8080/health`
- `GET http://localhost:8080/ready`
- `GET http://localhost:8080/metrics`
- `GET http://localhost:8080/ice-servers`
- `WS  ws://localhost:8080/ws`

## 2) Start UI

```bash
cd ui
cp config.example.env .env.local
npm install
npm run dev
```

Default UI env:

```env
NEXT_PUBLIC_BACKEND_HTTP_URL=http://localhost:8080
NEXT_PUBLIC_SIGNALING_WS_URL=ws://localhost:8080/ws
NEXT_PUBLIC_SIGNALING_TOKEN=
```

Open `http://localhost:3000`.

## 3) Start Backend With TURN (for friend testing)

Set in root `.env`:

```env
TURN_ENABLED=true
TURN_URLS=turn:<YOUR_PUBLIC_IP>:3478?transport=udp,turn:<YOUR_PUBLIC_IP>:3478?transport=tcp
TURN_USERNAME=testuser
TURN_PASSWORD=testpassword
TURN_REALM=webrtc-local
```

Keep UI env same unless signaling token is enabled.

## 4) Run With Docker

### Build and run both services

```bash
cp config.example.env .env
docker compose up --build
```

This starts:

- Backend on `http://localhost:8080`
- UI on `http://localhost:3000`

### Run detached

```bash
docker compose up -d --build
```

### Stop services

```bash
docker compose down
```

### Notes

- `docker-compose.yml` injects UI envs:
  - `NEXT_PUBLIC_BACKEND_HTTP_URL=http://localhost:8080`
  - `NEXT_PUBLIC_SIGNALING_WS_URL=ws://localhost:8080/ws`
- Backend envs are loaded from root `.env`.
- For TURN testing with Docker, set TURN vars in root `.env` before `docker compose up`.

### TURN profile with coturn container

`docker-compose.yml` includes a `coturn` service under profile `turn`.

Run app + coturn:

```bash
docker compose --profile turn up --build
```

Detached:

```bash
docker compose --profile turn up -d --build
```

When using this profile locally, set these in `.env`:

```env
TURN_ENABLED=true
TURN_URLS=turn:localhost:3478?transport=udp,turn:localhost:3478?transport=tcp
TURN_USERNAME=testuser
TURN_PASSWORD=testpassword
TURN_REALM=webrtc-local
```

For internet friend testing, replace `localhost` with your public host/IP.

## 5) Makefile Shortcuts

```bash
make up         # docker compose up --build
make upd        # docker compose up -d --build
make down       # docker compose down
make logs       # docker compose logs -f
make up-turn    # docker compose --profile turn up --build
make up-turn-d  # docker compose --profile turn up -d --build
```

## Coturn Requirements and Setup

Install coturn on a machine with public IP (VPS recommended):

```bash
sudo apt update
sudo apt install -y coturn
sudo turnadmin -a -u testuser -p testpassword -r webrtc-local
```

Minimal `/etc/turnserver.conf`:

```conf
listening-port=3478
fingerprint
lt-cred-mech
realm=webrtc-local
user=testuser:testpassword
external-ip=<YOUR_PUBLIC_IP>
no-cli
no-multicast-peers
```

Start service:

```bash
sudo systemctl enable coturn
sudo systemctl restart coturn
sudo systemctl status coturn
```

Firewall ports:

- UDP `3478`
- TCP `3478`
- UDP relay range `49152-65535`

## Quick Test Checklist

- Open UI in two tabs.
- Join same room from pre-join lobby.
- Verify both local and remote video.
- Toggle mic/camera and ensure no phantom peers appear.
- If internet/friend test fails, verify `GET /ice-servers` includes TURN entries.

## Documentation

See `docs/README.md` for architecture and concept docs.

