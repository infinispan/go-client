package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestSizeOpDecodeResponse(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteVInt(&buf, 42)

	result, err := (&SizeOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	size := result.(int32)
	if size != 42 {
		t.Errorf("size = %d, want 42", size)
	}
}

func TestSizeOpDecodeZero(t *testing.T) {
	var buf bytes.Buffer
	codec.WriteVInt(&buf, 0)

	result, err := (&SizeOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	size := result.(int32)
	if size != 0 {
		t.Errorf("size = %d, want 0", size)
	}
}
