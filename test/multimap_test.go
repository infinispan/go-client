package hotrod_test

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestMultimap_PutAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "mm-putget")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	mm := client.Multimap("mm-putget")

	if err := mm.Put(ctx, []byte("colors"), [][]byte{[]byte("red"), []byte("blue"), []byte("green")}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	values, err := mm.Get(ctx, []byte("colors"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	got := toStringSlice(values)
	sort.Strings(got)
	want := []string{"blue", "green", "red"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMultimap_GetNotExist(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "mm-getne")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	mm := client.Multimap("mm-getne")
	values, err := mm.Get(ctx, []byte("missing"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("expected empty, got %d values", len(values))
	}
}

func TestMultimap_RemoveKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "mm-rmkey")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	mm := client.Multimap("mm-rmkey")

	_ = mm.Put(ctx, []byte("k"), [][]byte{[]byte("v1"), []byte("v2")})

	removed, err := mm.RemoveKey(ctx, []byte("k"))
	if err != nil {
		t.Fatalf("RemoveKey: %v", err)
	}
	if !removed {
		t.Error("expected removed=true")
	}

	values, _ := mm.Get(ctx, []byte("k"))
	if len(values) != 0 {
		t.Errorf("expected empty after remove, got %d", len(values))
	}

	removed, err = mm.RemoveKey(ctx, []byte("k"))
	if err != nil {
		t.Fatalf("RemoveKey missing: %v", err)
	}
	if removed {
		t.Error("expected removed=false for missing key")
	}
}

func TestMultimap_RemoveEntry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "mm-rment")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	mm := client.Multimap("mm-rment")

	_ = mm.Put(ctx, []byte("k"), [][]byte{[]byte("v1"), []byte("v2")})

	removed, err := mm.RemoveEntry(ctx, []byte("k"), []byte("v1"))
	if err != nil {
		t.Fatalf("RemoveEntry: %v", err)
	}
	if !removed {
		t.Error("expected removed=true")
	}

	values, _ := mm.Get(ctx, []byte("k"))
	got := toStringSlice(values)
	if len(got) != 1 || got[0] != "v2" {
		t.Errorf("expected [v2], got %v", got)
	}
}

func TestMultimap_HasKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "mm-ck")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	mm := client.Multimap("mm-ck")

	ok, _ := mm.HasKey(ctx, []byte("k"))
	if ok {
		t.Error("expected false before put")
	}

	_ = mm.Put(ctx, []byte("k"), [][]byte{[]byte("v")})

	ok, err = mm.HasKey(ctx, []byte("k"))
	if err != nil {
		t.Fatalf("HasKey: %v", err)
	}
	if !ok {
		t.Error("expected true after put")
	}
}

func TestMultimap_HasValue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "mm-cv")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	mm := client.Multimap("mm-cv")

	_ = mm.Put(ctx, []byte("k"), [][]byte{[]byte("v1")})

	ok, err := mm.HasValue(ctx, []byte("v1"))
	if err != nil {
		t.Fatalf("HasValue: %v", err)
	}
	if !ok {
		t.Error("expected true for existing value")
	}

	ok, _ = mm.HasValue(ctx, []byte("v999"))
	if ok {
		t.Error("expected false for missing value")
	}
}

func TestMultimap_HasEntry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "mm-ce")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	mm := client.Multimap("mm-ce")

	_ = mm.Put(ctx, []byte("k"), [][]byte{[]byte("v1"), []byte("v2")})

	ok, err := mm.HasEntry(ctx, []byte("k"), []byte("v1"))
	if err != nil {
		t.Fatalf("HasEntry: %v", err)
	}
	if !ok {
		t.Error("expected true for existing entry")
	}

	ok, _ = mm.HasEntry(ctx, []byte("k"), []byte("v999"))
	if ok {
		t.Error("expected false for missing entry")
	}
}

func TestMultimap_Size(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "mm-size")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	mm := client.Multimap("mm-size")

	size, err := mm.Size(ctx)
	if err != nil {
		t.Fatalf("Size empty: %v", err)
	}
	if size != 0 {
		t.Errorf("expected 0, got %d", size)
	}

	_ = mm.Put(ctx, []byte("k1"), [][]byte{[]byte("v1"), []byte("v2")})
	_ = mm.Put(ctx, []byte("k2"), [][]byte{[]byte("v3")})

	size, err = mm.Size(ctx)
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if size != 3 {
		t.Errorf("expected 3, got %d", size)
	}
}

func toStringSlice(bs [][]byte) []string {
	ss := make([]string, len(bs))
	for i, b := range bs {
		ss[i] = string(b)
	}
	return ss
}
