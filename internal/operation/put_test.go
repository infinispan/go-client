package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestPutOpFlags(t *testing.T) {
	op := &PutOp{OpFlags: 0x0009}
	if op.Flags() != 0x0009 {
		t.Errorf("Flags() = 0x%04x, want 0x0009", op.Flags())
	}
}

func TestPutOpFlagsDefault(t *testing.T) {
	op := &PutOp{}
	if op.Flags() != 0 {
		t.Errorf("Flags() = 0x%04x, want 0", op.Flags())
	}
}

func writeMetadataValue(buf *bytes.Buffer, value []byte) {
	buf.WriteByte(0x03) // flags: both lifespan and maxidle infinite
	buf.Write([]byte{0, 0, 0, 0, 0, 0, 0, 1}) // version = 1
	_ = codec.WriteLPBytes(buf, value)
}

func TestPutOpDecodeSuccessWithPrevious(t *testing.T) {
	var buf bytes.Buffer
	writeMetadataValue(&buf, []byte("old-value"))

	result, err := (&PutOp{}).DecodeResponse(codec.StatusSuccessWithPrevious, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	resp, ok := result.(*PutResponse)
	if !ok {
		t.Fatalf("expected *PutResponse, got %T", result)
	}
	if string(resp.PreviousValue) != "old-value" {
		t.Errorf("PreviousValue = %q, want %q", resp.PreviousValue, "old-value")
	}
}

func TestPutOpDecodeNotExecWithPrevious(t *testing.T) {
	var buf bytes.Buffer
	writeMetadataValue(&buf, []byte("existing"))

	result, err := (&PutOp{}).DecodeResponse(codec.StatusNotExecWithPrevious, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*PutResponse)
	if string(resp.PreviousValue) != "existing" {
		t.Errorf("PreviousValue = %q, want %q", resp.PreviousValue, "existing")
	}
}

func TestPutOpDecodeSuccess(t *testing.T) {
	result, err := (&PutOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}
