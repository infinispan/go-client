package hotrod_test

import (
	"context"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestPutAllGetAll_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "bulk-basic")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	entries := map[string][]byte{
		"k1": []byte("v1"),
		"k2": []byte("v2"),
		"k3": []byte("v3"),
		"k4": []byte("v4"),
		"k5": []byte("v5"),
	}

	if err := cache.PutAll(ctx, entries); err != nil {
		t.Fatalf("PutAll: %v", err)
	}

	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}

	result, err := cache.GetAll(ctx, keys)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}

	if len(result) != len(entries) {
		t.Fatalf("GetAll returned %d entries, want %d", len(result), len(entries))
	}

	for k, want := range entries {
		got, ok := result[k]
		if !ok {
			t.Errorf("missing key %q", k)
			continue
		}
		if string(got) != string(want) {
			t.Errorf("key %q: got %q, want %q", k, string(got), string(want))
		}
	}
}

func TestPutAllGetAll_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "bulk-empty")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	if err := cache.PutAll(ctx, map[string][]byte{}); err != nil {
		t.Fatalf("PutAll empty: %v", err)
	}

	result, err := cache.GetAll(ctx, []string{})
	if err != nil {
		t.Fatalf("GetAll empty: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestPutAll_WithExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "bulk-ttl")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	entries := map[string][]byte{
		"e1": []byte("val1"),
		"e2": []byte("val2"),
	}

	if err := cache.PutAll(ctx, entries, hotrod.WithLifespan(1*time.Second)); err != nil {
		t.Fatalf("PutAll with lifespan: %v", err)
	}

	result, err := cache.GetAll(ctx, []string{"e1", "e2"})
	if err != nil {
		t.Fatalf("GetAll before expiry: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries before expiry, got %d", len(result))
	}

	time.Sleep(2 * time.Second)

	result, err = cache.GetAll(ctx, []string{"e1", "e2"})
	if err != nil {
		t.Fatalf("GetAll after expiry: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries after expiry, got %d", len(result))
	}
}
