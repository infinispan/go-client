package hotrod

import (
	"context"
	"fmt"
	"log/slog"

	"google.golang.org/protobuf/proto"
)

// TypedCache is a type-safe cache that automatically marshals keys and values
// using the configured Marshaller.
type TypedCache[K any, V any] struct {
	raw        *RemoteCache
	marshaller Marshaller
	newV       func() V
	logger     *slog.Logger
}

// NewTypedCache creates a typed cache with the given marshaller.
// newV is a factory function that creates a new zero-value V for unmarshalling.
// For proto.Message types, this is typically: func() *Person { return &Person{} }
func NewTypedCache[K any, V any](client *Client, cacheName string, m Marshaller, newV func() V) *TypedCache[K, V] {
	return &TypedCache[K, V]{
		raw:        client.Cache(cacheName),
		marshaller: m,
		newV:       newV,
		logger:     client.logger,
	}
}

// Put marshals and stores a key-value pair in the cache.
func (tc *TypedCache[K, V]) Put(ctx context.Context, key K, value V, opts ...PutOption) error {
	kb, err := tc.marshaller.MarshalKey(key)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}
	vb, err := tc.marshaller.MarshalValue(value)
	if err != nil {
		return fmt.Errorf("marshal value: %w", err)
	}
	return tc.raw.PutRaw(ctx, kb, vb, tc.marshaller.MediaType(), opts...)
}

// Remove marshals the key and deletes the entry from the cache.
func (tc *TypedCache[K, V]) Remove(ctx context.Context, key K, opts ...RemoveOption) error {
	kb, err := tc.marshaller.MarshalKey(key)
	if err != nil {
		return fmt.Errorf("marshal key: %w", err)
	}
	return tc.raw.Remove(ctx, kb, append([]RemoveOption{WithRemoveMediaType(tc.marshaller.MediaType())}, opts...)...)
}

// Get retrieves and unmarshals the value for a key.
func (tc *TypedCache[K, V]) Get(ctx context.Context, key K) (V, bool, error) {
	kb, err := tc.marshaller.MarshalKey(key)
	if err != nil {
		var zero V
		return zero, false, fmt.Errorf("marshal key: %w", err)
	}
	data, found, err := tc.raw.GetRaw(ctx, kb, tc.marshaller.MediaType())
	if err != nil || !found {
		var zero V
		return zero, found, err
	}
	v := tc.newV()
	if err := tc.marshaller.UnmarshalValue(data, v); err != nil {
		var zero V
		return zero, false, fmt.Errorf("unmarshal value: %w", err)
	}
	return v, true, nil
}

// Query executes an Ickle query and returns typed entity results.
// Query result values are already unwrapped, so proto.Unmarshal is used directly.
func (tc *TypedCache[K, V]) Query(ctx context.Context, ickle string, opts ...QueryOption) ([]V, error) {
	raw, err := tc.raw.Query(ctx, ickle, opts...)
	if err != nil {
		return nil, err
	}
	results := make([]V, 0, raw.NumResults)
	for _, e := range raw.Entries {
		if e.Value == nil {
			continue
		}
		v := tc.newV()
		msg, ok := any(v).(proto.Message)
		if !ok {
			return nil, fmt.Errorf("value type %T does not implement proto.Message", v)
		}
		if err := proto.Unmarshal(e.Value, msg); err != nil {
			return nil, fmt.Errorf("unmarshal query result: %w", err)
		}
		results = append(results, v)
	}
	return results, nil
}

// QueryProjection executes an Ickle projection query and returns the raw QueryResult.
func (tc *TypedCache[K, V]) QueryProjection(ctx context.Context, ickle string, opts ...QueryOption) (*QueryResult, error) {
	return tc.raw.Query(ctx, ickle, opts...)
}

// TypedCQEvent represents a typed continuous query event.
type TypedCQEvent[K any, V any] struct {
	Type  CQResultType
	Key   K
	Value V
}

// TypedContinuousQuery wraps a raw ContinuousQuery and delivers typed events.
type TypedContinuousQuery[K any, V any] struct {
	Events <-chan *TypedCQEvent[K, V]
	inner  *ContinuousQuery
	evCh   chan *TypedCQEvent[K, V]
	done   chan struct{}
	logger *slog.Logger
}

// ContinuousQuery registers a typed continuous query.
func (tc *TypedCache[K, V]) ContinuousQuery(ctx context.Context, query string, opts ...CQOption) (*TypedContinuousQuery[K, V], error) {
	inner, err := tc.raw.ContinuousQuery(ctx, query, opts...)
	if err != nil {
		return nil, err
	}
	evCh := make(chan *TypedCQEvent[K, V], 64)
	done := make(chan struct{})
	tcq := &TypedContinuousQuery[K, V]{
		Events: evCh,
		inner:  inner,
		evCh:   evCh,
		done:   done,
		logger: tc.logger,
	}
	go tcq.decodeLoop(tc.marshaller, tc.newV)
	return tcq, nil
}

// RemoveContinuousQuery unregisters a typed continuous query.
func (tc *TypedCache[K, V]) RemoveContinuousQuery(ctx context.Context, cq *TypedContinuousQuery[K, V]) error {
	close(cq.done)
	return tc.raw.RemoveContinuousQuery(ctx, cq.inner)
}

func (tcq *TypedContinuousQuery[K, V]) decodeLoop(m Marshaller, newV func() V) {
	defer close(tcq.evCh)
	for {
		select {
		case raw, ok := <-tcq.inner.Events:
			if !ok {
				return
			}
			ev := &TypedCQEvent[K, V]{Type: raw.Type}
			if raw.Key != nil {
				var k K
				if err := m.UnmarshalKey(raw.Key, &k); err != nil {
					tcq.logger.Warn("unmarshal typed CQ key", "err", err)
				} else {
					ev.Key = k
				}
			}
			if raw.Value != nil {
				v := newV()
				if err := m.UnmarshalValue(raw.Value, v); err != nil {
					tcq.logger.Warn("unmarshal typed CQ value", "err", err)
				} else {
					ev.Value = v
				}
			}
			select {
			case tcq.evCh <- ev:
			case <-tcq.done:
				return
			}
		case <-tcq.done:
			return
		}
	}
}
