package hotrod_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestIterator_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "iter-empty")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	it, err := client.Cache("iter-empty").Iterator(ctx)
	if err != nil {
		t.Fatalf("Iterator: %v", err)
	}
	defer it.Close()

	if it.Next() {
		t.Fatal("expected no entries from empty cache")
	}
	if err := it.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}
}

func TestIterator_AllEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "iter-all")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("iter-all")
	const n = 50
	for i := 0; i < n; i++ {
		k := fmt.Sprintf("key-%03d", i)
		v := fmt.Sprintf("val-%03d", i)
		if err := cache.Put(ctx, []byte(k), []byte(v)); err != nil {
			t.Fatalf("Put %s: %v", k, err)
		}
	}

	it, err := cache.Iterator(ctx)
	if err != nil {
		t.Fatalf("Iterator: %v", err)
	}
	defer it.Close()

	got := make(map[string]string)
	for it.Next() {
		k, v := it.Entry()
		got[string(k)] = string(v)
	}
	if err := it.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}

	if len(got) != n {
		t.Fatalf("got %d entries, want %d", len(got), n)
	}
	for i := 0; i < n; i++ {
		k := fmt.Sprintf("key-%03d", i)
		v := fmt.Sprintf("val-%03d", i)
		if got[k] != v {
			t.Errorf("entry %s = %q, want %q", k, got[k], v)
		}
	}
}

func TestIterator_WithMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "iter-meta")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("iter-meta")
	if err := cache.Put(ctx, []byte("k1"), []byte("v1"), hotrod.WithLifespan(300*time.Second)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	it, err := cache.Iterator(ctx, hotrod.WithIteratorMetadata())
	if err != nil {
		t.Fatalf("Iterator: %v", err)
	}
	defer it.Close()

	if !it.Next() {
		t.Fatal("expected at least one entry")
	}

	md := it.Metadata()
	if md == nil {
		t.Fatal("expected metadata to be non-nil")
	}
	if md.Version == 0 {
		t.Error("expected non-zero version")
	}
	if md.Lifespan <= 0 {
		t.Errorf("expected positive lifespan, got %d", md.Lifespan)
	}
	t.Logf("metadata: version=%d lifespan=%d created=%d", md.Version, md.Lifespan, md.Created)
}

func TestIterator_CloseEarly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "iter-close")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("iter-close")
	for i := 0; i < 20; i++ {
		if err := cache.Put(ctx, []byte(fmt.Sprintf("k%d", i)), []byte("v")); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	it, err := cache.Iterator(ctx, hotrod.WithIteratorBatchSize(5))
	if err != nil {
		t.Fatalf("Iterator: %v", err)
	}

	it.Next()
	if err := it.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := it.Close(); err != nil {
		t.Fatal("second Close should be no-op")
	}
}

func TestIterator_BatchSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "iter-batch")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("iter-batch")
	const n = 25
	for i := 0; i < n; i++ {
		if err := cache.Put(ctx, []byte(fmt.Sprintf("k%02d", i)), []byte(fmt.Sprintf("v%02d", i))); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	it, err := cache.Iterator(ctx, hotrod.WithIteratorBatchSize(5))
	if err != nil {
		t.Fatalf("Iterator: %v", err)
	}
	defer it.Close()

	var keys []string
	for it.Next() {
		k, _ := it.Entry()
		keys = append(keys, string(k))
	}
	if err := it.Err(); err != nil {
		t.Fatalf("Err: %v", err)
	}

	sort.Strings(keys)
	if len(keys) != n {
		t.Fatalf("got %d entries, want %d", len(keys), n)
	}
	for i := 0; i < n; i++ {
		expected := fmt.Sprintf("k%02d", i)
		if keys[i] != expected {
			t.Errorf("keys[%d] = %q, want %q", i, keys[i], expected)
		}
	}
}
