package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestExpiration_Lifespan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-lifespan")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put with 2 second lifespan
	err := cache.Put(ctx, []byte("key1"), []byte("value1"), hotrod.WithLifespan(2*time.Second))
	if err != nil {
		t.Fatalf("Put with lifespan: %v", err)
	}

	// Should exist immediately
	val, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found before expiry")
	}
	if string(val) != "value1" {
		t.Errorf("got %q, want %q", string(val), "value1")
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Should not exist after expiration
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected key not to be found after expiry")
	}
}

func TestExpiration_MaxIdle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-maxidle")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put with 3 second max idle time
	err := cache.Put(ctx, []byte("key1"), []byte("value1"), hotrod.WithMaxIdle(3*time.Second))
	if err != nil {
		t.Fatalf("Put with maxIdle: %v", err)
	}

	// Access within idle time (at 1 second)
	time.Sleep(1 * time.Second)
	_, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get within idle time: %v", err)
	}
	if !found {
		t.Error("expected key to be found within idle time")
	}

	// Access again within idle time (at 2 seconds from last access)
	time.Sleep(1 * time.Second)
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get within idle time (2): %v", err)
	}
	if !found {
		t.Error("expected key to be found within idle time")
	}

	// Wait for idle timeout (4 seconds without access)
	time.Sleep(4 * time.Second)

	// Should not exist after idle timeout
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after idle timeout: %v", err)
	}
	if found {
		t.Error("expected key not to be found after idle timeout")
	}
}

func TestExpiration_LifespanAndMaxIdle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-both")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put with both lifespan (5s) and maxIdle (2s)
	err := cache.Put(ctx, []byte("key1"), []byte("value1"),
		hotrod.WithLifespan(5*time.Second),
		hotrod.WithMaxIdle(2*time.Second))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Should exist immediately
	_, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}

	// Wait for maxIdle to expire (3 seconds without access)
	time.Sleep(3 * time.Second)

	// Should not exist (maxIdle expired before lifespan)
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after maxIdle: %v", err)
	}
	if found {
		t.Error("expected key to expire due to maxIdle")
	}
}

func TestExpiration_MillisecondPrecision(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-millis")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put with 1500ms lifespan
	err := cache.Put(ctx, []byte("key1"), []byte("value1"), hotrod.WithLifespan(1500*time.Millisecond))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Should exist after 1 second
	time.Sleep(1 * time.Second)
	_, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get at 1s: %v", err)
	}
	if !found {
		t.Error("expected key to exist at 1 second")
	}

	// Should not exist after 2 seconds
	time.Sleep(1100 * time.Millisecond)
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get at 2s: %v", err)
	}
	if found {
		t.Error("expected key to be expired at 2 seconds")
	}
}

func TestExpiration_PutAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-putall")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// PutAll with expiration
	entries := map[string][]byte{
		"k1": []byte("v1"),
		"k2": []byte("v2"),
		"k3": []byte("v3"),
	}

	err := cache.PutAll(ctx, entries, hotrod.WithLifespan(2*time.Second))
	if err != nil {
		t.Fatalf("PutAll: %v", err)
	}

	// All should exist immediately
	result, err := cache.GetAll(ctx, []string{"k1", "k2", "k3"})
	if err != nil {
		t.Fatalf("GetAll before expiry: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// None should exist after expiration
	result, err = cache.GetAll(ctx, []string{"k1", "k2", "k3"})
	if err != nil {
		t.Fatalf("GetAll after expiry: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries after expiry, got %d", len(result))
	}
}

func TestExpiration_Replace(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-replace")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put initial value without expiration
	err := cache.Put(ctx, []byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Replace with expiration
	replaced, err := cache.Replace(ctx, []byte("key1"), []byte("value2"), hotrod.WithLifespan(2*time.Second))
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if !replaced {
		t.Fatal("expected replace to succeed")
	}

	// Should exist with new value
	val, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if string(val) != "value2" {
		t.Errorf("got %q, want %q", string(val), "value2")
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Should not exist
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected key not to be found after expiry")
	}
}

func TestExpiration_PutIfAbsent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-putifabsent")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// PutIfAbsent with expiration
	created, err := cache.PutIfAbsent(ctx, []byte("key1"), []byte("value1"), hotrod.WithLifespan(2*time.Second))
	if err != nil {
		t.Fatalf("PutIfAbsent: %v", err)
	}
	if !created {
		t.Fatal("expected PutIfAbsent to create entry")
	}

	// Should exist
	_, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Should not exist
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected key not to be found after expiry")
	}
}

func TestExpiration_ZeroMeansNoExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-zero")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put with zero lifespan (should not expire)
	err := cache.Put(ctx, []byte("key1"), []byte("value1"), hotrod.WithLifespan(0))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Wait some time
	time.Sleep(2 * time.Second)

	// Should still exist
	_, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Error("expected key to still exist (zero lifespan = no expiry)")
	}
}

func TestExpiration_MixedLifespan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-mixed")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put entries with different lifespans
	err := cache.Put(ctx, []byte("short"), []byte("v1"), hotrod.WithLifespan(2*time.Second))
	if err != nil {
		t.Fatalf("Put short: %v", err)
	}

	err = cache.Put(ctx, []byte("long"), []byte("v2"), hotrod.WithLifespan(10*time.Second))
	if err != nil {
		t.Fatalf("Put long: %v", err)
	}

	err = cache.Put(ctx, []byte("none"), []byte("v3"))
	if err != nil {
		t.Fatalf("Put none: %v", err)
	}

	// All should exist initially
	result, err := cache.GetAll(ctx, []string{"short", "long", "none"})
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(result))
	}

	// Wait for short to expire (3 seconds)
	time.Sleep(3 * time.Second)

	// Short should be gone, long and none should remain
	result, err = cache.GetAll(ctx, []string{"short", "long", "none"})
	if err != nil {
		t.Fatalf("GetAll after short expiry: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 entries after short expiry, got %d", len(result))
	}
	if _, ok := result["short"]; ok {
		t.Error("expected 'short' to be expired")
	}
	if _, ok := result["long"]; !ok {
		t.Error("expected 'long' to still exist")
	}
	if _, ok := result["none"]; !ok {
		t.Error("expected 'none' to still exist")
	}
}

