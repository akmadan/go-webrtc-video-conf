package websocket

import (
	"sync"
	"time"
)

// MessageLimiter is a simple fixed-window limiter for inbound WS messages.
type MessageLimiter struct {
	mu         sync.Mutex
	maxPerSec  int
	windowFrom time.Time
	count      int
}

func NewMessageLimiter(maxPerSec int) *MessageLimiter {
	if maxPerSec <= 0 {
		maxPerSec = 30
	}
	return &MessageLimiter{
		maxPerSec:  maxPerSec,
		windowFrom: time.Now(),
	}
}

func (l *MessageLimiter) Allow() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.windowFrom) >= time.Second {
		l.windowFrom = now
		l.count = 0
	}
	if l.count >= l.maxPerSec {
		return false
	}
	l.count++
	return true
}

