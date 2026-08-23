package hotrod_test

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/protostream"
	"infinispan.org/go-client/internal/testproto"
	"github.com/testcontainers/testcontainers-go"
)

const indexedPersonProtoSchema = `syntax = "proto3"; package test; /** @Indexed */ message Person { /** @Field(store = Store.YES) */ string name = 1; /** @Field(store = Store.YES) */ int32 age = 2; }`

func registerSchema(t *testing.T, client *hotrod.Client, name, schema string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Schemas().Register(ctx, name, schema); err != nil {
		t.Fatalf("register schema %s: %v", name, err)
	}
}

func createIndexedCache(t *testing.T, container testcontainers.Container, name string) {
	t.Helper()
	ctx := context.Background()
	cacheConfig := `{"distributed-cache":{"mode":"SYNC","encoding":{"media-type":"application/x-protostream"},"indexing":{"enabled":true,"storage":"local-heap","indexed-entities":["test.Person"]}}}`
	_, output, err := container.Exec(ctx, []string{
		"bash", "-c",
		fmt.Sprintf("curl -s -o /dev/null -w '%%{http_code}' -X POST --digest -u admin:password -H 'Content-Type: application/json' http://localhost:11222/rest/v2/caches/%s -d '%s'", name, cacheConfig),
	})
	if err != nil {
		t.Fatalf("exec create indexed cache: %v", err)
	}
	body, _ := io.ReadAll(output)
	status := strings.TrimSpace(string(body))
	if len(status) >= 3 {
		status = status[len(status)-3:]
	}
	t.Logf("create indexed cache %s: HTTP %s", name, status)
	if status != "200" {
		t.Fatalf("failed to create indexed cache: HTTP %s", status)
	}
}

func TestContinuousQuery_JoiningAndLeaving(t *testing.T) {
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
	createIndexedCache(t, sharedContainer, "cq-test")

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "cq-test",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)
	rawCache := client.Cache("cq-test")

	cq, err := rawCache.ContinuousQuery(ctx, "FROM test.Person WHERE age >= 18")
	if err != nil {
		t.Fatalf("ContinuousQuery: %v", err)
	}
	defer func() { _ = rawCache.RemoveContinuousQuery(ctx, cq) }()

	// Put a matching entry → expect JOINING
	if err := cache.Put(ctx, "alice", &testproto.Person{Name: "Alice", Age: 25}); err != nil {
		t.Fatalf("Put alice: %v", err)
	}
	ev := expectCQEvent(t, cq, 5*time.Second)
	if ev.Type != hotrod.CQJoining {
		t.Errorf("expected CQJoining, got %d", ev.Type)
	}
	key, err := protostream.UnwrapString(ev.Key)
	if err != nil {
		t.Fatalf("UnwrapString key: %v", err)
	}
	if key != "alice" {
		t.Errorf("key = %q, want %q", key, "alice")
	}
	if ev.Value == nil {
		t.Error("expected non-nil value for JOINING event")
	}

	// Update entry to not match → expect LEAVING
	if err := cache.Put(ctx, "alice", &testproto.Person{Name: "Alice", Age: 10}); err != nil {
		t.Fatalf("Put alice (age=10): %v", err)
	}
	ev = expectCQEvent(t, cq, 5*time.Second)
	if ev.Type != hotrod.CQLeaving {
		t.Errorf("expected CQLeaving, got %d", ev.Type)
	}
}

func TestContinuousQuery_NonMatchingEntryNoEvent(t *testing.T) {
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
	createIndexedCache(t, sharedContainer, "cq-nomatch")

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "cq-nomatch",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)
	rawCache := client.Cache("cq-nomatch")

	cq, err := rawCache.ContinuousQuery(ctx, "FROM test.Person WHERE age >= 18")
	if err != nil {
		t.Fatalf("ContinuousQuery: %v", err)
	}
	defer func() { _ = rawCache.RemoveContinuousQuery(ctx, cq) }()

	// Put a non-matching entry → should NOT receive an event
	if err := cache.Put(ctx, "child", &testproto.Person{Name: "Child", Age: 10}); err != nil {
		t.Fatalf("Put child: %v", err)
	}
	expectNoCQEvent(t, cq, 2*time.Second)
}

func TestContinuousQuery_WithParam(t *testing.T) {
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
	createIndexedCache(t, sharedContainer, "cq-param")

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "cq-param",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)
	rawCache := client.Cache("cq-param")

	cq, err := rawCache.ContinuousQuery(ctx, "FROM test.Person WHERE age >= :minAge",
		hotrod.WithCQParam("minAge", int32(21)),
	)
	if err != nil {
		t.Fatalf("ContinuousQuery: %v", err)
	}
	defer func() { _ = rawCache.RemoveContinuousQuery(ctx, cq) }()

	// Put entry that doesn't match (age < 21)
	if err := cache.Put(ctx, "teen", &testproto.Person{Name: "Teen", Age: 18}); err != nil {
		t.Fatalf("Put teen: %v", err)
	}
	expectNoCQEvent(t, cq, 2*time.Second)

	// Put entry that matches (age >= 21)
	if err := cache.Put(ctx, "adult", &testproto.Person{Name: "Adult", Age: 30}); err != nil {
		t.Fatalf("Put adult: %v", err)
	}
	ev := expectCQEvent(t, cq, 5*time.Second)
	if ev.Type != hotrod.CQJoining {
		t.Errorf("expected CQJoining, got %d", ev.Type)
	}
}

func expectCQEvent(t *testing.T, cq *hotrod.ContinuousQuery, timeout time.Duration) *hotrod.CQEvent {
	t.Helper()
	select {
	case ev, ok := <-cq.Events:
		if !ok {
			t.Fatal("CQ event channel closed unexpectedly")
		}
		return ev
	case <-time.After(timeout):
		t.Fatal("timed out waiting for CQ event")
		return nil
	}
}

func expectNoCQEvent(t *testing.T, cq *hotrod.ContinuousQuery, wait time.Duration) {
	t.Helper()
	select {
	case ev := <-cq.Events:
		t.Fatalf("expected no CQ event, got type=%d", ev.Type)
	case <-time.After(wait):
		// good
	}
}
