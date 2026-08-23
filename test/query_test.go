package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/testproto"
	"google.golang.org/protobuf/proto"
)

func TestQuery_BasicIckle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	registerSchema(t, client, "person.proto", indexedPersonProtoSchema)
	createIndexedCache(t, sharedContainer, "q-basic")

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "q-basic",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	persons := []*testproto.Person{
		{Name: "Alice", Age: 30},
		{Name: "Bob", Age: 25},
		{Name: "Charlie", Age: 35},
	}
	for _, p := range persons {
		if err := cache.Put(ctx, p.Name, p); err != nil {
			t.Fatalf("Put %s: %v", p.Name, err)
		}
	}

	rawCache := client.Cache("q-basic")
	result, err := rawCache.Query(ctx, "FROM test.Person WHERE age >= 30")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if result.NumResults != 2 {
		t.Errorf("NumResults = %d, want 2", result.NumResults)
	}
	if len(result.Entries) != 2 {
		t.Fatalf("len(Entries) = %d, want 2", len(result.Entries))
	}

	names := make(map[string]bool)
	for _, entry := range result.Entries {
		if entry.TypeName != "test.Person" {
			t.Errorf("TypeName = %q, want %q", entry.TypeName, "test.Person")
		}
		p := &testproto.Person{}
		if err := proto.Unmarshal(entry.Value, p); err != nil {
			t.Fatalf("Unmarshal: %v", err)
		}
		names[p.Name] = true
	}
	if !names["Alice"] {
		t.Error("expected Alice in results")
	}
	if !names["Charlie"] {
		t.Error("expected Charlie in results")
	}
}

func TestQuery_WithParams(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	registerSchema(t, client, "person.proto", indexedPersonProtoSchema)
	createIndexedCache(t, sharedContainer, "q-params")

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "q-params",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	for _, p := range []*testproto.Person{
		{Name: "Alice", Age: 20},
		{Name: "Bob", Age: 30},
		{Name: "Charlie", Age: 40},
	} {
		if err := cache.Put(ctx, p.Name, p); err != nil {
			t.Fatalf("Put %s: %v", p.Name, err)
		}
	}

	rawCache := client.Cache("q-params")
	result, err := rawCache.Query(ctx, "FROM test.Person WHERE age > :minAge",
		hotrod.WithQueryParam("minAge", int32(25)),
	)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}

	if result.NumResults != 2 {
		t.Errorf("NumResults = %d, want 2", result.NumResults)
	}
}

func TestQuery_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	registerSchema(t, client, "person.proto", indexedPersonProtoSchema)
	createIndexedCache(t, sharedContainer, "q-page")

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "q-page",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	for i := 0; i < 5; i++ {
		p := &testproto.Person{Name: fmt.Sprintf("Person%d", i), Age: int32(20 + i)}
		if err := cache.Put(ctx, p.Name, p); err != nil {
			t.Fatalf("Put: %v", err)
		}
	}

	rawCache := client.Cache("q-page")

	// First page: 2 results
	result, err := rawCache.Query(ctx, "FROM test.Person ORDER BY age",
		hotrod.WithQueryMaxResults(2),
		hotrod.WithQueryStartOffset(0),
	)
	if err != nil {
		t.Fatalf("Query page 1: %v", err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("page 1: len(Entries) = %d, want 2", len(result.Entries))
	}

	// Second page: next 2 results
	result2, err := rawCache.Query(ctx, "FROM test.Person ORDER BY age",
		hotrod.WithQueryMaxResults(2),
		hotrod.WithQueryStartOffset(2),
	)
	if err != nil {
		t.Fatalf("Query page 2: %v", err)
	}
	if len(result2.Entries) != 2 {
		t.Errorf("page 2: len(Entries) = %d, want 2", len(result2.Entries))
	}

	// Third page: last 1 result
	result3, err := rawCache.Query(ctx, "FROM test.Person ORDER BY age",
		hotrod.WithQueryMaxResults(2),
		hotrod.WithQueryStartOffset(4),
	)
	if err != nil {
		t.Fatalf("Query page 3: %v", err)
	}
	if len(result3.Entries) != 1 {
		t.Errorf("page 3: len(Entries) = %d, want 1", len(result3.Entries))
	}
}
