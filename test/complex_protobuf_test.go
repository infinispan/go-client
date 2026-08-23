package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/testproto"
)

const complexProtoSchema = `syntax = "proto3";
package test;

enum Gender {
  MALE = 0;
  FEMALE = 1;
  OTHER = 2;
}

enum AccountType {
  CHECKING = 0;
  SAVINGS = 1;
  INVESTMENT = 2;
}

message Address {
  string street = 1;
  string city = 2;
  string state = 3;
  string postcode = 4;
  string country = 5;
}

message PhoneNumber {
  string number = 1;
  string type = 2;
}

message User {
  int32 id = 1;
  string name = 2;
  string surname = 3;
  Gender gender = 4;
  repeated int32 account_ids = 5;
  repeated Address addresses = 6;
  repeated PhoneNumber phone_numbers = 7;
  map<string, string> metadata = 8;
  map<string, int32> preferences = 9;
  optional string nickname = 10;
  optional int32 age = 11;
  optional string email = 12;
}

message Company {
  string name = 1;
  repeated User employees = 2;
  Address headquarters = 3;
  map<string, string> departments = 4;
}

message Contact {
  string name = 1;
  oneof contact_method {
    string email = 2;
    string phone = 3;
    Address address = 4;
  }
}

message Configuration {
  string name = 1;
  map<string, string> string_values = 2;
  map<string, int32> int_values = 3;
  map<string, bool> bool_values = 4;
}

message Collection {
  repeated string strings = 1;
  repeated int32 int32s = 2;
  repeated int64 int64s = 3;
  repeated bool bools = 4;
  repeated double doubles = 5;
  repeated Gender enums = 6;
}
`

// TestComplexProtobuf_NestedMessages tests nested message structures
// Based on ClientProtoStreamMarshallerTest.java
func TestComplexProtobuf_NestedMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "complex-nested")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	// Register schema
	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.User](
		client, "complex-nested",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.User { return &testproto.User{} },
	)

	// Create user with nested addresses
	user := &testproto.User{
		Id:      1,
		Name:    "John",
		Surname: "Doe",
		Gender:  testproto.Gender_MALE,
		Addresses: []*testproto.Address{
			{
				Street:   "123 Main St",
				City:     "Springfield",
				State:    "IL",
				Postcode: "62701",
				Country:  "USA",
			},
			{
				Street:   "456 Oak Ave",
				City:     "Portland",
				State:    "OR",
				Postcode: "97201",
				Country:  "USA",
			},
		},
	}

	// Put
	if err := cache.Put(ctx, 1, user); err != nil {
		t.Fatalf("Put user: %v", err)
	}

	// Get
	retrieved, found, err := cache.Get(ctx, 1)
	if err != nil {
		t.Fatalf("Get user: %v", err)
	}
	if !found {
		t.Fatal("expected user to be found")
	}

	// Verify nested structure
	if retrieved.Id != 1 {
		t.Errorf("Id = %d, want 1", retrieved.Id)
	}
	if retrieved.Name != "John" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "John")
	}
	if retrieved.Surname != "Doe" {
		t.Errorf("Surname = %q, want %q", retrieved.Surname, "Doe")
	}
	if retrieved.Gender != testproto.Gender_MALE {
		t.Errorf("Gender = %v, want MALE", retrieved.Gender)
	}

	// Verify addresses
	if len(retrieved.Addresses) != 2 {
		t.Fatalf("len(Addresses) = %d, want 2", len(retrieved.Addresses))
	}
	if retrieved.Addresses[0].Street != "123 Main St" {
		t.Errorf("Address[0].Street = %q, want %q", retrieved.Addresses[0].Street, "123 Main St")
	}
	if retrieved.Addresses[0].City != "Springfield" {
		t.Errorf("Address[0].City = %q, want %q", retrieved.Addresses[0].City, "Springfield")
	}
	if retrieved.Addresses[1].Street != "456 Oak Ave" {
		t.Errorf("Address[1].Street = %q, want %q", retrieved.Addresses[1].Street, "456 Oak Ave")
	}

	t.Log("✓ Nested messages (User with Addresses) work correctly")
}

