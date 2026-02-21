package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/akshitmadan/go-webrtc-video-conf/internal/observability"
	"github.com/akshitmadan/go-webrtc-video-conf/internal/signaling"
	"github.com/google/uuid"
)

var ErrRoomFull = errors.New("room is full")

// Hub maintains the set of active clients and broadcasts messages to rooms
type Hub struct {
	// Registered clients by room
	rooms map[string]map[string]*Client

	// Mutex for thread-safe operations
	mu sync.RWMutex

	// Runtime limits
	maxPeersPerRoom int
	bus             signaling.EventBus
	metrics         *observability.Metrics
	instanceID      string
}

// NewHub creates a new hub instance
func NewHub(maxPeersPerRoom int, bus signaling.EventBus, metrics *observability.Metrics) *Hub {
	if maxPeersPerRoom <= 0 {
		maxPeersPerRoom = 8
	}
	if bus == nil {
		bus = signaling.NewNoopBus()
	}
	return &Hub{
		rooms:           make(map[string]map[string]*Client),
		maxPeersPerRoom: maxPeersPerRoom,
		bus:             bus,
		metrics:         metrics,
		instanceID:      uuid.New().String(),
	}
}

// registerClient adds a client to a room
func (h *Hub) registerClient(client *Client) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.RoomID == "" {
		slog.Warn("client attempted to register without room id", "client_id", client.ID)
		return nil
	}

	// Initialize room if it doesn't exist
	if h.rooms[client.RoomID] == nil {
		h.rooms[client.RoomID] = make(map[string]*Client)
	}
	if len(h.rooms[client.RoomID]) >= h.maxPeersPerRoom {
		return ErrRoomFull
	}

	// Add client to room
	h.rooms[client.RoomID][client.ID] = client

	slog.Info("client joined room",
		"client_id", client.ID,
		"room_id", client.RoomID,
		"room_size", len(h.rooms[client.RoomID]),
	)

	peers := h.roomPeerSummariesUnsafe(client.RoomID, client.ID)

	// Notify other clients in the room about the new peer
	h.broadcastToRoomUnsafe(client.RoomID, Message{
		Type:      MessageTypePeerJoined,
		RoomID:    client.RoomID,
		PeerID:    client.ID,
		Data:      mustJSON(map[string]string{"name": client.Name}),
		Timestamp: time.Now().Unix(),
	}, client.ID) // Exclude the sender

	// Send confirmation to the new client
	client.Send <- Message{
		Type:      MessageTypeRoomJoined,
		RoomID:    client.RoomID,
		PeerID:    client.ID,
		Data:      mustJSON(map[string][]PeerSummary{"peers": peers}),
		Timestamp: time.Now().Unix(),
	}
	_ = h.bus.Publish(context.Background(), signaling.Event{
		Type:      signaling.EventPeerJoined,
		SourceID:  h.instanceID,
		RoomID:    client.RoomID,
		PeerID:    client.ID,
		Data:      mustJSON(map[string]string{"name": client.Name}),
		Timestamp: time.Now().Unix(),
	})

	return nil
}

// leaveRoom removes a client from a room.
func (h *Hub) leaveRoom(client *Client, closeClient bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if client.RoomID == "" {
		return
	}

	room, exists := h.rooms[client.RoomID]
	if !exists {
		return
	}

	if _, clientExists := room[client.ID]; clientExists {
		delete(room, client.ID)
		slog.Info("client left room",
			"client_id", client.ID,
			"room_id", client.RoomID,
			"room_size", len(room),
		)

		// Notify other clients in the room
		h.broadcastToRoomUnsafe(client.RoomID, Message{
			Type:      MessageTypePeerLeft,
			RoomID:    client.RoomID,
			PeerID:    client.ID,
			Timestamp: time.Now().Unix(),
		}, "")

		// Clean up empty rooms
		if len(room) == 0 {
			delete(h.rooms, client.RoomID)
			slog.Info("room removed", "room_id", client.RoomID)
		}
	}
	_ = h.bus.Publish(context.Background(), signaling.Event{
		Type:      signaling.EventPeerLeft,
		SourceID:  h.instanceID,
		RoomID:    client.RoomID,
		PeerID:    client.ID,
		Timestamp: time.Now().Unix(),
	})
	client.RoomID = ""

	if closeClient {
		client.Close()
	}
}

