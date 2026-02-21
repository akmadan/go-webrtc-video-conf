package server

import (
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/akshitmadan/go-webrtc-video-conf/internal/config"
)

func recoveryMiddleware(next http.Handler, _ *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "panic", rec)
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type ipBucket struct {
	windowFrom time.Time
	count      int
}

func httpRateLimitMiddleware(next http.Handler, cfg *config.Config) http.Handler {
	var (
		mu      sync.Mutex
		buckets = map[string]*ipBucket{}
		maxRPM  = cfg.Limits.MaxHTTPPerMinute
	)
	if maxRPM <= 0 {
		maxRPM = 300
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow health checks freely for probes.
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}

		ip := clientIP(r.RemoteAddr)
		now := time.Now()

		mu.Lock()
		bucket, ok := buckets[ip]
		if !ok {
			bucket = &ipBucket{windowFrom: now}
			buckets[ip] = bucket
		}
		if now.Sub(bucket.windowFrom) >= time.Minute {
			bucket.windowFrom = now
			bucket.count = 0
		}
		bucket.count++
		allowed := bucket.count <= maxRPM
		mu.Unlock()

		if !allowed {
			http.Error(w, "too many requests", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func clientIP(remoteAddr string) string {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}
	return host
}