// TestComplexProtobuf_RepeatedFields tests repeated scalar and message fields
func TestComplexProtobuf_RepeatedFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "complex-repeated")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.User](
		client, "complex-repeated",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.User { return &testproto.User{} },
	)

	user := &testproto.User{
		Id:         2,
		Name:       "Jane",
		Surname:    "Smith",
		Gender:     testproto.Gender_FEMALE,
		AccountIds: []int32{100, 200, 300, 400},
		PhoneNumbers: []*testproto.PhoneNumber{
			{Number: "555-1234", Type: "mobile"},
			{Number: "555-5678", Type: "home"},
			{Number: "555-9012", Type: "work"},
		},
	}

	if err := cache.Put(ctx, 2, user); err != nil {
		t.Fatalf("Put: %v", err)
	}

	retrieved, found, err := cache.Get(ctx, 2)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected user to be found")
	}

	// Verify repeated int32 field
	if len(retrieved.AccountIds) != 4 {
		t.Fatalf("len(AccountIds) = %d, want 4", len(retrieved.AccountIds))
	}
	for i, expected := range []int32{100, 200, 300, 400} {
		if retrieved.AccountIds[i] != expected {
			t.Errorf("AccountIds[%d] = %d, want %d", i, retrieved.AccountIds[i], expected)
		}
	}

	// Verify repeated message field
	if len(retrieved.PhoneNumbers) != 3 {
		t.Fatalf("len(PhoneNumbers) = %d, want 3", len(retrieved.PhoneNumbers))
	}
	if retrieved.PhoneNumbers[0].Number != "555-1234" {
		t.Errorf("PhoneNumbers[0].Number = %q, want %q", retrieved.PhoneNumbers[0].Number, "555-1234")
	}
	if retrieved.PhoneNumbers[0].Type != "mobile" {
		t.Errorf("PhoneNumbers[0].Type = %q, want %q", retrieved.PhoneNumbers[0].Type, "mobile")
	}

	t.Log("✓ Repeated fields (scalars and messages) work correctly")
}

// TestComplexProtobuf_MapFields tests map fields with various value types
func TestComplexProtobuf_MapFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "complex-maps")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.User](
		client, "complex-maps",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.User { return &testproto.User{} },
	)

	user := &testproto.User{
		Id:      3,
		Name:    "Bob",
		Surname: "Johnson",
		Gender:  testproto.Gender_MALE,
		Metadata: map[string]string{
			"department": "Engineering",
			"location":   "San Francisco",
			"level":      "Senior",
		},
		Preferences: map[string]int32{
			"theme":         1,
			"notifications": 0,
			"fontSize":      14,
		},
	}

	if err := cache.Put(ctx, 3, user); err != nil {
		t.Fatalf("Put: %v", err)
	}

	retrieved, found, err := cache.Get(ctx, 3)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected user to be found")
	}

	// Verify map<string, string>
	if len(retrieved.Metadata) != 3 {
		t.Fatalf("len(Metadata) = %d, want 3", len(retrieved.Metadata))
	}
	if retrieved.Metadata["department"] != "Engineering" {
		t.Errorf("Metadata[department] = %q, want %q", retrieved.Metadata["department"], "Engineering")
	}
	if retrieved.Metadata["location"] != "San Francisco" {
		t.Errorf("Metadata[location] = %q, want %q", retrieved.Metadata["location"], "San Francisco")
	}

	// Verify map<string, int32>
	if len(retrieved.Preferences) != 3 {
		t.Fatalf("len(Preferences) = %d, want 3", len(retrieved.Preferences))
	}
	if retrieved.Preferences["theme"] != 1 {
		t.Errorf("Preferences[theme] = %d, want 1", retrieved.Preferences["theme"])
	}
	if retrieved.Preferences["fontSize"] != 14 {
		t.Errorf("Preferences[fontSize] = %d, want 14", retrieved.Preferences["fontSize"])
	}

	t.Log("✓ Map fields (string→string and string→int32) work correctly")
}

