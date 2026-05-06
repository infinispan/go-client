package codec

import (
	"bytes"
	"testing"
)

func TestLPBytesRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single", []byte{0x42}},
		{"hello", []byte("hello")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteLPBytes(&buf, tt.data); err != nil {
				t.Fatalf("WriteLPBytes: %v", err)
			}
			got, err := ReadLPBytes(&buf)
			if err != nil {
				t.Fatalf("ReadLPBytes: %v", err)
			}
			if !bytes.Equal(got, tt.data) {
				t.Errorf("got %v, want %v", got, tt.data)
			}
		})
	}
}

func TestLPStringRoundTrip(t *testing.T) {
	tests := []string{"", "hello", "infinispan", "utf8: äöü"}
	for _, s := range tests {
		t.Run(s, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteLPString(&buf, s); err != nil {
				t.Fatalf("WriteLPString: %v", err)
			}
			got, err := ReadLPString(&buf)
			if err != nil {
				t.Fatalf("ReadLPString: %v", err)
			}
			if got != s {
				t.Errorf("got %q, want %q", got, s)
			}
		})
	}
}

func TestWriteMediaTypeNone(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMediaTypeNone(&buf); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(buf.Bytes(), []byte{0x00}) {
		t.Errorf("got %v, want [0x00]", buf.Bytes())
	}
}

func TestWriteMediaTypePredefined(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteMediaTypePredefined(&buf, MediaIDOctetStream); err != nil {
		t.Fatal(err)
	}
	// kind=0x01, predefinedId=vInt(3)=0x03, paramCount=vInt(0)=0x00
	want := []byte{0x01, 0x03, 0x00}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Errorf("got %v, want %v", buf.Bytes(), want)
	}
}

func TestSkipMediaTypeNone(t *testing.T) {
	buf := bytes.NewReader([]byte{0x00})
	if err := SkipMediaType(buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected all bytes consumed, %d remaining", buf.Len())
	}
}

func TestSkipMediaTypePredefined(t *testing.T) {
	data := []byte{0x01, 0x03, 0x00} // kind=predefined, id=3, 0 params
	buf := bytes.NewReader(data)
	if err := SkipMediaType(buf); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected all bytes consumed, %d remaining", buf.Len())
	}
}

func TestU2RoundTrip(t *testing.T) {
	tests := []uint16{0, 1, 255, 256, 11222, 65535}
	for _, v := range tests {
		var buf bytes.Buffer
		if err := WriteU2(&buf, v); err != nil {
			t.Fatal(err)
		}
		got, err := ReadU2(&buf)
		if err != nil {
			t.Fatal(err)
		}
		if got != v {
			t.Errorf("round trip %d: got %d", v, got)
		}
	}
}