// Disconnect unregisters a client from any room and closes it.
func (h *Hub) Disconnect(client *Client) {
	h.leaveRoom(client, true)
}

// HandleMessage processes incoming messages from clients
func (h *Hub) HandleMessage(client *Client, msg Message) {
	if err := validateMessage(msg); err != nil {
		client.Send <- Message{
			Type:  MessageTypeError,
			Error: err.Error(),
		}
		return
	}

	switch msg.Type {
	case MessageTypeJoin:
		if len(msg.Data) > 0 {
			var joinData JoinData
			if err := json.Unmarshal(msg.Data, &joinData); err == nil {
				client.Name = sanitizeName(joinData.Name, client.ID)
			}
		} else if client.Name == "" {
			client.Name = sanitizeName("", client.ID)
		}

		// Update client's room
		oldRoomID := client.RoomID
		client.RoomID = msg.RoomID

		// If client was in another room, unregister from it first
		if oldRoomID != "" && oldRoomID != msg.RoomID {
			client.RoomID = oldRoomID
			h.leaveRoom(client, false)
			client.RoomID = msg.RoomID
		}

		// Register client to new room
		if err := h.registerClient(client); err != nil {
			client.RoomID = oldRoomID
			client.Send <- Message{
				Type:  MessageTypeError,
				Error: err.Error(),
			}
		}

	case MessageTypeLeave:
		h.leaveRoom(client, false)

	case MessageTypeOffer, MessageTypeAnswer, MessageTypeICECandidate:
		if client.RoomID == "" {
			client.Send <- Message{
				Type:  MessageTypeError,
				Error: "join a room first",
			}
			return
		}
		msg.PeerID = client.ID
		msg.RoomID = client.RoomID
		msg.Timestamp = time.Now().Unix()

		// Forward signaling messages to target peer or all peers in room
		h.mu.RLock()
		if msg.TargetID != "" {
			// Send to specific peer
			h.sendToPeerUnsafe(client.RoomID, msg.TargetID, msg)
		} else {
			// Broadcast to all peers in room except sender
			h.broadcastToRoomUnsafe(client.RoomID, msg, client.ID)
		}
		h.mu.RUnlock()
		_ = h.bus.Publish(context.Background(), signaling.Event{
			Type:        signaling.EventSignaling,
			SourceID:    h.instanceID,
			RoomID:      msg.RoomID,
			PeerID:      msg.PeerID,
			TargetID:    msg.TargetID,
			MessageType: string(msg.Type),
			Data:        msg.Data,
			Timestamp:   time.Now().Unix(),
		})

	default:
		slog.Warn("unknown websocket message type", "type", msg.Type, "client_id", client.ID)
		client.Send <- Message{
			Type:  MessageTypeError,
			Error: "Unknown message type",
		}
	}
}

// HandleExternalEvent processes events received from a distributed bus (e.g. Redis).
func (h *Hub) HandleExternalEvent(event signaling.Event) {
	if event.SourceID == h.instanceID {
		return
	}

	switch event.Type {
	case signaling.EventPeerJoined:
		var joinData JoinData
		if len(event.Data) > 0 {
			_ = json.Unmarshal(event.Data, &joinData)
		}
		h.mu.RLock()
		h.broadcastToRoomUnsafe(event.RoomID, Message{
			Type:      MessageTypePeerJoined,
			RoomID:    event.RoomID,
			PeerID:    event.PeerID,
			Data:      mustJSON(map[string]string{"name": joinData.Name}),
			Timestamp: event.Timestamp,
		}, event.PeerID)
		h.mu.RUnlock()
	case signaling.EventPeerLeft:
		h.mu.RLock()
		h.broadcastToRoomUnsafe(event.RoomID, Message{
			Type:      MessageTypePeerLeft,
			RoomID:    event.RoomID,
			PeerID:    event.PeerID,
			Timestamp: event.Timestamp,
		}, "")
		h.mu.RUnlock()
	case signaling.EventSignaling:
		msg := Message{
			Type:      MessageType(event.MessageType),
			RoomID:    event.RoomID,
			PeerID:    event.PeerID,
			TargetID:  event.TargetID,
			Data:      event.Data,
			Timestamp: event.Timestamp,
		}
		h.mu.RLock()
		if event.TargetID != "" {
			h.sendToPeerUnsafe(event.RoomID, event.TargetID, msg)
		} else {
			h.broadcastToRoomUnsafe(event.RoomID, msg, event.PeerID)
		}
		h.mu.RUnlock()
	default:
		slog.Warn("unknown signaling event type from bus", "type", event.Type)
	}
}

