package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestContainsKeyOpDecodeExists(t *testing.T) {
	result, err := (&ContainsKeyOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if result != true {
		t.Errorf("expected true, got %v", result)
	}
}

func TestContainsKeyOpDecodeNotExists(t *testing.T) {
	result, err := (&ContainsKeyOp{}).DecodeResponse(codec.StatusKeyDoesNotExist, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	if result != false {
		t.Errorf("expected false, got %v", result)
	}
}
