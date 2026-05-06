package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestReplaceOpDecodeSuccess(t *testing.T) {
	result, err := (&ReplaceOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*CASResponse)
	if !resp.Success {
		t.Error("expected Success=true")
	}
}

func TestReplaceOpDecodeNotExecuted(t *testing.T) {
	result, err := (&ReplaceOp{}).DecodeResponse(codec.StatusNotExecuted, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*CASResponse)
	if resp.Success {
		t.Error("expected Success=false")
	}
}

func TestReplaceOpDecodeSuccessWithPrevious(t *testing.T) {
	var buf bytes.Buffer
	writeMetadataValue(&buf, []byte("old-value"))

	result, err := (&ReplaceOp{}).DecodeResponse(codec.StatusSuccessWithPrevious, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*CASResponse)
	if !resp.Success {
		t.Error("expected Success=true")
	}
	if string(resp.PreviousValue) != "old-value" {
		t.Errorf("PreviousValue = %q, want %q", resp.PreviousValue, "old-value")
	}
}
