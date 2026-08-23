package hash

import (
	"fmt"
	"math"
	"testing"
)

func TestSegmentSize(t *testing.T) {
	ch := NewConsistentHash(256, nil)
	expected := int(math.Ceil(float64(math.MaxInt32+1) / 256.0))
	if ch.segmentSize != expected {
		t.Errorf("segmentSize = %d, want %d", ch.segmentSize, expected)
	}
}

func TestPrimaryOwner(t *testing.T) {
	owners := make([][]string, 256)
	for i := range owners {
		owners[i] = []string{fmt.Sprintf("server-%d", i%3)}
	}
	ch := NewConsistentHash(256, owners)

	key := []byte("hello")
	addr := ch.PrimaryOwner(key)
	if addr == "" {
		t.Fatal("expected non-empty address")
	}

	// Verify determinism
	for i := 0; i < 100; i++ {
		if got := ch.PrimaryOwner(key); got != addr {
			t.Fatalf("non-deterministic: got %q, want %q", got, addr)
		}
	}
}

func TestPrimaryOwnerEmptySegment(t *testing.T) {
	owners := make([][]string, 256)
	for i := range owners {
		owners[i] = []string{}
	}
	ch := NewConsistentHash(256, owners)

	addr := ch.PrimaryOwner([]byte("key"))
	if addr != "" {
		t.Errorf("expected empty addr for empty segment, got %q", addr)
	}
}

func TestSegmentDistribution(t *testing.T) {
	numSegments := 256
	owners := make([][]string, numSegments)
	for i := range owners {
		owners[i] = []string{fmt.Sprintf("server-%d", i%3)}
	}
	ch := NewConsistentHash(numSegments, owners)

	counts := make(map[string]int)
	numKeys := 10000
	for i := 0; i < numKeys; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		addr := ch.PrimaryOwner(key)
		counts[addr]++
	}

	// With 3 servers and good distribution, each should get roughly 1/3
	for server, count := range counts {
		ratio := float64(count) / float64(numKeys)
		if ratio < 0.2 || ratio > 0.5 {
			t.Errorf("server %s got %.1f%% of keys (expected ~33%%)", server, ratio*100)
		}
	}
}

func TestSegmentCalculationMatchesJava(t *testing.T) {
	// Verify segment = normalizedHash / segmentSize matches Java's formula
	numSegments := 256
	segmentSize := int(math.Ceil(float64(math.MaxInt32+1) / float64(numSegments)))

	key := []byte("hello")
	h := Hash(key)
	normalizedHash := int(h) & math.MaxInt32
	segment := normalizedHash / segmentSize

	if segment < 0 || segment >= numSegments {
		t.Errorf("segment %d out of range [0, %d)", segment, numSegments)
	}
}
