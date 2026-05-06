package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestClearOpDecodeSuccess(t *testing.T) {
	result, err := (&ClearOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}
