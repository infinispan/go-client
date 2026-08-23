package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestNearCache_GetCachesLocally(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "nc-get")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	remote := client.Cache("nc-get")
	if err := remote.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	nc, err := hotrod.NewNearCache(ctx, client, "nc-get", hotrod.WithMaxNearCacheEntries(100))
	if err != nil {
		t.Fatalf("NewNearCache: %v", err)
	}
	defer nc.Close()

	// First get — cache miss, fetches from server
	val, found, err := nc.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(val) != "v1" {
		t.Errorf("first Get = %q, %v; want %q, true", val, found, "v1")
	}

	// Second get — should be cached locally
	val, found, err = nc.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get (cached): %v", err)
	}
	if !found || string(val) != "v1" {
		t.Errorf("cached Get = %q, %v; want %q, true", val, found, "v1")
	}
}

func TestNearCache_PutInvalidatesLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "nc-put")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	nc, err := hotrod.NewNearCache(ctx, client, "nc-put", hotrod.WithMaxNearCacheEntries(100))
	if err != nil {
		t.Fatalf("NewNearCache: %v", err)
	}
	defer nc.Close()

	// Put and populate near cache via get
	if err := nc.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put v1: %v", err)
	}
	_, _, _ = nc.Get(ctx, []byte("k1"))

	// Put new value — should invalidate local cache
	if err := nc.Put(ctx, []byte("k1"), []byte("v2")); err != nil {
		t.Fatalf("Put v2: %v", err)
	}

	// Get should return new value (re-fetched from server)
	val, found, err := nc.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after Put: %v", err)
	}
	if !found || string(val) != "v2" {
		t.Errorf("Get = %q, %v; want %q, true", val, found, "v2")
	}
}

func TestNearCache_RemoteUpdateInvalidates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "nc-remote")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client1, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient 1: %v", err)
	}
	defer client1.Close()

	client2, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient 2: %v", err)
	}
	defer client2.Close()

	// Put initial value via client1
	remote1 := client1.Cache("nc-remote")
	if err := remote1.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put v1: %v", err)
	}

	// Create near cache on client1 and populate it
	nc, err := hotrod.NewNearCache(ctx, client1, "nc-remote", hotrod.WithMaxNearCacheEntries(100))
	if err != nil {
		t.Fatalf("NewNearCache: %v", err)
	}
	defer nc.Close()

	val, found, err := nc.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(val) != "v1" {
		t.Fatalf("initial Get = %q, %v; want %q, true", val, found, "v1")
	}

	// Allow the bloom filter update to reach the server
	time.Sleep(200 * time.Millisecond)

	// Update from client2 (separate connection)
	remote2 := client2.Cache("nc-remote")
	if err := remote2.Put(ctx, []byte("k1"), []byte("v2")); err != nil {
		t.Fatalf("Put v2 from client2: %v", err)
	}

	// Wait for invalidation event to arrive
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		val, found, err = nc.Get(ctx, []byte("k1"))
		if err != nil {
			t.Fatalf("Get after remote update: %v", err)
		}
		if found && string(val) == "v2" {
			return // success
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("near cache still returned %q after remote update, want %q", val, "v2")
}

func TestNearCache_RemoveInvalidatesLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "nc-remove")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	nc, err := hotrod.NewNearCache(ctx, client, "nc-remove", hotrod.WithMaxNearCacheEntries(100))
	if err != nil {
		t.Fatalf("NewNearCache: %v", err)
	}
	defer nc.Close()

	// Put and populate near cache
	if err := nc.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	_, _, _ = nc.Get(ctx, []byte("k1"))

	// Remove
	if err := nc.Remove(ctx, []byte("k1")); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Get should return not-found
	val, found, err := nc.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after Remove: %v", err)
	}
	if found {
		t.Errorf("Get after Remove returned found=true, val=%q; want found=false", val)
	}
}
