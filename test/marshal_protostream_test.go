package hotrod_test

import (
	"testing"

	"infinispan.org/go-client/hotrod"
	"infinispan.org/go-client/internal/codec"
	"infinispan.org/go-client/internal/testproto"
)

func TestProtoStreamMarshallerMediaType(t *testing.T) {
	m := hotrod.ProtoStreamMarshaller()
	if m.MediaType() != codec.MediaIDProtostream {
		t.Errorf("MediaType() = %d, want %d", m.MediaType(), codec.MediaIDProtostream)
	}
}

func TestProtoStreamMarshalStringKey(t *testing.T) {
	m := hotrod.ProtoStreamMarshaller()
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
	m := hotrod.ProtoStreamMarshaller()
	data, err := m.MarshalKey(int32(42))
	if err != nil {
		t.Fatalf("MarshalKey(int32): %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestProtoStreamMarshalInt64Key(t *testing.T) {
	m := hotrod.ProtoStreamMarshaller()
	data, err := m.MarshalKey(int64(1000000))
	if err != nil {
		t.Fatalf("MarshalKey(int64): %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty data")
	}
}

func TestProtoStreamMarshalUnsupportedKey(t *testing.T) {
	m := hotrod.ProtoStreamMarshaller()
	_, err := m.MarshalKey(3.14)
	if err == nil {
		t.Fatal("expected error for unsupported key type")
	}
}

func TestProtoStreamMarshalValue(t *testing.T) {
	m := hotrod.ProtoStreamMarshaller()
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
	m := hotrod.ProtoStreamMarshaller()
	_, err := m.MarshalValue("not a proto message")
	if err == nil {
		t.Fatal("expected error for non-proto value")
	}
}

func TestProtoStreamMarshalProtoKey(t *testing.T) {
	m := hotrod.ProtoStreamMarshaller()
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
