package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/akshitmadan/go-webrtc-video-conf/internal/config"
	"github.com/akshitmadan/go-webrtc-video-conf/internal/observability"
	"github.com/akshitmadan/go-webrtc-video-conf/internal/server"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err.Error())
		os.Exit(1)
	}
	observability.InitLogger(cfg.LogLevel)

	// Create HTTP server
	srv := server.New(cfg)

	// Start server in a goroutine
	go func() {
		address := cfg.GetAddress()
		slog.Info("server starting", "address", address)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed to start", "error", err.Error())
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("server forced shutdown", "error", err.Error())
		os.Exit(1)
	}

	slog.Info("server exited")
}

