package operation

import (
	"bytes"
	"testing"

	"infinispan.org/go-client/internal/codec"
)

func TestStatsOpDecodeResponse(t *testing.T) {
	var buf bytes.Buffer
	_ = codec.WriteVInt(&buf, 2)
	_ = codec.WriteLPString(&buf, "timeSinceStart")
	_ = codec.WriteLPString(&buf, "42")
	_ = codec.WriteLPString(&buf, "currentNumberOfEntries")
	_ = codec.WriteLPString(&buf, "10")

	result, err := (&StatsOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	stats := result.(map[string]string)
	if len(stats) != 2 {
		t.Fatalf("expected 2 stats, got %d", len(stats))
	}
	if stats["timeSinceStart"] != "42" {
		t.Errorf("timeSinceStart = %q, want %q", stats["timeSinceStart"], "42")
	}
	if stats["currentNumberOfEntries"] != "10" {
		t.Errorf("currentNumberOfEntries = %q, want %q", stats["currentNumberOfEntries"], "10")
	}
}

func TestStatsOpDecodeEmpty(t *testing.T) {
	var buf bytes.Buffer
	_ = codec.WriteVInt(&buf, 0)

	result, err := (&StatsOp{}).DecodeResponse(codec.StatusSuccess, bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	stats := result.(map[string]string)
	if len(stats) != 0 {
		t.Errorf("expected empty stats, got %d entries", len(stats))
	}
}