// TestComplexProtobuf_EnumFields tests enum field handling
func TestComplexProtobuf_EnumFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "complex-enums")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.User](
		client, "complex-enums",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.User { return &testproto.User{} },
	)

	// Test all enum values
	users := []*testproto.User{
		{Id: 10, Name: "Male", Surname: "User", Gender: testproto.Gender_MALE},
		{Id: 11, Name: "Female", Surname: "User", Gender: testproto.Gender_FEMALE},
		{Id: 12, Name: "Other", Surname: "User", Gender: testproto.Gender_OTHER},
	}

	for _, user := range users {
		if err := cache.Put(ctx, user.Id, user); err != nil {
			t.Fatalf("Put user %d: %v", user.Id, err)
		}
	}

	// Verify each enum value
	for _, expected := range users {
		retrieved, found, err := cache.Get(ctx, expected.Id)
		if err != nil {
			t.Fatalf("Get user %d: %v", expected.Id, err)
		}
		if !found {
			t.Fatalf("User %d not found", expected.Id)
		}
		if retrieved.Gender != expected.Gender {
			t.Errorf("User %d: Gender = %v, want %v", expected.Id, retrieved.Gender, expected.Gender)
		}
	}

	t.Log("✓ Enum fields (MALE, FEMALE, OTHER) work correctly")
}

// TestComplexProtobuf_OptionalFields tests optional field handling (proto3)
func TestComplexProtobuf_OptionalFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "complex-optional")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.User](
		client, "complex-optional",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.User { return &testproto.User{} },
	)

	// User with optional fields set
	nickname := "Johnny"
	age := int32(30)
	email := "john@example.com"

	userWithOptionals := &testproto.User{
		Id:       20,
		Name:     "John",
		Surname:  "Doe",
		Gender:   testproto.Gender_MALE,
		Nickname: &nickname,
		Age:      &age,
		Email:    &email,
	}

	if err := cache.Put(ctx, 20, userWithOptionals); err != nil {
		t.Fatalf("Put user with optionals: %v", err)
	}

	retrieved, found, err := cache.Get(ctx, 20)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected user to be found")
	}

	// Verify optional fields are set
	if retrieved.Nickname == nil {
		t.Error("Nickname should not be nil")
	} else if *retrieved.Nickname != "Johnny" {
		t.Errorf("Nickname = %q, want %q", *retrieved.Nickname, "Johnny")
	}

	if retrieved.Age == nil {
		t.Error("Age should not be nil")
	} else if *retrieved.Age != 30 {
		t.Errorf("Age = %d, want 30", *retrieved.Age)
	}

	if retrieved.Email == nil {
		t.Error("Email should not be nil")
	} else if *retrieved.Email != "john@example.com" {
		t.Errorf("Email = %q, want %q", *retrieved.Email, "john@example.com")
	}

	// User without optional fields
	userWithoutOptionals := &testproto.User{
		Id:      21,
		Name:    "Jane",
		Surname: "Smith",
		Gender:  testproto.Gender_FEMALE,
		// No optional fields set
	}

	if err := cache.Put(ctx, 21, userWithoutOptionals); err != nil {
		t.Fatalf("Put user without optionals: %v", err)
	}

	retrieved2, found, err := cache.Get(ctx, 21)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected user to be found")
	}

	// Verify optional fields are nil
	if retrieved2.Nickname != nil {
		t.Errorf("Nickname should be nil, got %q", *retrieved2.Nickname)
	}
	if retrieved2.Age != nil {
		t.Errorf("Age should be nil, got %d", *retrieved2.Age)
	}
	if retrieved2.Email != nil {
		t.Errorf("Email should be nil, got %q", *retrieved2.Email)
	}

	t.Log("✓ Optional fields (present and absent) work correctly")
}

