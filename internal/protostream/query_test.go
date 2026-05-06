package protostream

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestEncodeQueryRequest_Simple(t *testing.T) {
	encoded := EncodeQueryRequest("FROM test.Person", 0, -1, nil)
	if len(encoded) == 0 {
		t.Fatal("EncodeQueryRequest returned empty")
	}

	var foundQuery bool
	err := ScanFields(encoded, func(fn int, wt int, val []byte) error {
		if fn == 1 && wt == wireLengthDelimited {
			if string(val) != "FROM test.Person" {
				t.Errorf("query = %q, want %q", val, "FROM test.Person")
			}
			foundQuery = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !foundQuery {
		t.Error("missing query string field")
	}
}

func TestEncodeQueryRequest_WithParams(t *testing.T) {
	params := []QueryParam{
		{Name: "minAge", Value: WrapInt32(18)},
		{Name: "name", Value: WrapString("Alice")},
	}
	encoded := EncodeQueryRequest("FROM test.Person WHERE age > :minAge AND name = :name", 0, 10, params)
	if len(encoded) == 0 {
		t.Fatal("EncodeQueryRequest returned empty")
	}

	var paramCount int
	var foundMaxResults bool
	err := ScanFields(encoded, func(fn int, wt int, val []byte) error {
		switch fn {
		case 4: // maxResults
			v, n := decodeUvarint(val)
			if n <= 0 {
				t.Fatal("invalid maxResults varint")
			}
			if int32(v) != 10 {
				t.Errorf("maxResults = %d, want 10", v)
			}
			foundMaxResults = true
		case 5: // namedParameters
			paramCount++
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !foundMaxResults {
		t.Error("missing maxResults field")
	}
	if paramCount != 2 {
		t.Errorf("param count = %d, want 2", paramCount)
	}
}

func TestEncodeQueryRequest_WithOffset(t *testing.T) {
	encoded := EncodeQueryRequest("FROM test.Person", 50, 10, nil)

	var foundOffset bool
	err := ScanFields(encoded, func(fn int, wt int, val []byte) error {
		if fn == 3 { // startOffset
			v, n := decodeUvarint(val)
			if n <= 0 {
				t.Fatal("invalid startOffset varint")
			}
			if int64(v) != 50 {
				t.Errorf("startOffset = %d, want 50", v)
			}
			foundOffset = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !foundOffset {
		t.Error("missing startOffset field")
	}
}

func TestDecodeQueryResponse(t *testing.T) {
	// Build a mock QueryResponse protobuf:
	// field 1 (numResults) = 2, field 2 (projectionSize) = 0,
	// field 3 (results) = two entries, field 4 (hitCount) = 5, field 5 (hitCountExact) = true
	inner := appendVarintField(nil, 1, 2)
	inner = appendVarintField(inner, 2, 0)
	inner = appendLenDelimited(inner, 3, []byte("result1"))
	inner = appendLenDelimited(inner, 3, []byte("result2"))
	inner = appendVarintField(inner, 4, 5)
	inner = appendVarintField(inner, 5, 1)

	resp, err := DecodeQueryResponse(inner)
	if err != nil {
		t.Fatalf("DecodeQueryResponse: %v", err)
	}

	if resp.NumResults != 2 {
		t.Errorf("NumResults = %d, want 2", resp.NumResults)
	}
	if resp.ProjectionSize != 0 {
		t.Errorf("ProjectionSize = %d, want 0", resp.ProjectionSize)
	}
	if len(resp.Results) != 2 {
		t.Fatalf("len(Results) = %d, want 2", len(resp.Results))
	}
	if string(resp.Results[0]) != "result1" {
		t.Errorf("Results[0] = %q, want %q", resp.Results[0], "result1")
	}
	if string(resp.Results[1]) != "result2" {
		t.Errorf("Results[1] = %q, want %q", resp.Results[1], "result2")
	}
	if resp.HitCount != 5 {
		t.Errorf("HitCount = %d, want 5", resp.HitCount)
	}
	if !resp.HitCountExact {
		t.Error("HitCountExact = false, want true")
	}
}

func TestWrapByTypeID(t *testing.T) {
	msg := []byte("test-payload")
	wrapped := WrapByTypeID(msg, 4400)

	var foundTypeID, foundMsg bool
	err := ScanFields(wrapped, func(fn int, wt int, val []byte) error {
		switch fn {
		case fieldWrappedDescriptorTypeID:
			v, n := decodeUvarint(val)
			if n <= 0 {
				t.Fatal("invalid typeID varint")
			}
			if int32(v) != 4400 {
				t.Errorf("typeID = %d, want 4400", v)
			}
			foundTypeID = true
		case fieldWrappedMessage:
			if string(val) != "test-payload" {
				t.Errorf("msg = %q, want %q", val, "test-payload")
			}
			foundMsg = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !foundTypeID {
		t.Error("missing typeID field (19)")
	}
	if !foundMsg {
		t.Error("missing message field (17)")
	}
}

func TestWrapFloat(t *testing.T) {
	wrapped := WrapFloat(1.5)
	if len(wrapped) == 0 {
		t.Fatal("WrapFloat returned empty")
	}

	var foundFloat bool
	err := ScanFields(wrapped, func(fn int, wt int, val []byte) error {
		if fn == fieldWrappedFloat && wt == wireFixed32 {
			if len(val) != 4 {
				t.Fatalf("expected 4 bytes, got %d", len(val))
			}
			bits := binary.LittleEndian.Uint32(val)
			got := math.Float32frombits(bits)
			if got != 1.5 {
				t.Errorf("float value = %f, want 1.5", got)
			}
			foundFloat = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !foundFloat {
		t.Error("missing wrappedFloat field (2)")
	}
}

func TestWrapFloatArray(t *testing.T) {
	values := []float32{0.9, 0.1, 0.1}
	wrapped := WrapFloatArray(values)
	if len(wrapped) == 0 {
		t.Fatal("WrapFloatArray returned empty")
	}

	var containerSize int
	var typeName string
	var containerMsg []byte
	var floats []float32
	var foundSize, foundType, foundMsg bool

	err := ScanFields(wrapped, func(fn int, wt int, val []byte) error {
		switch fn {
		case fieldWrappedContainerTypeName:
			typeName = string(val)
			foundType = true
		case fieldWrappedContainerSize:
			v, n := decodeUvarint(val)
			if n <= 0 {
				t.Fatal("invalid container size varint")
			}
			containerSize = int(v)
			foundSize = true
		case fieldWrappedContainerMessage:
			containerMsg = val
			foundMsg = true
		case fieldWrappedFloat:
			if wt != wireFixed32 || len(val) != 4 {
				t.Fatalf("expected fixed32 with 4 bytes, got wt=%d len=%d", wt, len(val))
			}
			bits := binary.LittleEndian.Uint32(val)
			floats = append(floats, math.Float32frombits(bits))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !foundType {
		t.Error("missing containerTypeName field (28)")
	}
	if typeName != "org.infinispan.protostream.commons.FloatArray" {
		t.Errorf("typeName = %q, want %q", typeName, "org.infinispan.protostream.commons.FloatArray")
	}
	if !foundSize {
		t.Error("missing containerSize field (27)")
	}
	if containerSize != 3 {
		t.Errorf("containerSize = %d, want 3", containerSize)
	}
	if !foundMsg {
		t.Error("missing containerMessage field (30)")
	}
	if len(containerMsg) != 0 {
		t.Errorf("containerMessage should be empty, got %d bytes", len(containerMsg))
	}
	if len(floats) != 3 {
		t.Fatalf("float count = %d, want 3", len(floats))
	}
	for i, want := range values {
		if floats[i] != want {
			t.Errorf("float[%d] = %f, want %f", i, floats[i], want)
		}
	}
}
