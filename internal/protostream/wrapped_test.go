package protostream

import (
	"bytes"
	"testing"
)

func TestWrapUnwrapString(t *testing.T) {
	tests := []string{"hello", "", "utf8: äöü", "a longer string value for testing"}
	for _, s := range tests {
		wrapped := WrapString(s)
		got, err := UnwrapString(wrapped)
		if err != nil {
			t.Fatalf("UnwrapString(%q): %v", s, err)
		}
		if got != s {
			t.Errorf("UnwrapString round-trip: got %q, want %q", got, s)
		}
	}
}

func TestWrapUnwrapMessage(t *testing.T) {
	inner := []byte{0x0A, 0x04, 0x4A, 0x6F, 0x68, 0x6E, 0x10, 0x1E}
	typeName := "test.Person"

	wrapped := WrapMessage(inner, typeName)
	gotBytes, gotName, err := UnwrapMessage(wrapped)
	if err != nil {
		t.Fatalf("UnwrapMessage: %v", err)
	}
	if gotName != typeName {
		t.Errorf("type name = %q, want %q", gotName, typeName)
	}
	if !bytes.Equal(gotBytes, inner) {
		t.Errorf("message bytes = %x, want %x", gotBytes, inner)
	}
}

func TestWrapMessageKnownBytes(t *testing.T) {
	inner := []byte{0x0A, 0x04, 0x4A, 0x6F, 0x68, 0x6E}
	typeName := "test.Person"

	wrapped := WrapMessage(inner, typeName)

	// Verify field 16 (wrappedDescriptorFullName) is present
	// tag = (16 << 3) | 2 = 130 = 0x82 0x01
	if wrapped[0] != 0x82 || wrapped[1] != 0x01 {
		t.Errorf("field 16 tag = %x %x, want 0x82 0x01", wrapped[0], wrapped[1])
	}
	// length of "test.Person" = 11
	if wrapped[2] != 11 {
		t.Errorf("field 16 length = %d, want 11", wrapped[2])
	}
	// "test.Person" bytes
	if string(wrapped[3:14]) != "test.Person" {
		t.Errorf("field 16 value = %q, want %q", string(wrapped[3:14]), "test.Person")
	}

	// Verify field 17 (wrappedMessage) follows
	// tag = (17 << 3) | 2 = 138 = 0x8A 0x01
	if wrapped[14] != 0x8A || wrapped[15] != 0x01 {
		t.Errorf("field 17 tag = %x %x, want 0x8A 0x01", wrapped[14], wrapped[15])
	}
	// length of inner = 6
	if wrapped[16] != 6 {
		t.Errorf("field 17 length = %d, want 6", wrapped[16])
	}
	if !bytes.Equal(wrapped[17:23], inner) {
		t.Errorf("field 17 value = %x, want %x", wrapped[17:23], inner)
	}
}

func TestWrapStringKnownBytes(t *testing.T) {
	wrapped := WrapString("hello")

	// tag for field 9 with wire type 2: (9 << 3) | 2 = 74 = 0x4A
	if wrapped[0] != 0x4A {
		t.Errorf("field 9 tag = 0x%02x, want 0x4A", wrapped[0])
	}
	// length = 5
	if wrapped[1] != 5 {
		t.Errorf("field 9 length = %d, want 5", wrapped[1])
	}
	if string(wrapped[2:7]) != "hello" {
		t.Errorf("field 9 value = %q, want %q", string(wrapped[2:7]), "hello")
	}
}

func TestWrapInt32(t *testing.T) {
	wrapped := WrapInt32(42)

	// tag for field 5 with wire type 0: (5 << 3) | 0 = 40 = 0x28
	if wrapped[0] != 0x28 {
		t.Errorf("field 5 tag = 0x%02x, want 0x28", wrapped[0])
	}
	// varint 42 = 0x2A
	if wrapped[1] != 0x2A {
		t.Errorf("varint value = 0x%02x, want 0x2A", wrapped[1])
	}
}

func TestWrapInt64(t *testing.T) {
	wrapped := WrapInt64(1000000)

	// tag for field 3 with wire type 0: (3 << 3) | 0 = 24 = 0x18
	if wrapped[0] != 0x18 {
		t.Errorf("field 3 tag = 0x%02x, want 0x18", wrapped[0])
	}
	// 1000000 = 0xC0 0x84 0x3D in varint
	if wrapped[1] != 0xC0 || wrapped[2] != 0x84 || wrapped[3] != 0x3D {
		t.Errorf("varint value = %x, want c0 84 3d", wrapped[1:4])
	}
}

func TestUnwrapMessageMissingFields(t *testing.T) {
	// Only field 17, no field 16
	data := appendLenDelimited(nil, fieldWrappedMessage, []byte{0x01})
	_, _, err := UnwrapMessage(data)
	if err == nil {
		t.Error("expected error for missing type name")
	}

	// Only field 16, no field 17
	data = appendLenDelimited(nil, fieldWrappedDescriptorFullName, []byte("test.Type"))
	_, _, err = UnwrapMessage(data)
	if err == nil {
		t.Error("expected error for missing message bytes")
	}
}

