package websocket

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/akshitmadan/go-webrtc-video-conf/internal/config"
	"github.com/gorilla/websocket"
)

// HandleWebSocket handles WebSocket connections
func HandleWebSocket(hub *Hub, cfg *config.Config) http.HandlerFunc {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			for _, allowed := range cfg.CORS.AllowedOrigins {
				if origin == allowed {
					return true
				}
			}
			return false
		},
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if !isAuthorized(r, cfg.Security.SignalingToken) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		// Upgrade HTTP connection to WebSocket
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			slog.Warn("websocket upgrade failed", "error", err.Error())
			return
		}

		// Generate unique client ID
		clientID := GenerateClientID()
		slog.Info("websocket connection opened", "client_id", clientID)
		if hub.metrics != nil {
			hub.metrics.WSConnectionOpened()
		}

		// Create new client
		client := NewClient(clientID, conn, hub, cfg.Limits.MaxWSMessagesPerSec)

		// Start client's read and write pumps
		go client.WritePump()
		go client.ReadPump()
	}
}

func isAuthorized(r *http.Request, signalingToken string) bool {
	if signalingToken == "" {
		return true
	}
	if r.URL.Query().Get("token") == signalingToken {
		return true
	}
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		return token == signalingToken
	}
	return false
}

