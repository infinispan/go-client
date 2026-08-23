package hotrod_test

import (
	"fmt"
	"testing"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/codec"
	"infinispan.org/go-client/internal/testproto"
)

func TestProtoStreamMarshallerMediaType(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}
	if m.MediaType() != codec.MediaIDProtostream {
		t.Errorf("MediaType() = %d, want %d", m.MediaType(), codec.MediaIDProtostream)
	}
}

func TestProtoStreamMarshalStringKey(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}
	data, err := m.MarshalKey("hello")
	if err != nil {
		t.Fatalf("MarshalKey(string): %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
	var result string
	if err := m.UnmarshalKey(data, &result); err != nil {
		t.Fatalf("UnmarshalKey(string): %v", err)
	}
	if result != "hello" {
		t.Errorf("round-trip: got %q, want %q", result, "hello")
	}
}

func TestProtoStreamMarshalInt32Key(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}
	data, err := m.MarshalKey(int32(42))
	if err != nil {
		t.Fatalf("MarshalKey(int32): %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestProtoStreamMarshalInt64Key(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}
	data, err := m.MarshalKey(int64(1000000))
	if err != nil {
		t.Fatalf("MarshalKey(int64): %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestProtoStreamMarshalUnsupportedKey(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}
	_, err := m.MarshalKey(3.14)
	if err == nil {
		t.Fatal("expected error for unsupported key type")
	}
}

func TestProtoStreamMarshalValue(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}
	person := &testproto.Person{Name: "John", Age: 30}

	data, err := m.MarshalValue(person)
	if err != nil {
		t.Fatalf("MarshalValue: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}

	result := &testproto.Person{}
	if err := m.UnmarshalValue(data, result); err != nil {
		t.Fatalf("UnmarshalValue: %v", err)
	}
	if result.Name != "John" {
		t.Errorf("Name = %q, want %q", result.Name, "John")
	}
	if result.Age != 30 {
		t.Errorf("Age = %d, want %d", result.Age, 30)
	}
}

func TestProtoStreamMarshalValueNotProto(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}
	_, err := m.MarshalValue("not a proto message")
	if err == nil {
		t.Fatal("expected error for non-proto value")
	}
}

func TestProtoStreamMarshalProtoKey(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}
	person := &testproto.Person{Name: "key-person", Age: 1}

	data, err := m.MarshalKey(person)
	if err != nil {
		t.Fatalf("MarshalKey(proto): %v", err)
	}

	result := &testproto.Person{}
	if err := m.UnmarshalKey(data, result); err != nil {
		t.Fatalf("UnmarshalKey(proto): %v", err)
	}
	if result.Name != "key-person" {
		t.Errorf("Name = %q, want %q", result.Name, "key-person")
	}
}

// TestProtoStreamMarshal_AllPrimitiveKeys tests all supported primitive key types
func TestProtoStreamMarshal_AllPrimitiveKeys(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}

	tests := []struct {
		name  string
		key   interface{}
		check func(data []byte) error
	}{
		{
			name: "string",
			key:  "test-string",
			check: func(data []byte) error {
				var result string
				if err := m.UnmarshalKey(data, &result); err != nil {
					return err
				}
				if result != "test-string" {
					return fmt.Errorf("got %q, want %q", result, "test-string")
				}
				return nil
			},
		},
		{
			name: "int32",
			key:  int32(42),
			check: func(data []byte) error {
				// Note: ProtoStream doesn't support unmarshalling int32 keys back
				// This just verifies marshalling succeeds
				if len(data) == 0 {
					return fmt.Errorf("empty data")
				}
				return nil
			},
		},
		{
			name: "int64",
			key:  int64(1234567890),
			check: func(data []byte) error {
				if len(data) == 0 {
					return fmt.Errorf("empty data")
				}
				return nil
			},
		},
		{
			name: "proto.Message",
			key:  &testproto.Person{Name: "proto-key", Age: 99},
			check: func(data []byte) error {
				result := &testproto.Person{}
				if err := m.UnmarshalKey(data, result); err != nil {
					return err
				}
				if result.Name != "proto-key" {
					return fmt.Errorf("Name = %q, want %q", result.Name, "proto-key")
				}
				if result.Age != 99 {
					return fmt.Errorf("Age = %d, want 99", result.Age)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := m.MarshalKey(tt.key)
			if err != nil {
				t.Fatalf("MarshalKey(%v): %v", tt.name, err)
			}
			if len(data) == 0 {
				t.Fatal("expected non-empty data")
			}

			if err := tt.check(data); err != nil {
				t.Errorf("check failed: %v", err)
			}
		})
	}
}

// TestProtoStreamMarshal_AllPrimitiveValues tests all primitive value types
func TestProtoStreamMarshal_AllPrimitiveValues(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}

	// Note: ProtoStream values must be proto.Message types
	// Primitive values need to be wrapped in messages

	tests := []struct {
		name  string
		value *testproto.Person
	}{
		{
			name:  "simple proto message",
			value: &testproto.Person{Name: "Alice", Age: 30},
		},
		{
			name:  "empty name",
			value: &testproto.Person{Name: "", Age: 0},
		},
		{
			name:  "max age",
			value: &testproto.Person{Name: "Elder", Age: 2147483647}, // max int32
		},
		{
			name:  "unicode name",
			value: &testproto.Person{Name: "张三", Age: 25},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := m.MarshalValue(tt.value)
			if err != nil {
				t.Fatalf("MarshalValue: %v", err)
			}
			if len(data) == 0 {
				t.Fatal("expected non-empty data")
			}

			result := &testproto.Person{}
			if err := m.UnmarshalValue(data, result); err != nil {
				t.Fatalf("UnmarshalValue: %v", err)
			}

			if result.Name != tt.value.Name {
				t.Errorf("Name = %q, want %q", result.Name, tt.value.Name)
			}
			if result.Age != tt.value.Age {
				t.Errorf("Age = %d, want %d", result.Age, tt.value.Age)
			}
		})
	}
}

// TestProtoStreamMarshal_EdgeCases tests edge cases and special values
func TestProtoStreamMarshal_EdgeCases(t *testing.T) {
	m := &hotrod.ProtoStreamMarshaller{}

	t.Run("empty string key", func(t *testing.T) {
		data, err := m.MarshalKey("")
		if err != nil {
			t.Fatalf("MarshalKey(empty string): %v", err)
		}
		var result string
		if err := m.UnmarshalKey(data, &result); err != nil {
			t.Fatalf("UnmarshalKey: %v", err)
		}
		if result != "" {
			t.Errorf("got %q, want empty string", result)
		}
	})

	t.Run("zero int32 key", func(t *testing.T) {
		data, err := m.MarshalKey(int32(0))
		if err != nil {
			t.Fatalf("MarshalKey(0): %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty data for zero value")
		}
	})

	t.Run("negative int64 key", func(t *testing.T) {
		data, err := m.MarshalKey(int64(-999))
		if err != nil {
			t.Fatalf("MarshalKey(-999): %v", err)
		}
		if len(data) == 0 {
			t.Error("expected non-empty data")
		}
	})

	t.Run("long string key", func(t *testing.T) {
		longStr := string(make([]byte, 1000))
		for i := range longStr {
			longStr = longStr[:i] + "a" + longStr[i+1:]
		}
		data, err := m.MarshalKey(longStr)
		if err != nil {
			t.Fatalf("MarshalKey(long string): %v", err)
		}
		var result string
		if err := m.UnmarshalKey(data, &result); err != nil {
			t.Fatalf("UnmarshalKey: %v", err)
		}
		if len(result) != 1000 {
			t.Errorf("len(result) = %d, want 1000", len(result))
		}
	})

	t.Run("zero proto message", func(t *testing.T) {
		person := &testproto.Person{} // All zero values
		data, err := m.MarshalValue(person)
		if err != nil {
			t.Fatalf("MarshalValue(zero proto): %v", err)
		}

		result := &testproto.Person{}
		if err := m.UnmarshalValue(data, result); err != nil {
			t.Fatalf("UnmarshalValue: %v", err)
		}
		if result.Name != "" {
			t.Errorf("Name = %q, want empty", result.Name)
		}
		if result.Age != 0 {
			t.Errorf("Age = %d, want 0", result.Age)
		}
	})
}
