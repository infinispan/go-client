package hotrod_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/testproto"
)

const essayProtoSchema = `syntax = "proto3";
package test;

message Essay {
  string title = 1;
  string content = 2;
}
`

// makeHugeString creates a large string of the specified size
// Pattern: repeating characters 'a' through 't' (20 chars)
func makeHugeString(size int) string {
	var builder strings.Builder
	builder.Grow(size)

	for i := 0; i < size; i++ {
		char := byte('a' + (i % 20))
		builder.WriteByte(char)
	}

	return builder.String()
}

// TestLargeMessage_1MB tests a 1MB message as a baseline
func TestLargeMessage_1MB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const size = 1_000_000 // 1MB

	createTestCache(t, sharedContainer, "essay-1mb")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Register schema
	if err := client.Schemas().Register(ctx, "essay.proto", essayProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.Essay](
		client, "essay-1mb",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Essay { return &testproto.Essay{} },
	)

	t.Logf("Creating 1MB string...")
	content := makeHugeString(size)
	essay := &testproto.Essay{
		Title:   "1MB Essay",
		Content: content,
	}

	// Put the large essay
	t.Logf("Putting 1MB essay to cache...")
	if err := cache.Put(ctx, 1, essay); err != nil {
		t.Fatalf("Put 1MB essay: %v", err)
	}

	// Get it back
	t.Logf("Getting 1MB essay from cache...")
	retrieved, found, err := cache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get 1MB essay: %v", err)
	}
	if !found {
		t.Fatal("expected essay to be found")
	}

	// Verify
	if retrieved.Title != "1MB Essay" {
		t.Errorf("Title = %q, want %q", retrieved.Title, "1MB Essay")
	}
	if len(retrieved.Content) != size {
		t.Errorf("Content length = %d, want %d", len(retrieved.Content), size)
	}
	if retrieved.Content != content {
		t.Error("Content mismatch after round-trip")
	}

	t.Logf("✓ Successfully stored and retrieved 1MB message")
}

// TestLargeMessage_10MB tests a 10MB message
// NOTE: This test will fail with default Infinispan server configuration.
// Default max-content-length appears to be between 8-10MB.
// To run this test, configure server with: max-content-length >= 10MB
func TestLargeMessage_10MB(t *testing.T) {
	t.Skip("Skipping: exceeds default server max-content-length (8-10MB)")

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const size = 10_000_000 // 10MB

	createTestCache(t, sharedContainer, "essay-10mb")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Register schema
	if err := client.Schemas().Register(ctx, "essay.proto", essayProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.Essay](
		client, "essay-10mb",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Essay { return &testproto.Essay{} },
	)

	t.Logf("Creating 10MB string...")
	content := makeHugeString(size)
	essay := &testproto.Essay{
		Title:   "10MB Essay",
		Content: content,
	}

	// Put the large essay
	t.Logf("Putting 10MB essay to cache...")
	start := time.Now()
	if err := cache.Put(ctx, 1, essay); err != nil {
		t.Fatalf("Put 10MB essay: %v", err)
	}
	putDuration := time.Since(start)
	t.Logf("Put took %v", putDuration)

	// Get it back
	t.Logf("Getting 10MB essay from cache...")
	start = time.Now()
	retrieved, found, err := cache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get 10MB essay: %v", err)
	}
	getDuration := time.Since(start)
	t.Logf("Get took %v", getDuration)

	if !found {
		t.Fatal("expected essay to be found")
	}

	// Verify
	if retrieved.Title != "10MB Essay" {
		t.Errorf("Title = %q, want %q", retrieved.Title, "10MB Essay")
	}
	if len(retrieved.Content) != size {
		t.Errorf("Content length = %d, want %d", len(retrieved.Content), size)
	}

	// Sample check (don't compare all 10MB char by char)
	if retrieved.Content[:100] != content[:100] {
		t.Error("Content prefix mismatch")
	}
	if retrieved.Content[size-100:] != content[size-100:] {
		t.Error("Content suffix mismatch")
	}

	t.Logf("✓ Successfully stored and retrieved 10MB message")
}

