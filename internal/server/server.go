package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/akshitmadan/go-webrtc-video-conf/internal/config"
	"github.com/akshitmadan/go-webrtc-video-conf/internal/observability"
	"github.com/akshitmadan/go-webrtc-video-conf/internal/signaling"
	"github.com/akshitmadan/go-webrtc-video-conf/internal/websocket"
)

// Server wraps the HTTP server and router
type Server struct {
	httpServer *http.Server
	config     *config.Config
	hub        *websocket.Hub
	metrics    *observability.Metrics
	bus        signaling.EventBus
	busCancel  context.CancelFunc
}

// New creates a new server instance
func New(cfg *config.Config) *Server {
	metrics := observability.NewMetrics()
	var bus signaling.EventBus = signaling.NewNoopBus()
	if cfg.Redis.Enabled {
		redisBus, err := signaling.NewRedisBus(
			cfg.Redis.Addr,
			cfg.Redis.Password,
			cfg.Redis.DB,
			cfg.Redis.Channel,
		)
		if err != nil {
			slog.Error("failed to initialize redis bus, falling back to no-op bus", "error", err.Error())
		} else {
			bus = redisBus
		}
	} else if cfg.Signaling.PublishEventsToLog {
		bus = signaling.NewLogBus()
	}

	// Create WebSocket hub
	hub := websocket.NewHub(cfg.Limits.MaxPeersPerRoom, bus, metrics)
	busCtx, busCancel := context.WithCancel(context.Background())
	if err := bus.Subscribe(busCtx, hub.HandleExternalEvent); err != nil {
		slog.Error("failed to subscribe to signaling bus", "error", err.Error())
	}

	mux := http.NewServeMux()

	// Register routes
	registerRoutes(mux, cfg, hub, metrics)

	srv := &http.Server{
		Addr:         cfg.GetAddress(),
		Handler:      corsMiddleware(recoveryMiddleware(httpRateLimitMiddleware(metrics.HTTPMiddleware(mux), cfg), cfg), cfg),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: srv,
		config:     cfg,
		hub:        hub,
		metrics:    metrics,
		bus:        bus,
		busCancel:  busCancel,
	}
}

// ListenAndServe starts the HTTP server
func (s *Server) ListenAndServe() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.busCancel != nil {
		s.busCancel()
	}
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return err
	}
	return s.bus.Close()
}

// registerRoutes registers all HTTP routes
func registerRoutes(mux *http.ServeMux, cfg *config.Config, hub *websocket.Hub, metrics *observability.Metrics) {
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/ice-servers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"iceServers": cfg.GetICEServers(),
		}); err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}
	})

	// WebSocket endpoint for signaling
	mux.HandleFunc("/ws", websocket.HandleWebSocket(hub, cfg))

	slog.Info("routes registered", "routes", "/health,/ready,/metrics,/ice-servers,/ws")
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(next http.Handler, cfg *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		
		// Check if origin is allowed
		allowed := false
		for _, allowedOrigin := range cfg.CORS.AllowedOrigins {
			if origin == allowedOrigin {
				allowed = true
				break
			}
		}

		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

