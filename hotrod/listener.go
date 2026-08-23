package hotrod

import (
	"context"
	"crypto/rand"

	"infinispan.org/go-client/internal/operation"
)

// EventType identifies the kind of cache entry event.
type EventType = operation.EventType

const (
	EventCreated  EventType = operation.EventCreated
	EventModified EventType = operation.EventModified
	EventRemoved  EventType = operation.EventRemoved
	EventExpired  EventType = operation.EventExpired
)

// CacheEntryEvent represents a server-side cache entry event (created, modified, removed, or expired).
type CacheEntryEvent = operation.CacheEntryEvent

// CacheListener receives cache entry events on its Events channel.
type CacheListener struct {
	Events <-chan *CacheEntryEvent
	id     []byte
	ch     chan *CacheEntryEvent
}

// ListenerOption configures an AddListener operation.
type ListenerOption func(*listenerConfig)

type listenerConfig struct {
	includeCurrentState bool
	interests           byte
	channelSize         int
}

// WithIncludeCurrentState causes the listener to receive events for all existing entries upon registration.
func WithIncludeCurrentState() ListenerOption {
	return func(c *listenerConfig) {
		c.includeCurrentState = true
	}
}

// WithListenerInterests restricts the listener to only the specified event types.
func WithListenerInterests(types ...EventType) ListenerOption {
	return func(c *listenerConfig) {
		c.interests = 0
		for _, t := range types {
			switch t {
			case EventCreated:
				c.interests |= operation.InterestCreated
			case EventModified:
				c.interests |= operation.InterestModified
			case EventRemoved:
				c.interests |= operation.InterestRemoved
			case EventExpired:
				c.interests |= operation.InterestExpired
			}
		}
	}
}

// WithChannelSize sets the buffer size of the event channel.
func WithChannelSize(n int) ListenerOption {
	return func(c *listenerConfig) {
		c.channelSize = n
	}
}

// AddListener registers a cache event listener and returns a CacheListener whose Events channel receives events.
func (rc *RemoteCache) AddListener(ctx context.Context, opts ...ListenerOption) (*CacheListener, error) {
	cfg := &listenerConfig{
		channelSize: 64,
		interests:   operation.InterestAll,
	}
	for _, o := range opts {
		o(cfg)
	}

	listenerID := make([]byte, 16)
	if _, err := rand.Read(listenerID); err != nil {
		return nil, err
	}

	ch := make(chan *CacheEntryEvent, cfg.channelSize)

	op := &operation.AddClientListenerOp{
		Cache:        rc.name,
		ListenerID:   listenerID,
		IncludeState: cfg.includeCurrentState,
		Interests:    cfg.interests,
	}

	if err := rc.client.pool.AddListener(ctx, op, ch); err != nil {
		return nil, err
	}

	return &CacheListener{
		Events: ch,
		id:     listenerID,
		ch:     ch,
	}, nil
}

// RemoveListener unregisters a previously added cache event listener.
func (rc *RemoteCache) RemoveListener(ctx context.Context, listener *CacheListener) error {
	op := &operation.RemoveClientListenerOp{
		Cache:      rc.name,
		ListenerID: listener.id,
	}
	return rc.client.pool.RemoveListener(ctx, op)
}
