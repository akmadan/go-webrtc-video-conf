# WebRTC Video Conferencing - High Level Overview

## 1. What is WebRTC?

**WebRTC (Web Real-Time Communication)** is a free, open-source project that provides web browsers and mobile applications with real-time communication capabilities via simple APIs. It enables:

- **Peer-to-peer (P2P) communication** - Direct data exchange between browsers without going through a server
- **Audio/Video streaming** - Capture and transmit media streams
- **Data channels** - Send arbitrary data between peers
- **Low latency** - Optimized for real-time communication

### Key WebRTC Concepts

#### **Signaling**
- WebRTC doesn't handle the initial connection setup itself
- **Signaling** is the process of exchanging connection information (SDP offers/answers, ICE candidates) between peers
- This is done through a **signaling server** (our Go backend will handle this)
- Once peers are connected, media flows directly P2P

#### **SDP (Session Description Protocol)**
- Describes the media capabilities of a peer (codecs, resolutions, etc.)
- Exchanged during the connection setup phase
- Contains information about what media streams are available

#### **ICE (Interactive Connectivity Establishment)**
- Protocol to establish the best network path between peers
- Handles NAT traversal (connecting through firewalls/routers)
- Uses STUN/TURN servers to discover public IP addresses and relay traffic if needed

#### **STUN/TURN Servers**
- **STUN**: Discovers your public IP address and port
- **TURN**: Relays traffic when direct P2P connection isn't possible (due to strict firewalls)
- Often provided by third-party services (Google, Twilio, etc.)

### WebRTC Connection Flow

```
1. Peer A captures media (camera/microphone)
2. Peer A creates an offer (SDP)
3. Offer sent to signaling server → forwarded to Peer B
4. Peer B creates an answer (SDP)
5. Answer sent to signaling server → forwarded to Peer A
6. Both peers exchange ICE candidates (network paths)
7. Once connection established, media flows directly P2P
```

---

## 2. Pion Library in Go

**Pion WebRTC** is a pure Go implementation of the WebRTC API. It's the most popular WebRTC library for Go.

### Why Pion?

- **Pure Go** - No C dependencies, easier to deploy
- **Cross-platform** - Works on Linux, macOS, Windows, ARM
- **Well-maintained** - Active development, good documentation
- **Flexible** - Can be used for both client and server-side WebRTC

### Key Pion Components

#### **webrtc.PeerConnection**
- Represents a WebRTC connection between peers
- Handles media streams, data channels, and connection state

#### **webrtc.Track**
- Represents a media track (audio or video)
- Can be added to a PeerConnection to send/receive media

#### **webrtc.SessionDescription**
- Contains SDP offer/answer data
- Used during signaling

#### **webrtc.ICECandidate**
- Represents a network path candidate
- Exchanged during connection establishment

### Common Pion Patterns

```go
// Create a PeerConnection
pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
    ICEServers: []webrtc.ICEServer{
        {URLs: []string{"stun:stun.l.google.com:19302"}},
    },
})

// Add a track to send
track, err := pc.NewTrack(...)
pc.AddTrack(track)

// Handle incoming tracks
pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTRPSReceiver) {
    // Process received media
})
```

---

## 3. Architecture Overview

### High-Level Architecture

```
┌─────────────┐         ┌──────────────┐         ┌─────────────┐
│   Browser   │◄───────►│  Go Backend  │◄───────►│   Browser   │
│  (Next.js)  │ Signaling│ (Signaling)  │ Signaling│  (Next.js)  │
└─────────────┘         └──────────────┘         └─────────────┘
      │                                              │
      │                                              │
      └──────────────────────────────────────────────┘
                    Direct P2P Media Flow
                    (After connection established)
```

### Component Breakdown

#### **1. Next.js Frontend (Client)**
- **Purpose**: User interface and WebRTC client
- **Responsibilities**:
  - Capture audio/video from user's device
  - Connect to signaling server (WebSocket)
  - Exchange SDP offers/answers and ICE candidates
  - Display remote video streams
  - Handle UI interactions (join room, mute, etc.)

#### **2. Go Backend (Signaling Server)**
- **Purpose**: Facilitate connection setup between peers
- **Responsibilities**:
  - WebSocket server for signaling
  - Room management (create/join rooms)
  - Relay signaling messages between peers in the same room
  - Handle peer connections/disconnections
  - **Note**: After signaling, media flows P2P (not through server)

