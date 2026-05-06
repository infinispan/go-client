package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestGetWithVersion(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "cas-gwv")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("cas-gwv")

	// Missing key returns not-found.
	vv, found, err := cache.GetWithVersion(ctx, []byte("missing"))
	if err != nil {
		t.Fatalf("GetWithVersion missing: %v", err)
	}
	if found {
		t.Fatal("expected not found for missing key")
	}
	if vv != nil {
		t.Fatal("expected nil VersionedValue for missing key")
	}

	// Put a value and retrieve with version.
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	vv, found, err = cache.GetWithVersion(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("GetWithVersion: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if string(vv.Value) != "v1" {
		t.Fatalf("got value %q, want %q", string(vv.Value), "v1")
	}
	if vv.Version == 0 {
		t.Fatal("expected non-zero version")
	}

	// Update and verify version changes.
	if err := cache.Put(ctx, []byte("k1"), []byte("v2")); err != nil {
		t.Fatalf("Put update: %v", err)
	}

	vv2, found, err := cache.GetWithVersion(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("GetWithVersion after update: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found after update")
	}
	if string(vv2.Value) != "v2" {
		t.Fatalf("got value %q, want %q", string(vv2.Value), "v2")
	}
	if vv2.Version == vv.Version {
		t.Fatal("expected version to change after update")
	}
}

func TestReplaceIfUnmodified(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "cas-riu")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("cas-riu")

	// Put initial value and get its version.
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	vv, _, err := cache.GetWithVersion(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("GetWithVersion: %v", err)
	}

	// Replace with correct version succeeds.
	ok, err := cache.ReplaceIfUnmodified(ctx, []byte("k1"), []byte("v2"), vv.Version)
	if err != nil {
		t.Fatalf("ReplaceIfUnmodified: %v", err)
	}
	if !ok {
		t.Fatal("expected replace to succeed with correct version")
	}

	// Verify the value was replaced.
	val, found, err := cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected key to exist")
	}
	if string(val) != "v2" {
		t.Fatalf("got %q, want %q", string(val), "v2")
	}

	// Replace with stale version fails.
	ok, err = cache.ReplaceIfUnmodified(ctx, []byte("k1"), []byte("v3"), vv.Version)
	if err != nil {
		t.Fatalf("ReplaceIfUnmodified stale: %v", err)
	}
	if ok {
		t.Fatal("expected replace to fail with stale version")
	}

	// Value should still be v2.
	val, _, err = cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after stale replace: %v", err)
	}
	if string(val) != "v2" {
		t.Fatalf("got %q, want %q after stale replace", string(val), "v2")
	}
}

func TestReplaceIfUnmodified_WithExpiry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "cas-riu-ttl")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("cas-riu-ttl")

	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	vv, _, err := cache.GetWithVersion(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("GetWithVersion: %v", err)
	}

	// Replace with a 1-second lifespan.
	ok, err := cache.ReplaceIfUnmodified(ctx, []byte("k1"), []byte("v2"), vv.Version, hotrod.WithLifespan(1*time.Second))
	if err != nil {
		t.Fatalf("ReplaceIfUnmodified with lifespan: %v", err)
	}
	if !ok {
		t.Fatal("expected replace to succeed")
	}

	time.Sleep(2 * time.Second)

	_, found, err := cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Fatal("expected key to have expired")
	}
}

func TestRemoveIfUnmodified(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "cas-rmiu")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("cas-rmiu")

	// Put and get version.
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	vv, _, err := cache.GetWithVersion(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("GetWithVersion: %v", err)
	}

	// Update the entry so the original version becomes stale.
	if err := cache.Put(ctx, []byte("k1"), []byte("v2")); err != nil {
		t.Fatalf("Put update: %v", err)
	}

	// Remove with stale version fails.
	ok, err := cache.RemoveIfUnmodified(ctx, []byte("k1"), vv.Version)
	if err != nil {
		t.Fatalf("RemoveIfUnmodified stale: %v", err)
	}
	if ok {
		t.Fatal("expected remove to fail with stale version")
	}

	// Key should still exist.
	_, found, err := cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after stale remove: %v", err)
	}
	if !found {
		t.Fatal("expected key to still exist after failed remove")
	}

	// Get the current version and remove with it.
	vv2, _, err := cache.GetWithVersion(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("GetWithVersion current: %v", err)
	}

	ok, err = cache.RemoveIfUnmodified(ctx, []byte("k1"), vv2.Version)
	if err != nil {
		t.Fatalf("RemoveIfUnmodified current: %v", err)
	}
	if !ok {
		t.Fatal("expected remove to succeed with current version")
	}

	// Key should be gone.
	_, found, err = cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after remove: %v", err)
	}
	if found {
		t.Fatal("expected key to be removed")
	}
}
