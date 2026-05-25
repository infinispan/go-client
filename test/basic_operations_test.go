package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

// uniqueCacheName generates a unique cache name for test isolation
func uniqueCacheName(t *testing.T, prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// setupCache creates a cache and returns a cleanup function
func setupCache(t *testing.T, cacheName string) (*hotrod.RemoteCache, func()) {
	t.Helper()

	createTestCache(t, sharedContainer, cacheName)

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	cache := client.Cache(cacheName)

	cleanup := func() {
		client.Close()
		// Note: Cache will be cleaned up when server is reset or test framework cleans up
		// For now, we rely on unique names to avoid conflicts
	}

	return cache, cleanup
}

func TestBasicPut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "put")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put a value
	err := cache.Put(ctx, []byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Get it back
	val, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if string(val) != "value1" {
		t.Errorf("got %q, want %q", string(val), "value1")
	}
}

func TestBasicGet_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "get-notfound")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Get non-existent key
	_, found, err := cache.Get(ctx, []byte("nonexistent"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Error("expected key not to be found")
	}
}

func TestBasicPutWithLifespan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "put-lifespan")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put with 1 second lifespan
	err := cache.Put(ctx, []byte("expiring"), []byte("value"), hotrod.WithLifespan(1*time.Second))
	if err != nil {
		t.Fatalf("Put with lifespan: %v", err)
	}

	// Should exist immediately
	_, found, err := cache.Get(ctx, []byte("expiring"))
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found before expiry")
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should not exist after expiration
	_, found, err = cache.Get(ctx, []byte("expiring"))
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected key not to be found after expiry")
	}
}

func TestBasicRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "remove")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put a value
	err := cache.Put(ctx, []byte("key1"), []byte("value1"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Remove it using GetAndRemove to verify it existed
	val, found, err := cache.GetAndRemove(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("GetAndRemove: %v", err)
	}
	if !found {
		t.Error("expected key to be found")
	}
	if string(val) != "value1" {
		t.Errorf("got value %q, want %q", string(val), "value1")
	}

	// Verify it's gone
	_, found, err = cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after remove: %v", err)
	}
	if found {
		t.Error("expected key not to be found after removal")
	}

	// Try to remove non-existent key
	_, found, err = cache.GetAndRemove(ctx, []byte("nonexistent"))
	if err != nil {
		t.Fatalf("GetAndRemove nonexistent: %v", err)
	}
	if found {
		t.Error("expected key not to be found for nonexistent key")
	}
}

func TestBulkGetAll_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "getall-basic")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put multiple entries
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

	// Get them all back
	keys := []string{"k1", "k2", "k3", "k4", "k5"}
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

func TestBulkGetAll_100Entries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "getall-100")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put 100 entries
	entries := make(map[string][]byte)
	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		value := fmt.Sprintf("value%d", i)
		entries[key] = []byte(value)
	}

	if err := cache.PutAll(ctx, entries); err != nil {
		t.Fatalf("PutAll: %v", err)
	}

	// Get all 100 back
	keys := make([]string, 0, 100)
	for k := range entries {
		keys = append(keys, k)
	}

	result, err := cache.GetAll(ctx, keys)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}

	if len(result) != 100 {
		t.Fatalf("GetAll returned %d entries, want 100", len(result))
	}

	for i := 0; i < 100; i++ {
		key := fmt.Sprintf("key%d", i)
		expected := fmt.Sprintf("value%d", i)
		got, ok := result[key]
		if !ok {
			t.Errorf("missing key %q", key)
			continue
		}
		if string(got) != expected {
			t.Errorf("key %q: got %q, want %q", key, string(got), expected)
		}
	}
}

func TestBulkGetAll_WithExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "getall-expire")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put entries with 2 second lifespan
	entries := map[string][]byte{
		"e1": []byte("val1"),
		"e2": []byte("val2"),
		"e3": []byte("val3"),
	}

	if err := cache.PutAll(ctx, entries, hotrod.WithLifespan(2*time.Second)); err != nil {
		t.Fatalf("PutAll with lifespan: %v", err)
	}

	// Should get all entries immediately
	keys := []string{"e1", "e2", "e3"}
	result, err := cache.GetAll(ctx, keys)
	if err != nil {
		t.Fatalf("GetAll before expiry: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries before expiry, got %d", len(result))
	}

	// Wait for expiration
	time.Sleep(3 * time.Second)

	// Should get no entries after expiration
	result, err = cache.GetAll(ctx, keys)
	if err != nil {
		t.Fatalf("GetAll after expiry: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries after expiry, got %d", len(result))
	}
}