// TestLargeMessage_50MB tests a 50MB message (getting close to 64MB limit)
// NOTE: This test requires server configuration with increased max-content-length
func TestLargeMessage_50MB(t *testing.T) {
	t.Skip("Skipping: requires server configuration with max-content-length >= 50MB")

	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const size = 50_000_000 // 50MB

	createTestCache(t, sharedContainer, "essay-50mb")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	// Extended timeout for large message
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Register schema
	if err := client.Schemas().Register(ctx, "essay.proto", essayProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.Essay](
		client, "essay-50mb",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Essay { return &testproto.Essay{} },
	)

	t.Logf("Creating 50MB string...")
	content := makeHugeString(size)
	essay := &testproto.Essay{
		Title:   "50MB Essay",
		Content: content,
	}

	// Put the large essay
	t.Logf("Putting 50MB essay to cache...")
	start := time.Now()
	if err := cache.Put(ctx, 1, essay); err != nil {
		t.Fatalf("Put 50MB essay: %v", err)
	}
	putDuration := time.Since(start)
	t.Logf("Put took %v", putDuration)

	// Get it back
	t.Logf("Getting 50MB essay from cache...")
	start = time.Now()
	retrieved, found, err := cache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get 50MB essay: %v", err)
	}
	getDuration := time.Since(start)
	t.Logf("Get took %v", getDuration)

	if !found {
		t.Fatal("expected essay to be found")
	}

	// Verify
	if retrieved.Title != "50MB Essay" {
		t.Errorf("Title = %q, want %q", retrieved.Title, "50MB Essay")
	}
	if len(retrieved.Content) != size {
		t.Errorf("Content length = %d, want %d", len(retrieved.Content), size)
	}

	// Sample check
	if retrieved.Content[:100] != content[:100] {
		t.Error("Content prefix mismatch")
	}
	if retrieved.Content[size-100:] != content[size-100:] {
		t.Error("Content suffix mismatch")
	}

	t.Logf("✓ Successfully stored and retrieved 50MB message")
}

// TestLargeMessage_Performance measures performance with various sizes
func TestLargeMessage_Performance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	sizes := []struct {
		name string
		size int
	}{
		{"100KB", 100_000},
		{"500KB", 500_000},
		{"1MB", 1_000_000},
		{"5MB", 5_000_000},
	}

	createTestCache(t, sharedContainer, "essay-perf")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Register schema
	if err := client.Schemas().Register(ctx, "essay.proto", essayProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.Essay](
		client, "essay-perf",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Essay { return &testproto.Essay{} },
	)

	for i, test := range sizes {
		t.Run(test.name, func(t *testing.T) {
			content := makeHugeString(test.size)
			essay := &testproto.Essay{
				Title:   fmt.Sprintf("Essay-%s", test.name),
				Content: content,
			}

			// Put
			start := time.Now()
			if err := cache.Put(ctx, int32(i+1), essay); err != nil {
				t.Fatalf("Put %s: %v", test.name, err)
			}
			putDuration := time.Since(start)

			// Get
			start = time.Now()
			retrieved, found, err := cache.Get(ctx, int32(i+1))
			if err != nil {
				t.Fatalf("Get %s: %v", test.name, err)
			}
			getDuration := time.Since(start)

			if !found {
				t.Fatal("expected essay to be found")
			}
			if len(retrieved.Content) != test.size {
				t.Errorf("Content length = %d, want %d", len(retrieved.Content), test.size)
			}

			// Calculate throughput
			putMBps := float64(test.size) / putDuration.Seconds() / 1_000_000
			getMBps := float64(test.size) / getDuration.Seconds() / 1_000_000

			t.Logf("%s: Put=%.2f MB/s (%v), Get=%.2f MB/s (%v)",
				test.name, putMBps, putDuration, getMBps, getDuration)
		})
	}
}

// TestLargeMessage_ConcurrentPuts tests multiple large messages concurrently
func TestLargeMessage_ConcurrentPuts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const size = 1_000_000 // 1MB each
	const numMessages = 5

	createTestCache(t, sharedContainer, "essay-concurrent")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Register schema
	if err := client.Schemas().Register(ctx, "essay.proto", essayProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.Essay](
		client, "essay-concurrent",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Essay { return &testproto.Essay{} },
	)

	// Put multiple essays concurrently
	errChan := make(chan error, numMessages)
	start := time.Now()

	for i := 0; i < numMessages; i++ {
		go func(id int) {
			content := makeHugeString(size)
			essay := &testproto.Essay{
				Title:   fmt.Sprintf("Essay-%d", id),
				Content: content,
			}
			errChan <- cache.Put(ctx, int32(id), essay)
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < numMessages; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent put %d failed: %v", i, err)
		}
	}
	duration := time.Since(start)

	t.Logf("Put %d x %dMB messages concurrently in %v", numMessages, size/1_000_000, duration)

	// Verify all essays are present
	for i := 0; i < numMessages; i++ {
		retrieved, found, err := cache.Get(ctx, int32(i))
		if err != nil {
			t.Errorf("Get essay %d: %v", i, err)
		}
		if !found {
			t.Errorf("Essay %d not found", i)
		}
		if len(retrieved.Content) != size {
			t.Errorf("Essay %d: length = %d, want %d", i, len(retrieved.Content), size)
		}
	}

	t.Logf("✓ All %d concurrent messages stored and verified", numMessages)
}
