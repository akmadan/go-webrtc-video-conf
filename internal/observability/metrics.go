package observability

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry *prometheus.Registry

	httpRequestsTotal    *prometheus.CounterVec
	httpDurationSeconds  *prometheus.HistogramVec
	wsConnectionsCurrent prometheus.Gauge
	wsMessagesTotal      *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	m := &Metrics{
		registry: registry,
		httpRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_http_requests_total",
				Help: "Total HTTP requests by route, method, and status code.",
			},
			[]string{"route", "method", "status"},
		),
		httpDurationSeconds: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "signaling_http_request_duration_seconds",
				Help:    "HTTP request latency in seconds.",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"route", "method"},
		),
		wsConnectionsCurrent: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "signaling_ws_connections_current",
				Help: "Current number of active websocket connections.",
			},
		),
		wsMessagesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "signaling_ws_messages_total",
				Help: "Total websocket messages by direction and type.",
			},
			[]string{"direction", "type"},
		),
	}

	registry.MustRegister(
		m.httpRequestsTotal,
		m.httpDurationSeconds,
		m.wsConnectionsCurrent,
		m.wsMessagesTotal,
	)
	return m
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapped := &statusCapturingWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(wrapped, r)

		route := r.URL.Path
		method := r.Method
		status := strconv.Itoa(wrapped.statusCode)
		m.httpRequestsTotal.WithLabelValues(route, method, status).Inc()
		m.httpDurationSeconds.WithLabelValues(route, method).Observe(time.Since(start).Seconds())
	})
}

func (m *Metrics) WSConnectionOpened() {
	m.wsConnectionsCurrent.Inc()
}

func (m *Metrics) WSConnectionClosed() {
	m.wsConnectionsCurrent.Dec()
}

func (m *Metrics) WSMessageIn(messageType string) {
	m.wsMessagesTotal.WithLabelValues("in", messageType).Inc()
}

func (m *Metrics) WSMessageOut(messageType string) {
	m.wsMessagesTotal.WithLabelValues("out", messageType).Inc()
}

type statusCapturingWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusCapturingWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusCapturingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (w *statusCapturingWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *statusCapturingWriter) Push(target string, opts *http.PushOptions) error {
	pusher, ok := w.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