#### **3. STUN/TURN Servers** (External)
- **Purpose**: NAT traversal and connection establishment
- **Options**:
  - Free STUN: `stun:stun.l.google.com:19302`
  - Commercial TURN: Twilio, Cloudflare, etc. (for production)

### Detailed Flow

#### **Room Join Flow**

```
1. User A opens Next.js app
2. User A clicks "Join Room" → enters room ID
3. Frontend connects to Go backend via WebSocket
4. Backend adds User A to room
5. User B joins same room
6. Backend notifies User A that User B joined
```

#### **Connection Establishment Flow**

```
1. User A creates WebRTC offer
   └─> Frontend captures local media
   └─> Creates PeerConnection with offer
   └─> Sends offer to Go backend via WebSocket

2. Go backend receives offer
   └─> Forwards offer to User B via WebSocket

3. User B receives offer
   └─> Creates PeerConnection
   └─> Sets remote description (User A's offer)
   └─> Creates answer
   └─> Sends answer to Go backend

4. Go backend forwards answer to User A

5. Both peers exchange ICE candidates
   └─> Via Go backend (signaling)
   └─> ICE candidates help find best network path

6. Connection established
   └─> Media flows directly P2P
   └─> No longer needs Go backend (except for signaling updates)
```

### Technology Stack

#### **Frontend (Next.js)**
- **Next.js** - React framework for UI
- **WebRTC API** - Native browser APIs (`getUserMedia`, `RTCPeerConnection`)
- **WebSocket** - For signaling (or HTTP polling as fallback)
- **Styling** - Tailwind CSS or similar for modern UI

#### **Backend (Go)**
- **Gorilla WebSocket** or **nhooyr.io/websocket** - WebSocket server
- **Pion WebRTC** - WebRTC implementation (optional on server for advanced features)
- **Gin/Echo** - HTTP framework (for REST endpoints if needed)
- **Room management** - In-memory or Redis for room state

### Key Design Decisions

#### **Why Signaling Server?**
- WebRTC peers need to exchange connection info before connecting
- Can't do this directly without knowing each other's addresses
- Signaling server acts as a "matchmaker"

#### **Why P2P?**
- **Low latency** - Direct connection is faster
- **Reduced server load** - Media doesn't go through server
- **Cost effective** - Less bandwidth on server
- **Privacy** - Media flows directly between users

#### **Limitations of P2P**
- **NAT/Firewall issues** - May need TURN servers
- **Scalability** - Each peer connects to every other peer (mesh topology)
- **For larger groups**: Consider SFU (Selective Forwarding Unit) architecture

### Scalability Considerations

#### **For 2-4 users (Current Plan)**
- Simple mesh topology (each peer connects to all others)
- P2P works well

#### **For larger groups (Future)**
- Consider **SFU (Selective Forwarding Unit)**
- Server receives all streams, forwards to each peer
- More server resources but better for 5+ users
- Pion can be used to build SFU in Go

---

## 4. Project Structure (Planned)

```
go-webrtc-video-conf/
├── backend/
│   ├── main.go              # Entry point
│   ├── server/
│   │   ├── websocket.go     # WebSocket handler
│   │   ├── room.go          # Room management
│   │   └── signaling.go     # Signaling logic
│   └── go.mod
├── frontend/
│   ├── pages/
│   │   ├── index.tsx        # Home page
│   │   └── room/[id].tsx    # Room page
│   ├── components/
│   │   ├── VideoCall.tsx    # Main video component
│   │   └── Controls.tsx     # Mute, video toggle, etc.
│   ├── hooks/
│   │   └── useWebRTC.ts     # WebRTC logic hook
│   └── package.json
└── README.md
```

---

## 5. Next Steps

1. **Set up Go backend** with WebSocket server
2. **Set up Next.js frontend** with basic UI
3. **Implement signaling** - Exchange SDP offers/answers
4. **Add media capture** - Get user's camera/microphone
5. **Establish P2P connection** - Handle ICE candidates
6. **Display video streams** - Show local and remote video
7. **Add controls** - Mute, video toggle, leave room
8. **Polish UI** - Make it beautiful and user-friendly

---

## Resources

- **WebRTC Spec**: https://www.w3.org/TR/webrtc/
- **Pion Documentation**: https://pion.ly/docs/
- **MDN WebRTC Guide**: https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API
- **STUN/TURN Servers**: Consider using free STUN or services like Twilio for TURN

