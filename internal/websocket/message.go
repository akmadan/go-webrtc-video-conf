package websocket

import "encoding/json"

// MessageType represents the type of WebSocket message
type MessageType string

const (
	// MessageTypeJoin is sent when a client joins a room
	MessageTypeJoin MessageType = "join"
	// MessageTypeLeave is sent when a client leaves a room
	MessageTypeLeave MessageType = "leave"
	// MessageTypeOffer is sent when a client sends an SDP offer
	MessageTypeOffer MessageType = "offer"
	// MessageTypeAnswer is sent when a client sends an SDP answer
	MessageTypeAnswer MessageType = "answer"
	// MessageTypeICECandidate is sent when a client sends an ICE candidate
	MessageTypeICECandidate MessageType = "ice-candidate"
	// MessageTypeError is sent when an error occurs
	MessageTypeError MessageType = "error"
	// MessageTypeRoomJoined is sent to confirm a client joined a room
	MessageTypeRoomJoined MessageType = "room-joined"
	// MessageTypePeerJoined is sent when a new peer joins the room
	MessageTypePeerJoined MessageType = "peer-joined"
	// MessageTypePeerLeft is sent when a peer leaves the room
	MessageTypePeerLeft MessageType = "peer-left"
)

// Message represents a WebSocket message
type Message struct {
	Type      MessageType `json:"type"`
	RoomID    string      `json:"roomId,omitempty"`
	PeerID    string      `json:"peerId,omitempty"`
	TargetID  string      `json:"targetId,omitempty"` // For targeting specific peer
	Data      json.RawMessage `json:"data,omitempty"` // SDP, ICE candidate, join metadata, etc.
	Error     string      `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp,omitempty"`
}

// JoinData carries user metadata sent during join.
type JoinData struct {
	Name string `json:"name,omitempty"`
}

// PeerSummary is shared in room events to expose participant identity.
type PeerSummary struct {
	PeerID string `json:"peerId"`
	Name   string `json:"name,omitempty"`
}

// SDPData represents Session Description Protocol data
type SDPData struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"` // "offer" or "answer"
}

// ICECandidateData represents ICE candidate data
type ICECandidateData struct {
	Candidate     string `json:"candidate"`
	SDPMLineIndex *int   `json:"sdpMLineIndex,omitempty"`
	SDPMid        string `json:"sdpMid,omitempty"`
}

