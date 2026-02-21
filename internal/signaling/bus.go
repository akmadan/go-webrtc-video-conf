package signaling

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/redis/go-redis/v9"
)

type EventType string

const (
	EventPeerJoined EventType = "peer_joined"
	EventPeerLeft   EventType = "peer_left"
	EventSignaling  EventType = "signaling_message"
)

type Event struct {
	Type        EventType       `json:"type"`
	SourceID    string          `json:"sourceId"`
	RoomID      string          `json:"roomId"`
	PeerID      string          `json:"peerId,omitempty"`
	TargetID    string          `json:"targetId,omitempty"`
	MessageType string          `json:"messageType,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
	Timestamp   int64           `json:"timestamp"`
}

// EventBus is an abstraction for propagating signaling events across processes.
// Today we use a local no-op publisher. A Redis-backed implementation can satisfy
// the same interface without touching websocket hub logic.
type EventBus interface {
	Publish(ctx context.Context, event Event) error
	Subscribe(ctx context.Context, handler func(Event)) error
	Close() error
}

type NoopBus struct{}

func NewNoopBus() *NoopBus {
	return &NoopBus{}
}

func (b *NoopBus) Publish(_ context.Context, _ Event) error {
	return nil
}

func (b *NoopBus) Subscribe(_ context.Context, _ func(Event)) error {
	return nil
}

func (b *NoopBus) Close() error {
	return nil
}

type LogBus struct{}

func NewLogBus() *LogBus {
	return &LogBus{}
}

func (b *LogBus) Publish(_ context.Context, event Event) error {
	slog.Debug("signaling event",
		"type", event.Type,
		"source_id", event.SourceID,
		"room_id", event.RoomID,
		"peer_id", event.PeerID,
		"target_id", event.TargetID,
		"message_type", event.MessageType,
	)
	return nil
}

func (b *LogBus) Subscribe(_ context.Context, _ func(Event)) error {
	return nil
}

func (b *LogBus) Close() error {
	return nil
}

type RedisBus struct {
	client  *redis.Client
	channel string
}

func NewRedisBus(addr, password string, db int, channel string) (*RedisBus, error) {
	if channel == "" {
		return nil, errors.New("redis channel is required")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return &RedisBus{
		client:  client,
		channel: channel,
	}, nil
}

func (b *RedisBus) Publish(ctx context.Context, event Event) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return b.client.Publish(ctx, b.channel, payload).Err()
}

func (b *RedisBus) Subscribe(ctx context.Context, handler func(Event)) error {
	pubsub := b.client.Subscribe(ctx, b.channel)
	if _, err := pubsub.Receive(ctx); err != nil {
		return err
	}

	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var event Event
				if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
					slog.Warn("failed to decode redis signaling event", "error", err.Error())
					continue
				}
				handler(event)
			}
		}
	}()

	return nil
}

func (b *RedisBus) Close() error {
	if b.client == nil {
		return nil
	}
	return b.client.Close()
}

