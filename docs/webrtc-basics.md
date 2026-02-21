# How WebRTC Works

WebRTC allows browsers to exchange audio/video/data in real time.

## Core APIs

- `getUserMedia` - capture camera/mic
- `RTCPeerConnection` - connect peers and exchange media
- `RTCDataChannel` - arbitrary peer-to-peer data

## Negotiation (Offer/Answer)

1. Peer A creates an SDP offer.
2. Offer sent through signaling server to Peer B.
3. Peer B sets remote description, creates SDP answer.
4. Answer sent back via signaling.

## ICE Candidate Exchange

- Both peers gather ICE candidates (possible network paths).
- Candidates exchanged through signaling.
- Best viable path is selected.

## Connection States

- `new` -> `connecting` -> `connected` (ideal)
- `disconnected` / `failed` / `closed` (error or teardown)

## Mesh Topology (Current Project)

- Each participant connects directly to every other participant.
- Works well for small rooms.
- For larger rooms, SFU architecture is recommended.

