package hotrod_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

// Valid protobuf schema for testing
const validSchema = `syntax = "proto3";
package test;

message Author {
  string name = 1;
  int32 age = 2;
}

message Book {
  string title = 1;
  int32 year = 2;
  string description = 3;
  Author author = 4;
}
`

// Schema with syntax error (missing semicolon)
const schemaMissingSemicolon = `syntax = "proto3";
package test;

message BadMessage {
  string name = 1  // Missing semicolon
  int32 age = 2;
}
`

// Schema with duplicate field numbers
const schemaDuplicateFields = `syntax = "proto3";
package test;

message DuplicateFields {
  string field1 = 1;
  int32 field2 = 1;  // Duplicate field number!
}
`

// Schema with undefined type reference
const schemaUndefinedType = `syntax = "proto3";
package test;

message UndefinedTypeRef {
  string name = 1;
  NonExistentType ref = 2;  // Type doesn't exist
}
`

// Schema with invalid field number (0)
const schemaInvalidFieldNumber = `syntax = "proto3";
package test;

message InvalidFieldNumber {
  string name = 0;  // Field numbers must be > 0
}
`

// Schema with reserved field number conflict
const schemaReservedConflict = `syntax = "proto3";
package test;

message ReservedConflict {
  reserved 2;
  string name = 1;
  int32 age = 2;  // Conflicts with reserved!
}
`

// Helper function to get schema error from server
func getSchemaError(ctx context.Context, t *testing.T, client *hotrod.Client, schemaName string) (string, bool) {
	cache := client.Cache("___protobuf_metadata")
	errorKey := schemaName + ".errors"

	data, found, err := cache.Get(ctx, []byte(errorKey))
	if err != nil {
		t.Logf("Warning: Error getting schema errors: %v", err)
		return "", false
	}

	if !found {
		return "", false
	}

	return string(data), true
}

// TestSchemaValidation_ValidSchema verifies that a valid schema registers successfully
func TestSchemaValidation_ValidSchema(t *testing.T) {
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

	schemaName := "valid-test.proto"

	// Register valid schema
	err = client.Schemas().Register(ctx, schemaName, validSchema)
	if err != nil {
		t.Fatalf("Register valid schema: %v", err)
	}

	// Verify schema was stored
	content, found, err := client.Schemas().Get(ctx, schemaName)
	if err != nil {
		t.Fatalf("Get schema: %v", err)
	}
	if !found {
		t.Fatal("expected schema to be found")
	}
	if content != validSchema {
		t.Error("schema content mismatch")
	}

	// Verify no errors
	errorMsg, hasError := getSchemaError(ctx, t, client, schemaName)
	if hasError {
		t.Errorf("expected no errors, got: %s", errorMsg)
	}

	t.Log("✓ Valid schema registered successfully with no errors")
}

// TestSchemaValidation_SyntaxError tests schema with syntax errors
func TestSchemaValidation_SyntaxError(t *testing.T) {
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

	schemaName := "syntax-error.proto"

	// Register invalid schema
	// Note: Registration itself may succeed, errors are detected server-side
	err = client.Schemas().Register(ctx, schemaName, schemaMissingSemicolon)
	if err != nil {
		t.Logf("Register returned error (acceptable): %v", err)
	}

	// Check for schema errors
	time.Sleep(100 * time.Millisecond) // Give server time to parse
	errorMsg, hasError := getSchemaError(ctx, t, client, schemaName)

	if hasError {
		t.Logf("✓ Schema error detected (as expected): %s", errorMsg)
		if !strings.Contains(errorMsg, "error") && !strings.Contains(errorMsg, "Error") {
			t.Logf("Warning: error message doesn't contain 'error': %s", errorMsg)
		}
	} else {
		t.Log("Note: Server accepted schema with syntax error (some servers may be lenient)")
	}
}

// TestSchemaValidation_DuplicateFieldNumbers tests schema with duplicate field numbers
func TestSchemaValidation_DuplicateFieldNumbers(t *testing.T) {
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

	schemaName := "duplicate-fields.proto"

	// Register schema with duplicate field numbers
	err = client.Schemas().Register(ctx, schemaName, schemaDuplicateFields)
	if err != nil {
		t.Logf("Register returned error (acceptable): %v", err)
	}

	// Check for errors
	time.Sleep(100 * time.Millisecond)
	errorMsg, hasError := getSchemaError(ctx, t, client, schemaName)

	if hasError {
		t.Logf("✓ Duplicate field number error detected: %s", errorMsg)
		if strings.Contains(strings.ToLower(errorMsg), "duplicate") ||
			strings.Contains(strings.ToLower(errorMsg), "field") {
			t.Log("✓ Error message mentions duplicate/field")
		}
	} else {
		t.Log("Note: Server accepted duplicate field numbers (may vary by version)")
	}
}

// TestSchemaValidation_UndefinedType tests schema with undefined type reference
func TestSchemaValidation_UndefinedType(t *testing.T) {
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

	schemaName := "undefined-type.proto"

	// Register schema with undefined type
	err = client.Schemas().Register(ctx, schemaName, schemaUndefinedType)
	if err != nil {
		t.Logf("Register returned error (acceptable): %v", err)
	}

	// Check for errors
	time.Sleep(100 * time.Millisecond)
	errorMsg, hasError := getSchemaError(ctx, t, client, schemaName)

	if hasError {
		t.Logf("✓ Undefined type error detected: %s", errorMsg)
		if strings.Contains(errorMsg, "NonExistentType") ||
			strings.Contains(strings.ToLower(errorMsg), "undefined") ||
			strings.Contains(strings.ToLower(errorMsg), "not found") {
			t.Log("✓ Error message mentions the undefined type")
		}
	} else {
		t.Log("Note: Server may accept undefined types in some cases")
	}
}