// TestComplexProtobuf_DeeplyNested tests deeply nested structures
func TestComplexProtobuf_DeeplyNested(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "complex-deep")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Company](
		client, "complex-deep",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Company { return &testproto.Company{} },
	)

	// Company with employees (nested Users with nested Addresses)
	company := &testproto.Company{
		Name: "Tech Corp",
		Headquarters: &testproto.Address{
			Street:   "1 Tech Way",
			City:     "San Francisco",
			State:    "CA",
			Postcode: "94105",
			Country:  "USA",
		},
		Employees: []*testproto.User{
			{
				Id:      100,
				Name:    "Alice",
				Surname: "Anderson",
				Gender:  testproto.Gender_FEMALE,
				Addresses: []*testproto.Address{
					{Street: "10 Main St", City: "Oakland", State: "CA", Postcode: "94601", Country: "USA"},
				},
				Metadata: map[string]string{
					"role": "CEO",
				},
			},
			{
				Id:      101,
				Name:    "Bob",
				Surname: "Brown",
				Gender:  testproto.Gender_MALE,
				Addresses: []*testproto.Address{
					{Street: "20 Oak Ave", City: "Berkeley", State: "CA", Postcode: "94701", Country: "USA"},
				},
				Metadata: map[string]string{
					"role": "CTO",
				},
			},
		},
		Departments: map[string]string{
			"engineering": "Building 1",
			"sales":       "Building 2",
			"hr":          "Building 3",
		},
	}

	if err := cache.Put(ctx, "tech-corp", company); err != nil {
		t.Fatalf("Put company: %v", err)
	}

	retrieved, found, err := cache.Get(ctx, "tech-corp")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected company to be found")
	}

	// Verify deeply nested structure
	if retrieved.Name != "Tech Corp" {
		t.Errorf("Name = %q, want %q", retrieved.Name, "Tech Corp")
	}

	if retrieved.Headquarters.City != "San Francisco" {
		t.Errorf("Headquarters.City = %q, want %q", retrieved.Headquarters.City, "San Francisco")
	}

	if len(retrieved.Employees) != 2 {
		t.Fatalf("len(Employees) = %d, want 2", len(retrieved.Employees))
	}

	if retrieved.Employees[0].Name != "Alice" {
		t.Errorf("Employees[0].Name = %q, want %q", retrieved.Employees[0].Name, "Alice")
	}

	if len(retrieved.Employees[0].Addresses) != 1 {
		t.Fatalf("len(Employees[0].Addresses) = %d, want 1", len(retrieved.Employees[0].Addresses))
	}

	if retrieved.Employees[0].Addresses[0].City != "Oakland" {
		t.Errorf("Employees[0].Addresses[0].City = %q, want %q",
			retrieved.Employees[0].Addresses[0].City, "Oakland")
	}

	if retrieved.Employees[0].Metadata["role"] != "CEO" {
		t.Errorf("Employees[0].Metadata[role] = %q, want %q",
			retrieved.Employees[0].Metadata["role"], "CEO")
	}

	if len(retrieved.Departments) != 3 {
		t.Fatalf("len(Departments) = %d, want 3", len(retrieved.Departments))
	}

	t.Log("✓ Deeply nested structures (Company→Users→Addresses) work correctly")
}

