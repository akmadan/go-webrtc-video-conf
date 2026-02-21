package websocket

import (
	"log/slog"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period (must be less than pongWait)
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512 * 1024 // 512 KB
)

// Client represents a WebSocket client connection
type Client struct {
	ID       string
	RoomID   string
	Name     string
	Conn     *websocket.Conn
	Send     chan Message
	Hub      *Hub
	Limiter  *MessageLimiter
	mu       sync.Mutex
	IsClosed bool
}

// NewClient creates a new WebSocket client
func NewClient(id string, conn *websocket.Conn, hub *Hub, maxMessagesPerSec int) *Client {
	return &Client{
		ID:       id,
		Conn:     conn,
		Send:     make(chan Message, 256),
		Hub:      hub,
		Limiter:  NewMessageLimiter(maxMessagesPerSec),
		IsClosed: false,
	}
}

// ReadPump pumps messages from the WebSocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Disconnect(c)
		c.Conn.Close()
		if c.Hub.metrics != nil {
			c.Hub.metrics.WSConnectionClosed()
		}
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				slog.Warn("websocket read error",
					"client_id", c.ID,
					"error", err.Error(),
				)
			}
			break
		}
		if c.Hub.metrics != nil {
			c.Hub.metrics.WSMessageIn(string(msg.Type))
		}
		if !c.Limiter.Allow() {
			c.Send <- Message{
				Type:  MessageTypeError,
				Error: "rate limit exceeded",
			}
			break
		}

		// Handle the message
		c.Hub.HandleMessage(c, msg)
	}
}

// WritePump pumps messages from the hub to the WebSocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				slog.Warn("websocket write error",
					"client_id", c.ID,
					"error", err.Error(),
				)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// Close safely closes the client connection
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.IsClosed {
		return
	}

	c.IsClosed = true
	close(c.Send)
	c.Conn.Close()
}

