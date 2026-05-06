package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestPutIfAbsentOpDecodeSuccess(t *testing.T) {
	result, err := (&PutIfAbsentOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*CASResponse)
	if !resp.Success {
		t.Error("expected Success=true")
	}
	if resp.PreviousValue != nil {
		t.Errorf("expected nil PreviousValue, got %q", resp.PreviousValue)
	}
}

func TestPutIfAbsentOpDecodeNotExecuted(t *testing.T) {
	result, err := (&PutIfAbsentOp{}).DecodeResponse(codec.StatusNotExecuted, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*CASResponse)
	if resp.Success {
		t.Error("expected Success=false")
	}
}

func TestPutIfAbsentOpDecodeNotExecWithPrevious(t *testing.T) {
	var buf bytes.Buffer
	writeMetadataValue(&buf, []byte("existing"))

	result, err := (&PutIfAbsentOp{}).DecodeResponse(codec.StatusNotExecWithPrevious, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*CASResponse)
	if resp.Success {
		t.Error("expected Success=false")
	}
	if string(resp.PreviousValue) != "existing" {
		t.Errorf("PreviousValue = %q, want %q", resp.PreviousValue, "existing")
	}
}
