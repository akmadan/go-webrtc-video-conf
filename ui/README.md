# PulseMeet UI (Next.js)

Modern frontend for the WebRTC conferencing app.

## Features

- Beautiful dark/glass UI
- Room-based call flow (`/room/:roomId`)
- Local media capture (mic + camera)
- P2P WebRTC negotiation over Go signaling server
- Dynamic ICE server fetch from backend (`/ice-servers`)
- Call controls: mute/unmute, camera toggle, leave room

## Setup

1. Install dependencies:

```bash
npm install
```

2. Configure environment:

```bash
cp config.example.env .env.local
```

3. Start frontend:

```bash
npm run dev
```

App runs at `http://localhost:3000`.

## Backend Requirements

Run your Go backend at `http://localhost:8080` (or update env vars).  
The UI expects:

- `GET /ice-servers`
- `WS /ws`

## Local Testing

1. Open `http://localhost:3000`
2. Enter a room ID (or share one)
3. Open the same room in a second tab/window
4. Allow camera/mic permissions
5. Verify local + remote video streams

## Environment Variables

- `NEXT_PUBLIC_BACKEND_HTTP_URL` - backend base URL
- `NEXT_PUBLIC_SIGNALING_WS_URL` - websocket signaling URL
- `NEXT_PUBLIC_SIGNALING_TOKEN` - optional signaling token (if backend uses one)