// broadcastToRoom sends a message to all clients in a room
func (h *Hub) broadcastToRoom(roomID string, msg Message, excludeClientID string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.broadcastToRoomUnsafe(roomID, msg, excludeClientID)
}

// broadcastToRoomUnsafe sends a message to all clients in a room (assumes lock is held)
func (h *Hub) broadcastToRoomUnsafe(roomID string, msg Message, excludeClientID string) {
	room, exists := h.rooms[roomID]
	if !exists {
		return
	}

	for id, client := range room {
		if id != excludeClientID && !client.IsClosed {
			select {
			case client.Send <- msg:
				if h.metrics != nil {
					h.metrics.WSMessageOut(string(msg.Type))
				}
			default:
				slog.Warn("failed to send message to client (channel full)", "client_id", id)
			}
		}
	}
}

// sendToPeer sends a message to a specific peer
func (h *Hub) sendToPeer(roomID, peerID string, msg Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	h.sendToPeerUnsafe(roomID, peerID, msg)
}

// sendToPeerUnsafe sends a message to a specific peer (assumes lock is held)
func (h *Hub) sendToPeerUnsafe(roomID, peerID string, msg Message) {
	room, ok := h.rooms[roomID]
	if !ok {
		slog.Warn("room not found for peer", "room_id", roomID, "peer_id", peerID)
		return
	}
	client, exists := room[peerID]
	if !exists || client.IsClosed {
		slog.Warn("peer not found in room", "peer_id", peerID, "room_id", roomID)
		return
	}

	select {
	case client.Send <- msg:
		if h.metrics != nil {
			h.metrics.WSMessageOut(string(msg.Type))
		}
	default:
		slog.Warn("failed to send message to peer (channel full)", "peer_id", peerID)
	}
}

// GenerateClientID generates a unique client ID
func GenerateClientID() string {
	return uuid.New().String()
}

// GetRoomClients returns all client IDs in a room
func (h *Hub) GetRoomClients(roomID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.getRoomClientsUnsafe(roomID)
}

// GetRoomClientsExcluding returns all client IDs except one in a room.
func (h *Hub) GetRoomClientsExcluding(roomID, excludeID string) []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, exists := h.rooms[roomID]
	if !exists {
		return []string{}
	}

	clients := make([]string, 0, len(room))
	for id := range room {
		if id != excludeID {
			clients = append(clients, id)
		}
	}

	return clients
}

func (h *Hub) getRoomClientsUnsafe(roomID string) []string {
	room, exists := h.rooms[roomID]
	if !exists {
		return []string{}
	}

	clients := make([]string, 0, len(room))
	for id := range room {
		clients = append(clients, id)
	}

	return clients
}

func (h *Hub) roomPeerSummariesUnsafe(roomID, excludeID string) []PeerSummary {
	room, exists := h.rooms[roomID]
	if !exists {
		return []PeerSummary{}
	}
	peers := make([]PeerSummary, 0, len(room))
	for id, client := range room {
		if id == excludeID {
			continue
		}
		peers = append(peers, PeerSummary{
			PeerID: id,
			Name:   client.Name,
		})
	}
	return peers
}

func mustJSON(value any) []byte {
	payload, err := json.Marshal(value)
	if err != nil {
		return []byte(`{}`)
	}
	return payload
}

func sanitizeName(name, fallbackID string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		short := fallbackID
		if len(short) > 8 {
			short = short[:8]
		}
		return "Guest-" + short
	}
	if len(clean) > 64 {
		return clean[:64]
	}
	return clean
}

