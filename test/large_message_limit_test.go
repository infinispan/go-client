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

// TestLargeMessage_FindDefaultLimit tests various sizes to find the default
// server max-content-length limit
func TestLargeMessage_FindDefaultLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Test various sizes to find the limit
	sizes := []struct {
		name string
		size int
	}{
		{"1MB", 1_000_000},
		{"2MB", 2_000_000},
		{"3MB", 3_000_000},
		{"4MB", 4_000_000},
		{"5MB", 5_000_000},
		{"6MB", 6_000_000},
		{"7MB", 7_000_000},
		{"8MB", 8_000_000},
	}

	createTestCache(t, sharedContainer, "essay-limit-test")

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
		client, "essay-limit-test",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Essay { return &testproto.Essay{} },
	)

	var maxSuccessful int
	var firstFailure int

	for i, test := range sizes {
		t.Run(test.name, func(t *testing.T) {
			var builder strings.Builder
			builder.Grow(test.size)
			for j := 0; j < test.size; j++ {
				builder.WriteByte(byte('a' + (j % 20)))
			}
			content := builder.String()

			essay := &testproto.Essay{
				Title:   fmt.Sprintf("Essay-%s", test.name),
				Content: content,
			}

			// Try to put
			err := cache.Put(ctx, int32(i+100), essay)
			if err != nil {
				if strings.Contains(err.Error(), "connection reset") ||
					strings.Contains(err.Error(), "EOF") ||
					strings.Contains(err.Error(), "broken pipe") {
					t.Logf("✗ %s: Failed (likely exceeds server max-content-length)", test.name)
					if firstFailure == 0 {
						firstFailure = test.size
					}
				} else {
					t.Fatalf("Unexpected error for %s: %v", test.name, err)
				}
				return
			}

			// Successfully put
			t.Logf("✓ %s: Success", test.name)
			maxSuccessful = test.size
		})
	}

	if firstFailure > 0 {
		t.Logf("Server max-content-length appears to be between %d and %d bytes",
			maxSuccessful, firstFailure)
	} else {
		t.Logf("All tested sizes (up to %d bytes) succeeded", maxSuccessful)
	}
}

// TestLargeMessage_ServerLimitDocumented documents that the default Infinispan
// server has a max-content-length limit (typically 5MB by default)
func TestLargeMessage_ServerLimitDocumented(t *testing.T) {
	t.Log("Infinispan server has a default max-content-length limit")
	t.Log("Default is typically 5MB (5242880 bytes)")
	t.Log("")
	t.Log("Messages larger than this limit will fail with:")
	t.Log("  - 'connection reset by peer'")
	t.Log("  - 'EOF'")
	t.Log("  - 'broken pipe'")
	t.Log("")
	t.Log("To increase the limit, configure the server with:")
	t.Log("  <hotrod-connector max-content-length=\"100MB\"/>")
	t.Log("")
	t.Log("Or via environment variable:")
	t.Log("  INFINISPAN_SERVER_HOTROD_MAX_CONTENT_LENGTH=104857600")
}
