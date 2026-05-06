package hotrod

import (
	"context"
	"fmt"
)

// TypedCache is a type-safe cache that automatically marshals keys and values
// using the configured Marshaller.
type TypedCache[K any, V any] struct {
	raw        *RemoteCache
	marshaller Marshaller
	newV       func() V
}

// NewTypedCache creates a typed cache with the given marshaller.
// newV is a factory function that creates a new zero-value V for unmarshalling.
// For proto.Message types, this is typically: func() *Person { return &Person{} }
func NewTypedCache[K any, V any](client *Client, cacheName string, m Marshaller, newV func() V) *TypedCache[K, V] {
	return &TypedCache[K, V]{
		raw:        client.Cache(cacheName),
		marshaller: m,
		newV:       newV,
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
