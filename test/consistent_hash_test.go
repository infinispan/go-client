package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/testproto"
)

func TestConsistentHashRouting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "hash-test")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("hash-test")

	// Put many keys to exercise hash-based routing
	numKeys := 100
	for i := 0; i < numKeys; i++ {
		key := []byte(fmt.Sprintf("hash-key-%d", i))
		val := []byte(fmt.Sprintf("value-%d", i))
		if err := cache.Put(ctx, key, val); err != nil {
			t.Fatalf("Put key %d: %v", i, err)
		}
	}

	// Read them all back
	for i := 0; i < numKeys; i++ {
		key := []byte(fmt.Sprintf("hash-key-%d", i))
		expected := fmt.Sprintf("value-%d", i)
		val, found, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get key %d: %v", i, err)
		}
		if !found {
			t.Fatalf("key %d not found", i)
		}
		if string(val) != expected {
			t.Errorf("key %d: got %q, want %q", i, string(val), expected)
		}
	}
}

func TestConsistentHashWithTypedCache(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "hash-typed")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "person.proto", personProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "hash-typed",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("person-%d", i)
		if err := cache.Put(ctx, key, &testproto.Person{Name: key, Age: int32(i)}); err != nil {
			t.Fatalf("Put %s: %v", key, err)
		}
	}

	for i := 0; i < 50; i++ {
		key := fmt.Sprintf("person-%d", i)
		person, found, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get %s: %v", key, err)
		}
		if !found {
			t.Fatalf("%s not found", key)
		}
		if person.Name != key {
			t.Errorf("Name = %q, want %q", person.Name, key)
		}
		if person.Age != int32(i) {
			t.Errorf("Age = %d, want %d", person.Age, i)
		}
	}
}

func TestConsistentHashWithLifespan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "hash-ttl")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("hash-ttl")

	if err := cache.Put(ctx, []byte("ttl-key"), []byte("expires"), hotrod.WithLifespan(1*time.Second)); err != nil {
		t.Fatalf("Put: %v", err)
	}

	val, found, err := cache.Get(ctx, []byte("ttl-key"))
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if !found || string(val) != "expires" {
		t.Fatalf("expected found=true val=%q, got found=%v val=%q", "expires", found, string(val))
	}

	time.Sleep(2 * time.Second)

	_, found, err = cache.Get(ctx, []byte("ttl-key"))
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected key expired")
	}
}
