package codec

import (
	"bytes"
	"testing"
)

func TestWriteReadVInt(t *testing.T) {
	tests := []struct {
		name  string
		value int32
		bytes []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"127", 127, []byte{0x7F}},
		{"128", 128, []byte{0x80, 0x01}},
		{"300", 300, []byte{0xAC, 0x02}},
		{"16384", 16384, []byte{0x80, 0x80, 0x01}},
		{"max_vint", 0x7FFFFFFF, []byte{0xFF, 0xFF, 0xFF, 0xFF, 0x07}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteVInt(&buf, tt.value); err != nil {
				t.Fatalf("WriteVInt(%d): %v", tt.value, err)
			}
			if !bytes.Equal(buf.Bytes(), tt.bytes) {
				t.Errorf("WriteVInt(%d) = %v, want %v", tt.value, buf.Bytes(), tt.bytes)
			}

			got, err := ReadVInt(&buf)
			if err != nil {
				t.Fatalf("ReadVInt: %v", err)
			}
			if got != tt.value {
				t.Errorf("ReadVInt = %d, want %d", got, tt.value)
			}
		})
	}
}

func TestWriteReadVLong(t *testing.T) {
	tests := []struct {
		name  string
		value int64
		bytes []byte
	}{
		{"zero", 0, []byte{0x00}},
		{"one", 1, []byte{0x01}},
		{"127", 127, []byte{0x7F}},
		{"128", 128, []byte{0x80, 0x01}},
		{"large", 123456789, []byte{0x95, 0x9A, 0xEF, 0x3A}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteVLong(&buf, tt.value); err != nil {
				t.Fatalf("WriteVLong(%d): %v", tt.value, err)
			}
			if !bytes.Equal(buf.Bytes(), tt.bytes) {
				t.Errorf("WriteVLong(%d) = %v, want %v", tt.value, buf.Bytes(), tt.bytes)
			}

			got, err := ReadVLong(&buf)
			if err != nil {
				t.Fatalf("ReadVLong: %v", err)
			}
			if got != tt.value {
				t.Errorf("ReadVLong = %d, want %d", got, tt.value)
			}
		})
	}
}

func TestVIntRoundTrip(t *testing.T) {
	values := []int32{0, 1, 127, 128, 255, 256, 16383, 16384, 2097151, 2097152, 0x7FFFFFFF}
	for _, v := range values {
		var buf bytes.Buffer
		if err := WriteVInt(&buf, v); err != nil {
			t.Fatalf("WriteVInt(%d): %v", v, err)
		}
		got, err := ReadVInt(&buf)
		if err != nil {
			t.Fatalf("ReadVInt for %d: %v", v, err)
		}
		if got != v {
			t.Errorf("round trip %d: got %d", v, got)
		}
	}
}

func TestVLongRoundTrip(t *testing.T) {
	values := []int64{0, 1, 127, 128, 16384, 2097152, 268435456, 0x7FFFFFFFFFFFFFFF}
	for _, v := range values {
		var buf bytes.Buffer
		if err := WriteVLong(&buf, v); err != nil {
			t.Fatalf("WriteVLong(%d): %v", v, err)
		}
		got, err := ReadVLong(&buf)
		if err != nil {
			t.Fatalf("ReadVLong for %d: %v", v, err)
		}
		if got != v {
			t.Errorf("round trip %d: got %d", v, got)
		}
	}
}

func TestAppendVInt(t *testing.T) {
	tests := []struct {
		value int32
		want  []byte
	}{
		{0, []byte{0x00}},
		{300, []byte{0xAC, 0x02}},
	}
	for _, tt := range tests {
		got := AppendVInt(nil, tt.value)
		if !bytes.Equal(got, tt.want) {
			t.Errorf("AppendVInt(%d) = %v, want %v", tt.value, got, tt.want)
		}
	}
}

func TestReadVIntTooManyBytes(t *testing.T) {
	buf := bytes.NewReader([]byte{0x80, 0x80, 0x80, 0x80, 0x80, 0x80})
	_, err := ReadVInt(buf)
	if err == nil {
		t.Error("expected error for too many bytes")
	}
}
