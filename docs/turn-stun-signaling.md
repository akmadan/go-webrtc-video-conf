# STUN, TURN, and Signaling Server

These three are related but different roles in WebRTC.

## STUN Server

- Helps discover public-facing IP/port.
- Enables direct P2P where NAT conditions allow.
- Usually low-cost/free.

## TURN Server

- Relay fallback when direct P2P fails.
- Required for many real-world network combinations.
- Higher bandwidth cost because media may pass through TURN relay.

## Signaling Server (This Go Backend)

- Exchanges offer/answer and ICE candidates.
- Manages rooms and peer join/leave.
- Does not carry media in current mesh architecture.

## When You Need TURN

- Different networks, strict firewalls, symmetric NATs
- Corporate/VPN environments
- Friend testing over internet

## Local Development Guidance

- Same device or same LAN: usually works with STUN only.
- Internet testing: enable TURN and verify `/ice-servers`.