// TestComplexProtobuf_OneofFields tests oneof field handling
func TestComplexProtobuf_OneofFields(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "complex-oneof")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[string, *testproto.Contact](
		client, "complex-oneof",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.Contact { return &testproto.Contact{} },
	)

	// Contact with email (oneof)
	contactEmail := &testproto.Contact{
		Name: "Contact1",
		ContactMethod: &testproto.Contact_Email{
			Email: "contact1@example.com",
		},
	}

	if err := cache.Put(ctx, "contact1", contactEmail); err != nil {
		t.Fatalf("Put contact with email: %v", err)
	}

	retrieved1, found, err := cache.Get(ctx, "contact1")
	if err != nil {
		t.Fatalf("Get contact1: %v", err)
	}
	if !found {
		t.Fatal("expected contact1 to be found")
	}

	if email, ok := retrieved1.ContactMethod.(*testproto.Contact_Email); ok {
		if email.Email != "contact1@example.com" {
			t.Errorf("Email = %q, want %q", email.Email, "contact1@example.com")
		}
	} else {
		t.Errorf("ContactMethod is not email: %T", retrieved1.ContactMethod)
	}

	// Contact with phone (oneof)
	contactPhone := &testproto.Contact{
		Name: "Contact2",
		ContactMethod: &testproto.Contact_Phone{
			Phone: "555-1234",
		},
	}

	if err := cache.Put(ctx, "contact2", contactPhone); err != nil {
		t.Fatalf("Put contact with phone: %v", err)
	}

	retrieved2, found, err := cache.Get(ctx, "contact2")
	if err != nil {
		t.Fatalf("Get contact2: %v", err)
	}
	if !found {
		t.Fatal("expected contact2 to be found")
	}

	if phone, ok := retrieved2.ContactMethod.(*testproto.Contact_Phone); ok {
		if phone.Phone != "555-1234" {
			t.Errorf("Phone = %q, want %q", phone.Phone, "555-1234")
		}
	} else {
		t.Errorf("ContactMethod is not phone: %T", retrieved2.ContactMethod)
	}

	// Contact with address (oneof)
	contactAddress := &testproto.Contact{
		Name: "Contact3",
		ContactMethod: &testproto.Contact_Address{
			Address: &testproto.Address{
				Street:   "789 Elm St",
				City:     "Boston",
				State:    "MA",
				Postcode: "02101",
				Country:  "USA",
			},
		},
	}

	if err := cache.Put(ctx, "contact3", contactAddress); err != nil {
		t.Fatalf("Put contact with address: %v", err)
	}

	retrieved3, found, err := cache.Get(ctx, "contact3")
	if err != nil {
		t.Fatalf("Get contact3: %v", err)
	}
	if !found {
		t.Fatal("expected contact3 to be found")
	}

	if addr, ok := retrieved3.ContactMethod.(*testproto.Contact_Address); ok {
		if addr.Address.City != "Boston" {
			t.Errorf("Address.City = %q, want %q", addr.Address.City, "Boston")
		}
	} else {
		t.Errorf("ContactMethod is not address: %T", retrieved3.ContactMethod)
	}

	t.Log("✓ Oneof fields (email, phone, address) work correctly")
}

// TestComplexProtobuf_EmptyCollections tests empty repeated fields and maps
func TestComplexProtobuf_EmptyCollections(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "complex-empty")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	if err := client.Schemas().Register(ctx, "complex.proto", complexProtoSchema); err != nil {
		t.Fatalf("Register schema: %v", err)
	}

	cache := hotrod.NewTypedCache[int32, *testproto.User](
		client, "complex-empty",
		&hotrod.ProtoStreamMarshaller{},
		func() *testproto.User { return &testproto.User{} },
	)

	// User with empty collections
	user := &testproto.User{
		Id:           30,
		Name:         "Empty",
		Surname:      "User",
		Gender:       testproto.Gender_OTHER,
		AccountIds:   []int32{},              // Empty slice
		Addresses:    []*testproto.Address{}, // Empty slice
		PhoneNumbers: nil,                    // Nil slice
		Metadata:     map[string]string{},    // Empty map
		Preferences:  nil,                    // Nil map
	}

	if err := cache.Put(ctx, 30, user); err != nil {
		t.Fatalf("Put user with empty collections: %v", err)
	}

	retrieved, found, err := cache.Get(ctx, 30)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !found {
		t.Fatal("expected user to be found")
	}

	// Verify empty collections
	if len(retrieved.AccountIds) != 0 {
		t.Errorf("len(AccountIds) = %d, want 0", len(retrieved.AccountIds))
	}
	if len(retrieved.Addresses) != 0 {
		t.Errorf("len(Addresses) = %d, want 0", len(retrieved.Addresses))
	}
	if len(retrieved.PhoneNumbers) != 0 {
		t.Errorf("len(PhoneNumbers) = %d, want 0", len(retrieved.PhoneNumbers))
	}
	if len(retrieved.Metadata) != 0 {
		t.Errorf("len(Metadata) = %d, want 0", len(retrieved.Metadata))
	}
	if len(retrieved.Preferences) != 0 {
		t.Errorf("len(Preferences) = %d, want 0", len(retrieved.Preferences))
	}

	t.Log("✓ Empty collections (slices and maps) work correctly")
}