func TestExpiration_Size(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-size")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put entries with expiration
	entries := map[string][]byte{
		"k1": []byte("v1"),
		"k2": []byte("v2"),
		"k3": []byte("v3"),
	}

	err := cache.PutAll(ctx, entries, hotrod.WithLifespan(2*time.Second))
	if err != nil {
		t.Fatalf("PutAll: %v", err)
	}

	// Size should be 3
	size, err := cache.Size(ctx)
	if err != nil {
		t.Fatalf("Size before expiry: %v", err)
	}
	if size != 3 {
		t.Errorf("expected size 3, got %d", size)
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Size should be 0
	size, err = cache.Size(ctx)
	if err != nil {
		t.Fatalf("Size after expiry: %v", err)
	}
	if size != 0 {
		t.Errorf("expected size 0 after expiry, got %d", size)
	}
}

func TestExpiration_ContainsKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-contains")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put with expiration
	err := cache.Put(ctx, []byte("key1"), []byte("value1"), hotrod.WithLifespan(2*time.Second))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Should contain key
	exists, err := cache.ContainsKey(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("ContainsKey before expiry: %v", err)
	}
	if !exists {
		t.Error("expected key to exist before expiry")
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Should not contain key
	exists, err = cache.ContainsKey(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("ContainsKey after expiry: %v", err)
	}
	if exists {
		t.Error("expected key not to exist after expiry")
	}
}

func TestExpiration_CacheLevelDefaultLifespan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-default-lifespan")

	// Create cache with default lifespan of 3 seconds (maxIdle = -1 means no max idle)
	createCacheWithExpiration(t, sharedContainer, cacheName, 3, -1)

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	connCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(connCtx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache(cacheName)
	ctx := context.Background()

	// Put entry WITHOUT explicit expiration - should use cache default lifespan of 3s
	err = cache.Put(ctx, []byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Should exist immediately
	val, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found before expiry")
	}
	if string(val) != "value1" {
		t.Errorf("got %q, want %q", string(val), "value1")
	}

	// Wait for cache default lifespan to expire (3 seconds + buffer)
	time.Sleep(4 * time.Second)

	// Should not exist after cache default lifespan
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected key not to be found after cache default lifespan")
	}

	// Test that explicit expiration overrides cache default
	err = cache.Put(ctx, []byte("key2"), []byte("value2"), hotrod.WithLifespan(10*time.Second))
	if err != nil {
		t.Fatalf("Put with explicit lifespan: %v", err)
	}

	// After 4 seconds, key2 should still exist (has 10s lifespan)
	time.Sleep(4 * time.Second)
	val, found, err = cache.Get(ctx, []byte("key2"))
	if err != nil {
		t.Fatalf("Get key2: %v", err)
	}
	if !found {
		t.Error("expected key2 to still exist (explicit lifespan overrides default)")
	}
	if string(val) != "value2" {
		t.Errorf("got %q, want %q", string(val), "value2")
	}
}

func TestExpiration_CacheLevelDefaultMaxIdle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "expire-default-maxidle")

	// Create cache with default maxIdle of 3 seconds (lifespan = -1 means no lifespan)
	createCacheWithExpiration(t, sharedContainer, cacheName, -1, 3)

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	connCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(connCtx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache(cacheName)
	ctx := context.Background()

	// Put entry WITHOUT explicit expiration - should use cache default maxIdle of 3s
	err = cache.Put(ctx, []byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Access within idle time (at 1 second) - should reset maxIdle timer
	time.Sleep(1 * time.Second)
	val, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get within idle time: %v", err)
	}
	if !found {
		t.Error("expected key to be found within idle time")
	}
	if string(val) != "value1" {
		t.Errorf("got %q, want %q", string(val), "value1")
	}

	// Access again at 2 seconds from last access - should still exist and reset timer
	time.Sleep(2 * time.Second)
	val, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get within idle time (2): %v", err)
	}
	if !found {
		t.Error("expected key to be found within idle time")
	}
	if string(val) != "value1" {
		t.Errorf("got %q, want %q", string(val), "value1")
	}

	// Wait for cache default maxIdle timeout (4 seconds without access)
	time.Sleep(4 * time.Second)

	// Should not exist after idle timeout
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after idle timeout: %v", err)
	}
	if found {
		t.Error("expected key not to be found after cache default maxIdle timeout")
	}

	// Test that explicit maxIdle overrides cache default
	err = cache.Put(ctx, []byte("key2"), []byte("value2"), hotrod.WithMaxIdle(10*time.Second))
	if err != nil {
		t.Fatalf("Put with explicit maxIdle: %v", err)
	}

	// After 4 seconds, key2 should still exist (has 10s maxIdle)
	time.Sleep(4 * time.Second)
	val, found, err = cache.Get(ctx, []byte("key2"))
	if err != nil {
		t.Fatalf("Get key2: %v", err)
	}
	if !found {
		t.Error("expected key2 to still exist (explicit maxIdle overrides default)")
	}
	if string(val) != "value2" {
		t.Errorf("got %q, want %q", string(val), "value2")
	}
}
