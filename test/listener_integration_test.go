package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestListener_CreatedModifiedRemoved(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "listener-test")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("listener-test")

	listener, err := cache.AddListener(ctx)
	if err != nil {
		t.Fatalf("AddListener: %v", err)
	}
	defer func() { _ = cache.RemoveListener(ctx, listener) }()

	// Put → Created event
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	ev := expectEvent(t, listener, 5*time.Second)
	if ev.Type != hotrod.EventCreated {
		t.Errorf("expected EventCreated, got %v", ev.Type)
	}
	if string(ev.Key) != "k1" {
		t.Errorf("key = %q, want %q", ev.Key, "k1")
	}

	// Replace → Modified event
	if err := cache.Put(ctx, []byte("k1"), []byte("v2")); err != nil {
		t.Fatalf("Put (modify): %v", err)
	}
	ev = expectEvent(t, listener, 5*time.Second)
	if ev.Type != hotrod.EventModified {
		t.Errorf("expected EventModified, got %v", ev.Type)
	}
	if string(ev.Key) != "k1" {
		t.Errorf("key = %q, want %q", ev.Key, "k1")
	}

	// Remove → Removed event
	if err := cache.Remove(ctx, []byte("k1")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	ev = expectEvent(t, listener, 5*time.Second)
	if ev.Type != hotrod.EventRemoved {
		t.Errorf("expected EventRemoved, got %v", ev.Type)
	}
	if string(ev.Key) != "k1" {
		t.Errorf("key = %q, want %q", ev.Key, "k1")
	}
}

func TestListener_FilterByInterest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "listener-interest")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("listener-interest")

	// Only listen for created events
	listener, err := cache.AddListener(ctx, hotrod.WithListenerInterests(hotrod.EventCreated))
	if err != nil {
		t.Fatalf("AddListener: %v", err)
	}
	defer func() { _ = cache.RemoveListener(ctx, listener) }()

	// Put → should receive created event
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	ev := expectEvent(t, listener, 5*time.Second)
	if ev.Type != hotrod.EventCreated {
		t.Errorf("expected EventCreated, got %v", ev.Type)
	}

	// Modify and remove → should NOT receive events
	if err := cache.Put(ctx, []byte("k1"), []byte("v2")); err != nil {
		t.Fatalf("Put (modify): %v", err)
	}
	if err := cache.Remove(ctx, []byte("k1")); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	expectNoEvent(t, listener, 2*time.Second)
}

func TestListener_RemoveStopsEvents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "listener-remove")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("listener-remove")

	listener, err := cache.AddListener(ctx)
	if err != nil {
		t.Fatalf("AddListener: %v", err)
	}

	// Verify listener is active
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	expectEvent(t, listener, 5*time.Second)

	// Remove listener
	if err := cache.RemoveListener(ctx, listener); err != nil {
		t.Fatalf("RemoveListener: %v", err)
	}

	// Put again — no events should arrive
	if err := cache.Put(ctx, []byte("k2"), []byte("v2")); err != nil {
		t.Fatalf("Put after remove: %v", err)
	}
	expectNoEvent(t, listener, 2*time.Second)
}

func expectEvent(t *testing.T, listener *hotrod.CacheListener, timeout time.Duration) *hotrod.CacheEntryEvent {
	t.Helper()
	select {
	case ev, ok := <-listener.Events:
		if !ok {
			t.Fatal("listener channel closed unexpectedly")
		}
		return ev
	case <-time.After(timeout):
		t.Fatal("timed out waiting for event")
		return nil
	}
}

func expectNoEvent(t *testing.T, listener *hotrod.CacheListener, wait time.Duration) {
	t.Helper()
	select {
	case ev := <-listener.Events:
		t.Fatalf("expected no event, got %v for key %q", ev.Type, ev.Key)
	case <-time.After(wait):
		// good, no event
	}
}