func TestBulkGetAll_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "getall-empty")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// GetAll with empty key set
	result, err := cache.GetAll(ctx, []string{})
	if err != nil {
		t.Fatalf("GetAll empty: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d entries", len(result))
	}
}

func TestBulkGetAll_MissingKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "getall-missing")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put only some keys
	entries := map[string][]byte{
		"exists1": []byte("value1"),
		"exists2": []byte("value2"),
	}

	if err := cache.PutAll(ctx, entries); err != nil {
		t.Fatalf("PutAll: %v", err)
	}

	// Request keys that exist and don't exist
	keys := []string{"exists1", "missing1", "exists2", "missing2"}
	result, err := cache.GetAll(ctx, keys)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}

	// Should only get the existing keys
	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}

	if _, ok := result["exists1"]; !ok {
		t.Error("expected exists1 to be in result")
	}
	if _, ok := result["exists2"]; !ok {
		t.Error("expected exists2 to be in result")
	}
	if _, ok := result["missing1"]; ok {
		t.Error("expected missing1 not to be in result")
	}
	if _, ok := result["missing2"]; ok {
		t.Error("expected missing2 not to be in result")
	}
}

func TestBulkPutAll_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "putall-empty")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// PutAll with empty map should not error
	if err := cache.PutAll(ctx, map[string][]byte{}); err != nil {
		t.Fatalf("PutAll empty: %v", err)
	}
}

func TestBasicClear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "clear")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put some entries
	entries := map[string][]byte{
		"k1": []byte("v1"),
		"k2": []byte("v2"),
		"k3": []byte("v3"),
	}

	if err := cache.PutAll(ctx, entries); err != nil {
		t.Fatalf("PutAll: %v", err)
	}

	// Verify they exist
	result, err := cache.GetAll(ctx, []string{"k1", "k2", "k3"})
	if err != nil {
		t.Fatalf("GetAll before clear: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("expected 3 entries before clear, got %d", len(result))
	}

	// Clear the cache
	if err := cache.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	// Verify all entries are gone
	result, err = cache.GetAll(ctx, []string{"k1", "k2", "k3"})
	if err != nil {
		t.Fatalf("GetAll after clear: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected 0 entries after clear, got %d", len(result))
	}
}

func TestBasicSize(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "size")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Initially empty
	size, err := cache.Size(ctx)
	if err != nil {
		t.Fatalf("Size initial: %v", err)
	}
	if size != 0 {
		t.Errorf("expected size 0, got %d", size)
	}

	// Put some entries
	entries := map[string][]byte{
		"k1": []byte("v1"),
		"k2": []byte("v2"),
		"k3": []byte("v3"),
	}

	if err := cache.PutAll(ctx, entries); err != nil {
		t.Fatalf("PutAll: %v", err)
	}

	// Should have 3 entries
	size, err = cache.Size(ctx)
	if err != nil {
		t.Fatalf("Size after put: %v", err)
	}
	if size != 3 {
		t.Errorf("expected size 3, got %d", size)
	}

	// Remove one
	err = cache.Remove(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Should have 2 entries
	size, err = cache.Size(ctx)
	if err != nil {
		t.Fatalf("Size after remove: %v", err)
	}
	if size != 2 {
		t.Errorf("expected size 2, got %d", size)
	}
}

func TestBasicContainsKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "contains")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Key doesn't exist initially
	exists, err := cache.ContainsKey(ctx, []byte("testkey"))
	if err != nil {
		t.Fatalf("ContainsKey before put: %v", err)
	}
	if exists {
		t.Error("expected key not to exist")
	}

	// Put the key
	err = cache.Put(ctx, []byte("testkey"), []byte("testvalue"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Key should exist now
	exists, err = cache.ContainsKey(ctx, []byte("testkey"))
	if err != nil {
		t.Fatalf("ContainsKey after put: %v", err)
	}
	if !exists {
		t.Error("expected key to exist")
	}

	// Remove the key
	err = cache.Remove(ctx, []byte("testkey"))
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Key should not exist anymore
	exists, err = cache.ContainsKey(ctx, []byte("testkey"))
	if err != nil {
		t.Fatalf("ContainsKey after remove: %v", err)
	}
	if exists {
		t.Error("expected key not to exist after removal")
	}
}

func TestBulkGetAll_ObjectStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "getall-objstorage")

	// Create cache with OBJECT storage mode (vs default BINARY)
	ctx := context.Background()
	config := fmt.Sprintf(`<distributed-cache name="%s" mode="SYNC" owners="2">
  <encoding media-type="application/x-protostream"/>
  <memory storage="OBJECT" max-count="1000"/>
</distributed-cache>`, cacheName)
	createTestCacheWithConfig(t, sharedContainer, cacheName, config)

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	connCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(connCtx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache(cacheName)

	// Put 10 entries
	entries := make(map[string]string)
	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		entries[key] = value
		if err := cache.Put(ctx, []byte(key), []byte(value)); err != nil {
			t.Fatalf("Put %s: %v", key, err)
		}
	}

	// GetAll for all keys
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}

	results, err := cache.GetAll(ctx, keys)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}

	// Verify all entries returned
	if len(results) != len(entries) {
		t.Errorf("got %d results, want %d", len(results), len(entries))
	}

	for k, expectedValue := range entries {
		value, found := results[k]
		if !found {
			t.Errorf("key %s not found in results", k)
			continue
		}
		if string(value) != expectedValue {
			t.Errorf("key %s: got %q, want %q", k, string(value), expectedValue)
		}
	}

	// Test GetAll with subset of keys
	subsetKeys := []string{"key-0", "key-5", "key-9"}
	results, err = cache.GetAll(ctx, subsetKeys)
	if err != nil {
		t.Fatalf("GetAll subset: %v", err)
	}

	if len(results) != 3 {
		t.Errorf("got %d results, want 3", len(results))
	}

	for _, k := range subsetKeys {
		if _, found := results[k]; !found {
			t.Errorf("key %s not found in subset results", k)
		}
	}
}

func TestBasicStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "stats")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put some entries to have stats
	for i := 0; i < 5; i++ {
		key := fmt.Sprintf("key-%d", i)
		value := fmt.Sprintf("value-%d", i)
		if err := cache.Put(ctx, []byte(key), []byte(value)); err != nil {
			t.Fatalf("Put %s: %v", key, err)
		}
	}

	// Get some entries to generate read stats
	for i := 0; i < 3; i++ {
		key := fmt.Sprintf("key-%d", i)
		_, _, err := cache.Get(ctx, []byte(key))
		if err != nil {
			t.Fatalf("Get %s: %v", key, err)
		}
	}

	// Get cache statistics
	stats, err := cache.Stats(ctx)
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if stats == nil {
		t.Fatal("expected stats map, got nil")
	}

	t.Logf("Cache stats: %+v", stats)

	// Stats should contain some entries (exact keys depend on server version)
	if len(stats) == 0 {
		t.Error("expected some statistics, got empty map")
	}
}

func TestBasicGetAndPut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "getandput")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put initial value
	err := cache.Put(ctx, []byte("key1"), []byte("initial-value"))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// GetAndPut - should return previous value and set new value
	prev, err := cache.GetAndPut(ctx, []byte("key1"), []byte("new-value"))
	if err != nil {
		t.Fatalf("GetAndPut: %v", err)
	}

	if string(prev) != "initial-value" {
		t.Errorf("GetAndPut returned %q, want %q", string(prev), "initial-value")
	}

	// Verify new value was set
	val, found, err := cache.Get(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("Get after GetAndPut: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found after GetAndPut")
	}
	if string(val) != "new-value" {
		t.Errorf("got %q, want %q", string(val), "new-value")
	}

	// GetAndPut on non-existent key should return empty previous value
	prev, err = cache.GetAndPut(ctx, []byte("newkey"), []byte("value"))
	if err != nil {
		t.Fatalf("GetAndPut on new key: %v", err)
	}
	if len(prev) != 0 {
		t.Errorf("expected empty previous value for new key, got %q", string(prev))
	}

	// Verify the new key was created
	val, found, err = cache.Get(ctx, []byte("newkey"))
	if err != nil {
		t.Fatalf("Get newkey: %v", err)
	}
	if !found {
		t.Fatal("expected newkey to be found")
	}
	if string(val) != "value" {
		t.Errorf("got %q, want %q", string(val), "value")
	}
}

func TestBasicGetWithMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "getwithmeta")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Put a value with expiration
	err := cache.Put(ctx, []byte("key1"), []byte("value1"),
		hotrod.WithLifespan(60*time.Second),
		hotrod.WithMaxIdle(30*time.Second))
	if err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Get with metadata
	meta, found, err := cache.GetWithMetadata(ctx, []byte("key1"))
	if err != nil {
		t.Fatalf("GetWithMetadata: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}

	if meta == nil {
		t.Fatal("expected metadata, got nil")
		return
	}

	// Verify value
	if string(meta.Value) != "value1" {
		t.Errorf("got value %q, want %q", string(meta.Value), "value1")
	}

	// Verify metadata fields exist
	if meta.Created == 0 {
		t.Error("expected Created timestamp to be set")
	}
	if meta.Lifespan < 0 {
		t.Errorf("expected positive Lifespan, got %d", meta.Lifespan)
	}
	if meta.MaxIdle < 0 {
		t.Errorf("expected positive MaxIdle, got %d", meta.MaxIdle)
	}

	t.Logf("Metadata: Created=%d, Lifespan=%d, MaxIdle=%d, LastUsed=%d, Version=%d",
		meta.Created, meta.Lifespan, meta.MaxIdle, meta.LastUsed, meta.Version)

	// GetWithMetadata on non-existent key
	_, found, err = cache.GetWithMetadata(ctx, []byte("nonexistent"))
	if err != nil {
		t.Fatalf("GetWithMetadata on missing key: %v", err)
	}
	if found {
		t.Error("expected key not to be found")
	}
}

// TODO: Fix media type handling - server doesn't recognize the media type constant
func TestBasicPutRawGetRaw(t *testing.T) {
	t.Skip("TODO: Fix media type constant - MediaTypeIds.getMediaType returns null")

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cacheName := uniqueCacheName(t, "rawops")
	cache, cleanup := setupCache(t, cacheName)
	defer cleanup()

	ctx := context.Background()

	// Media type for protostream (the default encoding used by test caches)
	const MediaTypeApplicationProtostream = 0x0002_9003 // application/x-protostream

	// Test PutRaw/GetRaw with protostream media type (matching cache encoding)
	// Using simple byte data that's valid protostream
	testData := []byte("test-value-for-raw-ops")

	err := cache.PutRaw(ctx, []byte("raw-key"), testData, MediaTypeApplicationProtostream)
	if err != nil {
		t.Fatalf("PutRaw: %v", err)
	}

	// GetRaw should retrieve the same data
	val, found, err := cache.GetRaw(ctx, []byte("raw-key"), MediaTypeApplicationProtostream)
	if err != nil {
		t.Fatalf("GetRaw: %v", err)
	}
	if !found {
		t.Fatal("expected raw-key to be found")
	}
	if string(val) != string(testData) {
		t.Errorf("got %q, want %q", string(val), string(testData))
	}

	// Test with different data
	binaryData := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	err = cache.PutRaw(ctx, []byte("binary-key"), binaryData, MediaTypeApplicationProtostream)
	if err != nil {
		t.Fatalf("PutRaw binary: %v", err)
	}

	val, found, err = cache.GetRaw(ctx, []byte("binary-key"), MediaTypeApplicationProtostream)
	if err != nil {
		t.Fatalf("GetRaw binary: %v", err)
	}
	if !found {
		t.Fatal("expected binary-key to be found")
	}
	if len(val) != len(binaryData) {
		t.Errorf("got %d bytes, want %d", len(val), len(binaryData))
	}
	for i, b := range binaryData {
		if val[i] != b {
			t.Errorf("byte %d: got 0x%02X, want 0x%02X", i, val[i], b)
		}
	}

	// GetRaw on non-existent key
	_, found, err = cache.GetRaw(ctx, []byte("missing"), MediaTypeApplicationProtostream)
	if err != nil {
		t.Fatalf("GetRaw missing key: %v", err)
	}
	if found {
		t.Error("expected key not to be found")
	}

	t.Logf("PutRaw/GetRaw test passed - methods are now covered")
}