func TestUnwrapStringMissingField(t *testing.T) {
	// Empty data
	_, err := UnwrapString([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}

	// Wrong field (field 3 instead of 9)
	data := appendVarintField(nil, 3, 42)
	_, err = UnwrapString(data)
	if err == nil {
		t.Error("expected error for wrong field")
	}
}

func TestScanFieldsIgnoresUnknown(t *testing.T) {
	// Build a WrappedMessage with an extra unknown field between 16 and 17
	var data []byte
	data = appendLenDelimited(data, fieldWrappedDescriptorFullName, []byte("test.Foo"))
	data = appendVarintField(data, 99, 12345) // unknown field
	data = appendLenDelimited(data, fieldWrappedMessage, []byte{0xAA, 0xBB})

	msgBytes, typeName, err := UnwrapMessage(data)
	if err != nil {
		t.Fatalf("UnwrapMessage with unknown field: %v", err)
	}
	if typeName != "test.Foo" {
		t.Errorf("typeName = %q, want %q", typeName, "test.Foo")
	}
	if !bytes.Equal(msgBytes, []byte{0xAA, 0xBB}) {
		t.Errorf("msgBytes = %x, want aabb", msgBytes)
	}
}

func TestWrapMessageEmptyPayload(t *testing.T) {
	wrapped := WrapMessage([]byte{}, "test.Empty")
	gotBytes, gotName, err := UnwrapMessage(wrapped)
	if err != nil {
		t.Fatalf("UnwrapMessage(empty): %v", err)
	}
	if gotName != "test.Empty" {
		t.Errorf("type name = %q, want %q", gotName, "test.Empty")
	}
	if len(gotBytes) != 0 {
		t.Errorf("expected empty bytes, got %x", gotBytes)
	}
}

func TestUnwrapBytes(t *testing.T) {
	inner := []byte{0x0A, 0x04, 0x74, 0x65, 0x73, 0x74}
	wrapped := WrapMessage(inner, "test.Foo")
	got, err := UnwrapBytes(wrapped)
	if err != nil {
		t.Fatalf("UnwrapBytes: %v", err)
	}
	if !bytes.Equal(got, inner) {
		t.Errorf("UnwrapBytes = %x, want %x", got, inner)
	}
}

func TestUnwrapBytesWithTypeId(t *testing.T) {
	inner := []byte{0xDE, 0xAD}
	var data []byte
	data = appendVarintField(data, fieldWrappedDescriptorTypeID, 42)
	data = appendLenDelimited(data, fieldWrappedMessage, inner)
	got, err := UnwrapBytes(data)
	if err != nil {
		t.Fatalf("UnwrapBytes with typeId: %v", err)
	}
	if !bytes.Equal(got, inner) {
		t.Errorf("UnwrapBytes = %x, want %x", got, inner)
	}
}

func TestUnwrapBytesMissingField17(t *testing.T) {
	data := appendLenDelimited(nil, fieldWrappedDescriptorFullName, []byte("test.Type"))
	_, err := UnwrapBytes(data)
	if err == nil {
		t.Error("expected error for missing wrappedMessage field")
	}
}

func TestDecodeCQResultJoining(t *testing.T) {
	key := []byte{0x01, 0x02}
	value := []byte{0x03, 0x04}
	var data []byte
	data = appendVarintField(data, 1, uint64(CQJoining))
	data = appendLenDelimited(data, 2, key)
	data = appendLenDelimited(data, 3, value)

	r, err := DecodeCQResult(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.ResultType != CQJoining {
		t.Errorf("ResultType = %d, want %d", r.ResultType, CQJoining)
	}
	if !bytes.Equal(r.Key, key) {
		t.Errorf("Key = %x, want %x", r.Key, key)
	}
	if !bytes.Equal(r.Value, value) {
		t.Errorf("Value = %x, want %x", r.Value, value)
	}
	if len(r.Projections) != 0 {
		t.Errorf("Projections = %d entries, want 0", len(r.Projections))
	}
}

func TestDecodeCQResultLeaving(t *testing.T) {
	key := []byte{0x05, 0x06}
	var data []byte
	data = appendVarintField(data, 1, uint64(CQLeaving))
	data = appendLenDelimited(data, 2, key)

	r, err := DecodeCQResult(data)
	if err != nil {
		t.Fatal(err)
	}
	if r.ResultType != CQLeaving {
		t.Errorf("ResultType = %d, want %d", r.ResultType, CQLeaving)
	}
	if !bytes.Equal(r.Key, key) {
		t.Errorf("Key = %x, want %x", r.Key, key)
	}
	if r.Value != nil {
		t.Errorf("Value = %x, want nil", r.Value)
	}
}

func TestDecodeCQResultWithProjections(t *testing.T) {
	proj1 := []byte{0x0A}
	proj2 := []byte{0x0B, 0x0C}
	var data []byte
	data = appendVarintField(data, 1, uint64(CQUpdated))
	data = appendLenDelimited(data, 2, []byte{0x01})
	data = appendLenDelimited(data, 4, proj1)
	data = appendLenDelimited(data, 4, proj2)

	r, err := DecodeCQResult(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Projections) != 2 {
		t.Fatalf("Projections = %d entries, want 2", len(r.Projections))
	}
	if !bytes.Equal(r.Projections[0], proj1) {
		t.Errorf("Projections[0] = %x, want %x", r.Projections[0], proj1)
	}
	if !bytes.Equal(r.Projections[1], proj2) {
		t.Errorf("Projections[1] = %x, want %x", r.Projections[1], proj2)
	}
}

func TestDecodeCQResultMissingKey(t *testing.T) {
	var data []byte
	data = appendVarintField(data, 1, uint64(CQJoining))
	_, err := DecodeCQResult(data)
	if err == nil {
		t.Error("expected error for missing key field")
	}
}
