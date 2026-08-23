package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/testproto"
)

const personProtoSchema = `syntax = "proto3";
package test;

message Person {
  string name = 1;
  int32 age = 2;
}
`

func TestProtoStreamPutGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-test")

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
		client, "proto-test",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	if err := cache.Put(ctx, "john", &testproto.Person{Name: "John", Age: 30}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	person, found, err := cache.Get(ctx, "john")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found")
	}
	if person.Name != "John" {
		t.Errorf("Name = %q, want %q", person.Name, "John")
	}
	if person.Age != 30 {
		t.Errorf("Age = %d, want %d", person.Age, 30)
	}
}

func TestProtoStreamKeyNotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-miss")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "proto-miss",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	_, found, err := cache.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if found {
		t.Error("expected key not found")
	}
}

func TestProtoStreamWithLifespan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-ttl")

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
		client, "proto-ttl",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	if err := cache.Put(ctx, "ephemeral", &testproto.Person{Name: "Temp", Age: 1}, hotrod.WithLifespan(1*time.Second)); err != nil {
		t.Fatalf("Put with lifespan: %v", err)
	}

	person, found, err := cache.Get(ctx, "ephemeral")
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if !found || person.Name != "Temp" {
		t.Fatalf("expected Temp before expiry, got found=%v name=%q", found, person.Name)
	}

	time.Sleep(2 * time.Second)

	_, found, err = cache.Get(ctx, "ephemeral")
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected key to be expired")
	}
}

func TestSchemaRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	schemas := client.Schemas()

	if err := schemas.Register(ctx, "test-schema.proto", personProtoSchema); err != nil {
		t.Fatalf("Register: %v", err)
	}

	content, found, err := schemas.Get(ctx, "test-schema.proto")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected schema to be found")
	}
	if content != personProtoSchema {
		t.Errorf("schema content mismatch:\ngot:  %q\nwant: %q", content, personProtoSchema)
	}
}
