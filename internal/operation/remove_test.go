package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestRemoveOpFlags(t *testing.T) {
	op := &RemoveOp{OpFlags: 0x0001}
	if op.Flags() != 0x0001 {
		t.Errorf("Flags() = 0x%04x, want 0x0001", op.Flags())
	}
}

func TestRemoveOpDecodeSuccessWithPrevious(t *testing.T) {
	var buf bytes.Buffer
	writeMetadataValue(&buf, []byte("removed-value"))

	result, err := (&RemoveOp{}).DecodeResponse(codec.StatusSuccessWithPrevious, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	resp, ok := result.(*RemoveResponse)
	if !ok {
		t.Fatalf("expected *RemoveResponse, got %T", result)
	}
	if !resp.Existed {
		t.Error("Existed = false, want true")
	}
	if string(resp.PreviousValue) != "removed-value" {
		t.Errorf("PreviousValue = %q, want %q", resp.PreviousValue, "removed-value")
	}
}

func TestRemoveOpDecodeSuccess(t *testing.T) {
	result, err := (&RemoveOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*RemoveResponse)
	if !resp.Existed {
		t.Error("Existed = false, want true")
	}
	if resp.PreviousValue != nil {
		t.Errorf("PreviousValue = %v, want nil", resp.PreviousValue)
	}
}

func TestRemoveOpDecodeKeyDoesNotExist(t *testing.T) {
	result, err := (&RemoveOp{}).DecodeResponse(codec.StatusKeyDoesNotExist, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	resp := result.(*RemoveResponse)
	if resp.Existed {
		t.Error("Existed = true, want false")
	}
}
