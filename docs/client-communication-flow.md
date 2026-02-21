# Client Communication Flow

## Pre-Join

1. User opens room route.
2. Pre-join lobby requests local media preview.
3. User selects name/devices.
4. User clicks Join.

## Join and Signaling

1. Client opens WebSocket to `/ws`.
2. Sends `join` with room ID and optional name.
3. Backend responds `room-joined` with current peers.
4. Existing peers receive `peer-joined`.

## Offer/Answer Exchange

1. Existing peer creates offer and sends to new peer.
2. New peer sets remote offer, creates answer.
3. Existing peer sets remote answer.

## ICE Exchange

1. Both sides send `ice-candidate` messages.
2. Candidate added to remote peer connection.
3. Media path established.

## Media Lifecycle

- Local tracks added to each `RTCPeerConnection`.
- Device switch replaces sender tracks (`replaceTrack`).
- Mute/camera toggles enable or disable local tracks.

## Leave / Disconnect

- Peer sends `leave` or socket closes.
- Backend emits `peer-left`.
- Frontend removes peer connection and remote tile.

