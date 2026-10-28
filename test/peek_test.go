package hotrod_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestPeek_ReturnsValueAndMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "peek-metadata")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	err := cache.Put(ctx, []byte("key"), []byte("value"), hotrod.WithLifespan(10*time.Minute), hotrod.WithMaxIdle(5*time.Minute))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	mv, found, err := cache.Peek(ctx, []byte("key"))
	if err != nil {
		t.Fatalf("Peek: %v", err)
	}
	if !found || mv == nil {
		t.Fatal("expected entry to be found")
	}
	if !bytes.Equal(mv.Value, []byte("value")) {
		t.Errorf("unexpected value: got %q want %q", mv.Value, "value")
	}
	if mv.Lifespan != 600 {
		t.Errorf("unexpected lifespan: got %d want 600", mv.Lifespan)
	}
	if mv.MaxIdle != 300 {
		t.Errorf("unexpected max idle: got %d want 300", mv.MaxIdle)
	}
}

func TestPeek_NonExistentKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "peek-missing")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mv, found, err := cache.Peek(ctx, []byte("no-such-key"))
	if err != nil {
		t.Fatalf("Peek: %v", err)
	}
	if found || mv != nil {
		t.Error("expected no entry for missing key")
	}
}

func TestPeek_DoesNotResetMaxIdle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "peek-idle")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	idle := 2 * time.Second
	err := cache.Put(ctx, []byte("peeked"), []byte("value"), hotrod.WithMaxIdle(idle))
	if err != nil {
		t.Fatalf("Put peeked: %v", err)
	}
	err = cache.Put(ctx, []byte("accessed"), []byte("value"), hotrod.WithMaxIdle(idle))
	if err != nil {
		t.Fatalf("Put accessed: %v", err)
	}

	time.Sleep(time.Second)

	mv, found, err := cache.Peek(ctx, []byte("peeked"))
	if err != nil {
		t.Fatalf("Peek peeked: %v", err)
	}
	if !found || mv == nil {
		t.Fatal("expected entry to be found before idle expiration")
	}

	val, found, err := cache.Get(ctx, []byte("accessed"))
	if err != nil {
		t.Fatalf("Get accessed: %v", err)
	}
	if !found || !bytes.Equal(val, []byte("value")) {
		t.Fatal("expected entry to be found before idle expiration")
	}

	time.Sleep(1600 * time.Millisecond)

	val, found, err = cache.Get(ctx, []byte("peeked"))
	if err != nil {
		t.Fatalf("Get peeked: %v", err)
	}
	if found {
		t.Error("expected entry to have expired since Peek does not reset max-idle")
	}

	val, found, err = cache.Get(ctx, []byte("accessed"))
	if err != nil {
		t.Fatalf("Get accessed: %v", err)
	}
	if !found || !bytes.Equal(val, []byte("value")) {
		t.Error("expected entry to still exist since Get reset max-idle")
	}
}