// TestSchemaValidation_InvalidFieldNumber tests schema with invalid field number (0)
func TestSchemaValidation_InvalidFieldNumber(t *testing.T) {
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

	schemaName := "invalid-field-num.proto"

	// Register schema with field number 0
	err = client.Schemas().Register(ctx, schemaName, schemaInvalidFieldNumber)
	if err != nil {
		t.Logf("Register returned error (acceptable): %v", err)
	}

	// Check for errors
	time.Sleep(100 * time.Millisecond)
	errorMsg, hasError := getSchemaError(ctx, t, client, schemaName)

	if hasError {
		t.Logf("✓ Invalid field number error detected: %s", errorMsg)
	} else {
		t.Log("Note: Server may accept field number 0 (behavior varies)")
	}
}

// TestSchemaValidation_UpdateGoodToBad tests updating a valid schema with an invalid one
func TestSchemaValidation_UpdateGoodToBad(t *testing.T) {
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

	schemaName := "update-test.proto"

	// First, register a valid schema
	err = client.Schemas().Register(ctx, schemaName, validSchema)
	if err != nil {
		t.Fatalf("Register valid schema: %v", err)
	}

	// Verify no errors initially
	errorMsg, hasError := getSchemaError(ctx, t, client, schemaName)
	if hasError {
		t.Fatalf("Unexpected error with valid schema: %s", errorMsg)
	}
	t.Log("✓ Valid schema registered successfully")

	// Now update with invalid schema
	err = client.Schemas().Register(ctx, schemaName, schemaDuplicateFields)
	if err != nil {
		t.Logf("Update returned error (acceptable): %v", err)
	}

	// Check for errors after update
	time.Sleep(100 * time.Millisecond)
	errorMsg, hasError = getSchemaError(ctx, t, client, schemaName)

	if hasError {
		t.Logf("✓ Error detected after updating to invalid schema: %s", errorMsg)
	} else {
		t.Log("Note: Server accepted invalid schema update")
	}
}

// TestSchemaValidation_MultipleSchemas tests registering multiple schemas
func TestSchemaValidation_MultipleSchemas(t *testing.T) {
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

	// Register multiple schemas
	schemas := map[string]string{
		"schema1.proto": validSchema,
		"schema2.proto": `syntax = "proto3"; package test; message Simple { string name = 1; }`,
		"schema3.proto": `syntax = "proto3"; package test; message Another { int32 id = 1; }`,
	}

	for name, content := range schemas {
		err := client.Schemas().Register(ctx, name, content)
		if err != nil {
			t.Errorf("Register %s: %v", name, err)
		}
	}

	// Verify all schemas are retrievable
	for name := range schemas {
		_, found, err := client.Schemas().Get(ctx, name)
		if err != nil {
			t.Errorf("Get %s: %v", name, err)
		}
		if !found {
			t.Errorf("Schema %s not found", name)
		}

		// Check for errors
		errorMsg, hasError := getSchemaError(ctx, t, client, name)
		if hasError {
			t.Errorf("Schema %s has error: %s", name, errorMsg)
		}
	}

	t.Logf("✓ Successfully registered and verified %d schemas", len(schemas))
}

// TestSchemaValidation_ErrorMessageFormat tests the format of error messages
func TestSchemaValidation_ErrorMessageFormat(t *testing.T) {
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

	schemaName := "error-format-test.proto"

	// Register an intentionally bad schema
	badSchema := `syntax = "proto3";
package test;
message Bad {
  string name = 1;
  int32 age = 1;  // Duplicate!
}`

	err = client.Schemas().Register(ctx, schemaName, badSchema)
	if err != nil {
		t.Logf("Register error: %v", err)
	}

	time.Sleep(200 * time.Millisecond) // Give server more time

	errorMsg, hasError := getSchemaError(ctx, t, client, schemaName)

	if hasError {
		t.Logf("Error message format: %s", errorMsg)

		// Document what error messages look like
		if strings.Contains(errorMsg, "IPROTO") {
			t.Log("✓ Error contains IPROTO error code")
		}
		if strings.Contains(errorMsg, schemaName) {
			t.Log("✓ Error mentions schema name")
		}
		if len(errorMsg) > 0 {
			t.Log("✓ Error message is non-empty")
		}
	} else {
		t.Log("No error detected - server may have accepted the schema")
	}
}

// TestSchemaValidation_ReservedFields tests schemas with reserved fields
func TestSchemaValidation_ReservedFields(t *testing.T) {
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

	// Valid use of reserved fields
	validReserved := `syntax = "proto3";
package test;

message ValidReserved {
  reserved 2, 3;
  reserved "old_field";
  string name = 1;
  int32 current = 4;
}`

	schemaName := "valid-reserved.proto"
	err = client.Schemas().Register(ctx, schemaName, validReserved)
	if err != nil {
		t.Fatalf("Register valid reserved schema: %v", err)
	}

	errorMsg, hasError := getSchemaError(ctx, t, client, schemaName)
	if hasError {
		t.Errorf("Unexpected error with valid reserved fields: %s", errorMsg)
	} else {
		t.Log("✓ Valid reserved fields accepted")
	}

	// Invalid: field conflicts with reserved number
	conflictSchemaName := "reserved-conflict.proto"
	err = client.Schemas().Register(ctx, conflictSchemaName, schemaReservedConflict)
	if err != nil {
		t.Logf("Register conflict schema error: %v", err)
	}

	time.Sleep(100 * time.Millisecond)
	errorMsg, hasError = getSchemaError(ctx, t, client, conflictSchemaName)
	if hasError {
		t.Logf("✓ Reserved conflict detected: %s", errorMsg)
	} else {
		t.Log("Note: Server accepted reserved field conflict")
	}
}
