package hotrod_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"infinispan.org/go-client/hotrod"
)

func serverImage() string {
	if img := os.Getenv("INFINISPAN_SERVER_IMAGE"); img != "" {
		return img
	}
	return "quay.io/infinispan/server:latest"
}

func createTestCache(t *testing.T, container testcontainers.Container, name string) {
	t.Helper()
	ctx := context.Background()
	cmd := fmt.Sprintf("create cache --template=org.infinispan.DIST_SYNC %s", name)
	_, output, err := container.Exec(ctx, []string{
		"bash", "-c",
		fmt.Sprintf("echo '%s' | /opt/infinispan/bin/cli.sh -c http://admin:password@localhost:11222", cmd),
	})
	if err != nil {
		t.Fatalf("exec create cache: %v", err)
	}
	body, _ := io.ReadAll(output)
	t.Logf("create cache output: %s", body)
}

// createTestCacheWithConfig creates a cache with custom XML configuration
func createTestCacheWithConfig(t *testing.T, container testcontainers.Container, name, config string) {
	t.Helper()
	ctx := context.Background()

	// Write config to a temporary file in the container
	tmpFile := fmt.Sprintf("/tmp/%s.xml", name)
	escapedConfig := strings.ReplaceAll(config, "'", "'\\''")

	_, output, err := container.Exec(ctx, []string{
		"bash", "-c",
		fmt.Sprintf("echo '%s' > %s", escapedConfig, tmpFile),
	})
	if err != nil {
		t.Fatalf("write config file: %v", err)
	}
	_, _ = io.ReadAll(output)

	// Create cache from the file
	cmd := fmt.Sprintf("create cache --file=%s %s", tmpFile, name)
	_, output, err = container.Exec(ctx, []string{
		"bash", "-c",
		fmt.Sprintf("echo '%s' | /opt/infinispan/bin/cli.sh -c http://admin:password@localhost:11222", cmd),
	})
	if err != nil {
		t.Fatalf("exec create cache with config: %v", err)
	}
	body, _ := io.ReadAll(output)
	t.Logf("create cache output: %s", body)
}

// createCacheWithExpiration creates a cache with default expiration settings
func createCacheWithExpiration(t *testing.T, container testcontainers.Container, name string, lifespanSeconds, maxIdleSeconds int64) {
	t.Helper()
	config := fmt.Sprintf(`<distributed-cache name="%s">
  <encoding media-type="application/x-protostream"/>
  <expiration lifespan="%d" max-idle="%d"/>
</distributed-cache>`, name, lifespanSeconds*1000, maxIdleSeconds*1000)
	createTestCacheWithConfig(t, container, name, config)
}

func TestEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "e2e")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("e2e")

	if err := cache.Put(ctx, []byte("greeting"), []byte("hello world")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("greeting"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if string(val) != "hello world" {
		t.Errorf("value = %q, want %q", string(val), "hello world")
	}

	_, found, err = cache.Get(ctx, []byte("missing"))
	if err != nil {
		t.Fatalf("Get missing: %v", err)
	}
	if found {
		t.Error("expected key not found for missing key")
	}
}

func TestEndToEnd_PutWithLifespan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "ttl")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("ttl")

	if err := cache.Put(ctx, []byte("ephemeral"), []byte("gone soon"), hotrod.WithLifespan(1*time.Second)); err != nil {
		t.Fatalf("Put with lifespan: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("ephemeral"))
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if !found || string(val) != "gone soon" {
		t.Fatalf("expected value %q before expiry, got found=%v val=%q", "gone soon", found, string(val))
	}

	time.Sleep(2 * time.Second)

	_, found, err = cache.Get(ctx, []byte("ephemeral"))
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected key to be expired")
	}
}

func TestGetAndPut(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "put-prev")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("put-prev")

	// First put — no previous value
	prev, err := cache.GetAndPut(ctx, []byte("k1"), []byte("v1"))
	if err != nil {
		t.Fatalf("GetAndPut (first): %v", err)
	}
	if prev != nil {
		t.Errorf("expected nil previous, got %q", string(prev))
	}

	// Second put — should return previous value
	prev, err = cache.GetAndPut(ctx, []byte("k1"), []byte("v2"))
	if err != nil {
		t.Fatalf("GetAndPut (second): %v", err)
	}
	if string(prev) != "v1" {
		t.Errorf("previous = %q, want %q", string(prev), "v1")
	}
}

func TestGetAndRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "rm-prev")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("rm-prev")

	if err := cache.Put(ctx, []byte("k1"), []byte("val")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Remove existing key — should return previous value
	prev, existed, err := cache.GetAndRemove(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("GetAndRemove: %v", err)
	}
	if !existed {
		t.Error("expected existed=true")
	}
	if string(prev) != "val" {
		t.Errorf("previous = %q, want %q", string(prev), "val")
	}

	// Remove non-existing key — should return nil
	prev, existed, err = cache.GetAndRemove(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("GetAndRemove (missing): %v", err)
	}
	if existed {
		t.Error("expected existed=false")
	}
	if prev != nil {
		t.Errorf("expected nil previous, got %q", string(prev))
	}
}

func TestPutWithFlags(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "put-flags")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("put-flags")

	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Put with FORCE_RETURN_VALUE flag — same as GetAndPut but using explicit flag
	if err := cache.Put(ctx, []byte("k1"), []byte("v2"), hotrod.WithPutFlag(hotrod.FlagForceReturnValue)); err != nil {
		t.Fatalf("Put with flag: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(val) != "v2" {
		t.Errorf("expected v2, got found=%v val=%q", found, string(val))
	}
}

func TestEndToEnd_ExplicitScramAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "scramauth")

	uri := fmt.Sprintf("hotrod://%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithAuth("SCRAM-SHA-256", "admin", "password"))
	if err != nil {
		t.Fatalf("NewClient with explicit SCRAM auth: %v", err)
	}
	defer client.Close()

	cache := client.Cache("scramauth")

	if err := cache.Put(ctx, []byte("k"), []byte("v")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("k"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found || string(val) != "v" {
		t.Errorf("expected found=true val=%q, got found=%v val=%q", "v", found, string(val))
	}
}
