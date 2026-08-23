package codec

import (
	"bytes"
	"testing"
	"time"
)

func TestEncodeTimeUnitsDefaultDefault(t *testing.T) {
	tu := EncodeTimeUnits(0, 0)
	want := (TimeUnitDefault << 4) | TimeUnitDefault // 0x77
	if tu.Packed != want {
		t.Errorf("packed = 0x%02x, want 0x%02x", tu.Packed, want)
	}
}

func TestEncodeTimeUnitsSecondsDefault(t *testing.T) {
	tu := EncodeTimeUnits(10*time.Second, 0)
	wantPacked := (TimeUnitSeconds << 4) | TimeUnitDefault // 0x07
	if tu.Packed != wantPacked {
		t.Errorf("packed = 0x%02x, want 0x%02x", tu.Packed, wantPacked)
	}
	if tu.Lifespan != 10 {
		t.Errorf("lifespan = %d, want 10", tu.Lifespan)
	}
}

func TestEncodeTimeUnitsInfiniteInfinite(t *testing.T) {
	tu := EncodeTimeUnits(-1, -1)
	want := (TimeUnitInfinite << 4) | TimeUnitInfinite // 0x88
	if tu.Packed != want {
		t.Errorf("packed = 0x%02x, want 0x%02x", tu.Packed, want)
	}
}

func TestEncodeTimeUnitsMilliseconds(t *testing.T) {
	tu := EncodeTimeUnits(500*time.Millisecond, 200*time.Millisecond)
	lifespanNibble := (tu.Packed >> 4) & 0x0F
	maxIdleNibble := tu.Packed & 0x0F
	if lifespanNibble != TimeUnitMilliseconds {
		t.Errorf("lifespan unit = %d, want %d", lifespanNibble, TimeUnitMilliseconds)
	}
	if maxIdleNibble != TimeUnitMilliseconds {
		t.Errorf("maxidle unit = %d, want %d", maxIdleNibble, TimeUnitMilliseconds)
	}
	if tu.Lifespan != 500 {
		t.Errorf("lifespan = %d, want 500", tu.Lifespan)
	}
	if tu.MaxIdle != 200 {
		t.Errorf("maxidle = %d, want 200", tu.MaxIdle)
	}
}

func TestTimeUnitsWrite(t *testing.T) {
	tu := EncodeTimeUnits(10*time.Second, 0)
	var buf bytes.Buffer
	if err := tu.Write(&buf); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	if b[0] != 0x07 {
		t.Errorf("packed byte = 0x%02x, want 0x07", b[0])
	}
	// Lifespan value follows (vLong(10) = 0x0A), no maxIdle (default)
	if b[1] != 0x0A {
		t.Errorf("lifespan value byte = 0x%02x, want 0x0A", b[1])
	}
	if len(b) != 2 {
		t.Errorf("expected 2 bytes, got %d", len(b))
	}
}

func TestTimeUnitsWriteBothPresent(t *testing.T) {
	tu := EncodeTimeUnits(5*time.Second, 3*time.Second)
	var buf bytes.Buffer
	if err := tu.Write(&buf); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	wantPacked := (TimeUnitSeconds << 4) | TimeUnitSeconds // 0x00
	if b[0] != wantPacked {
		t.Errorf("packed = 0x%02x, want 0x%02x", b[0], wantPacked)
	}
	if b[1] != 5 {
		t.Errorf("lifespan = %d, want 5", b[1])
	}
	if b[2] != 3 {
		t.Errorf("maxidle = %d, want 3", b[2])
	}
}
