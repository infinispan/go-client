package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/testproto"
)

const advancedProtoSchema = `syntax = "proto3";
package test;

message Person {
  string name = 1;
  int32 age = 2;
}
`

// TestProtoStream_MultipleEntries tests multiple put/get operations with protostream
func TestProtoStream_MultipleEntries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-multiple")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "person.proto", advancedProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "proto-multiple",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	// Put multiple entries
	people := map[string]*testproto.Person{
		"alice": {Name: "Alice", Age: 30},
		"bob":   {Name: "Bob", Age: 25},
		"carol": {Name: "Carol", Age: 35},
		"dave":  {Name: "Dave", Age: 40},
		"eve":   {Name: "Eve", Age: 28},
	}

	for key, person := range people {
		if err := cache.Put(ctx, key, person); err != nil {
			t.Fatalf("Put %s: %v", key, err)
		}
	}

	// Get all entries and verify
	for key, expected := range people {
		person, found, err := cache.Get(ctx, key)
		if err != nil {
			t.Fatalf("Get %s: %v", key, err)
		}
		if !found {
			t.Fatalf("key %q not found", key)
		}
		if person.Name != expected.Name {
			t.Errorf("%s: Name = %q, want %q", key, person.Name, expected.Name)
		}
		if person.Age != expected.Age {
			t.Errorf("%s: Age = %d, want %d", key, person.Age, expected.Age)
		}
	}

	t.Logf("✓ Successfully stored and retrieved %d ProtoStream entries", len(people))
}

// TestProtoStream_UpdateOperations tests update operations with protostream
func TestProtoStream_UpdateOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-update")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "person.proto", advancedProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "proto-update",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	// Put initial value
	if err := cache.Put(ctx, "person1", &testproto.Person{Name: "Original", Age: 20}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Update (Put again)
	if err := cache.Put(ctx, "person1", &testproto.Person{Name: "Updated", Age: 21}); err != nil {
		t.Fatalf("Put (update): %v", err)
	}

	// Verify update
	person, found, err := cache.Get(ctx, "person1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected person to be found")
	}
	if person.Name != "Updated" {
		t.Errorf("Name = %q, want %q", person.Name, "Updated")
	}
	if person.Age != 21 {
		t.Errorf("Age = %d, want 21", person.Age)
	}

	t.Log("✓ Update operations with ProtoStream work correctly")
}

// TestProtoStream_RemoveOperations tests remove operations with protostream
func TestProtoStream_RemoveOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-remove")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "person.proto", advancedProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "proto-remove",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	// Put value
	if err := cache.Put(ctx, "person1", &testproto.Person{Name: "ToRemove", Age: 50}); err != nil {
		t.Fatalf("Put: %v", err)
	}

	// Remove
	if err := cache.Remove(ctx, "person1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Verify it's removed
	_, found, err := cache.Get(ctx, "person1")
	if err != nil {
		t.Fatalf("Get after remove: %v", err)
	}
	if found {
		t.Error("expected person to be removed")
	}

	t.Log("✓ Remove operations with ProtoStream work correctly")
}

// TestProtoStream_WithExpiration tests expiration with protostream
func TestProtoStream_WithExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-expiration")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "person.proto", advancedProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "proto-expiration",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	// Put with lifespan
	if err := cache.Put(ctx, "expires", &testproto.Person{Name: "Expires", Age: 1},
		hotrod.WithLifespan(1*time.Second)); err != nil {
		t.Fatalf("Put with lifespan: %v", err)
	}

	// Should exist immediately
	person, found, err := cache.Get(ctx, "expires")
	if err != nil {
		t.Fatalf("Get before expiry: %v", err)
	}
	if !found {
		t.Fatal("expected person before expiry")
	}
	if person.Name != "Expires" {
		t.Errorf("Name = %q, want %q", person.Name, "Expires")
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Should be gone
	_, found, err = cache.Get(ctx, "expires")
	if err != nil {
		t.Fatalf("Get after expiry: %v", err)
	}
	if found {
		t.Error("expected person to be expired")
	}

	t.Log("✓ Expiration with ProtoStream works correctly")
}

// TestProtoStream_ConcurrentOperations tests concurrent put/get with protostream
func TestProtoStream_ConcurrentOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-concurrent")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "person.proto", advancedProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Person](
		client, "proto-concurrent",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Person { return &testproto.Person{} },
	)

	// Put entries concurrently
	const numEntries = 10
	errChan := make(chan error, numEntries)

	for i := 0; i < numEntries; i++ {
		go func(id int) {
			key := fmt.Sprintf("person%d", id)
			person := &testproto.Person{
				Name: fmt.Sprintf("Person%d", id),
				Age:  int32(id * 10),
			}
			errChan <- cache.Put(ctx, key, person)
		}(i)
	}

	// Wait for all puts
	for i := 0; i < numEntries; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent put %d failed: %v", i, err)
		}
	}

	// Verify all entries
	for i := 0; i < numEntries; i++ {
		key := fmt.Sprintf("person%d", i)
		person, found, err := cache.Get(ctx, key)
		if err != nil {
			t.Errorf("Get %s: %v", key, err)
		}
		if !found {
			t.Errorf("Person %d not found", i)
		}
		expectedName := fmt.Sprintf("Person%d", i)
		if person.Name != expectedName {
			t.Errorf("%s: Name = %q, want %q", key, person.Name, expectedName)
		}
	}

	t.Logf("✓ Successfully completed %d concurrent ProtoStream operations", numEntries)
}

// TestProtoStream_LargeProtoMessages tests protobuf messages with many fields
func TestProtoStream_LargeProtoMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "proto-large")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Register complex schema
	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.User](
		client, "proto-large",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.User { return &testproto.User{} },
	)

	// Create user with many fields
	user := &testproto.User{
		Id:      1,
		Name:    "ComplexUser",
		Surname: "WithManyFields",
		Gender:  testproto.Gender_MALE,
		AccountIds: []int32{100, 200, 300, 400, 500},
		Addresses: []*testproto.Address{
			{Street: "123 Main St", City: "City1", State: "ST", Postcode: "12345", Country: "USA"},
			{Street: "456 Oak Ave", City: "City2", State: "ST", Postcode: "67890", Country: "USA"},
		},
		PhoneNumbers: []*testproto.PhoneNumber{
			{Number: "555-1234", Type: "mobile"},
			{Number: "555-5678", Type: "home"},
		},
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
		Preferences: map[string]int32{
			"pref1": 1,
			"pref2": 2,
		},
	}

	// Put
	if err := cache.Put(ctx, 1, user); err != nil {
		t.Fatalf("Put large proto: %v", err)
	}

	// Get
	retrieved, found, err := cache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get large proto: %v", err)
	}
	if !found {
		t.Fatal("expected user to be found")
	}

	// Verify all fields
	if retrieved.Name != "ComplexUser" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "ComplexUser")
	}
	if len(retrieved.AccountIds) != 5 {
		t.Errorf("len(AccountIds) = %d, want 5", len(retrieved.AccountIds))
	}
	if len(retrieved.Addresses) != 2 {
		t.Errorf("len(Addresses) = %d, want 2", len(retrieved.Addresses))
	}
	if len(retrieved.PhoneNumbers) != 2 {
		t.Errorf("len(PhoneNumbers) = %d, want 2", len(retrieved.PhoneNumbers))
	}
	if len(retrieved.Metadata) != 3 {
		t.Errorf("len(Metadata) = %d, want 3", len(retrieved.Metadata))
	}

	t.Log("✓ Large proto messages with ProtoStream work correctly")
}
