package codec

import (
	"fmt"
	"io"
)

func WriteLPBytes(w io.Writer, b []byte) error {
	if err := WriteVInt(w, int32(len(b))); err != nil {
		return err
	}
	if len(b) > 0 {
		_, err := w.Write(b)
		return err
	}
	return nil
}

func ReadLPBytes(r io.Reader) ([]byte, error) {
	n, err := ReadVInt(r)
	if err != nil {
		return nil, fmt.Errorf("read lp_bytes length: %w", err)
	}
	if n == 0 {
		return []byte{}, nil
	}
	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read lp_bytes data: %w", err)
	}
	return buf, nil
}

func WriteLPString(w io.Writer, s string) error {
	return WriteLPBytes(w, []byte(s))
}

func ReadLPString(r io.Reader) (string, error) {
	b, err := ReadLPBytes(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func WriteMediaTypeNone(w io.Writer) error {
	_, err := w.Write([]byte{MediaTypeNone})
	return err
}

func WriteMediaTypePredefined(w io.Writer, id int32) error {
	if _, err := w.Write([]byte{MediaTypePredefined}); err != nil {
		return err
	}
	if err := WriteVInt(w, id); err != nil {
		return err
	}
	return WriteVInt(w, 0)
}

func WriteU1(w io.Writer, v byte) error {
	_, err := w.Write([]byte{v})
	return err
}

func ReadU1(r io.Reader) (byte, error) {
	var b [1]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return b[0], nil
}

func WriteU2(w io.Writer, v uint16) error {
	_, err := w.Write([]byte{byte(v >> 8), byte(v)})
	return err
}

func ReadU2(r io.Reader) (uint16, error) {
	var b [2]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, err
	}
	return uint16(b[0])<<8 | uint16(b[1]), nil
}

func WriteLong(w io.Writer, v int64) error {
	b := [8]byte{
		byte(v >> 56), byte(v >> 48), byte(v >> 40), byte(v >> 32),
		byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v),
	}
	_, err := w.Write(b[:])
	return err
}

func ReadLong(r io.Reader) (int64, error) {
	var b [8]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, fmt.Errorf("read long: %w", err)
	}
	v := int64(b[0])<<56 | int64(b[1])<<48 | int64(b[2])<<40 | int64(b[3])<<32 |
		int64(b[4])<<24 | int64(b[5])<<16 | int64(b[6])<<8 | int64(b[7])
	return v, nil
}

func WriteInt32(w io.Writer, v int32) error {
	b := [4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
	_, err := w.Write(b[:])
	return err
}

func ReadInt32(r io.Reader) (int32, error) {
	var b [4]byte
	if _, err := io.ReadFull(r, b[:]); err != nil {
		return 0, fmt.Errorf("read int32: %w", err)
	}
	return int32(b[0])<<24 | int32(b[1])<<16 | int32(b[2])<<8 | int32(b[3]), nil
}

func WriteSignedVInt(w io.Writer, v int32) error {
	return WriteVInt(w, (v<<1)^(v>>31))
}

func ReadSignedVInt(r io.Reader) (int32, error) {
	raw, err := ReadVInt(r)
	if err != nil {
		return 0, err
	}
	u := uint32(raw)
	return int32(u>>1) ^ -int32(u&1), nil
}

const (
	flagInfiniteLifespan byte = 0x01
	flagInfiniteMaxIdle  byte = 0x02
)

type EntryMetadata struct {
	Created  int64
	Lifespan int32
	LastUsed int64
	MaxIdle  int32
	Version  int64
}

func ReadMetadata(r io.Reader) (*EntryMetadata, []byte, error) {
	flags, err := ReadU1(r)
	if err != nil {
		return nil, nil, fmt.Errorf("read metadata flags: %w", err)
	}
	var md EntryMetadata
	if flags&flagInfiniteLifespan == 0 {
		if md.Created, err = ReadLong(r); err != nil {
			return nil, nil, fmt.Errorf("read creation: %w", err)
		}
		n, err := ReadVInt(r)
		if err != nil {
			return nil, nil, fmt.Errorf("read lifespan: %w", err)
		}
		md.Lifespan = n
	} else {
		md.Lifespan = -1
	}
	if flags&flagInfiniteMaxIdle == 0 {
		if md.LastUsed, err = ReadLong(r); err != nil {
			return nil, nil, fmt.Errorf("read lastUsed: %w", err)
		}
		n, err := ReadVInt(r)
		if err != nil {
			return nil, nil, fmt.Errorf("read maxIdle: %w", err)
		}
		md.MaxIdle = n
	} else {
		md.MaxIdle = -1
	}
	if md.Version, err = ReadLong(r); err != nil {
		return nil, nil, fmt.Errorf("read version: %w", err)
	}
	value, err := ReadLPBytes(r)
	if err != nil {
		return nil, nil, err
	}
	return &md, value, nil
}

func ReadMetadataValue(r io.Reader) ([]byte, error) {
	flags, err := ReadU1(r)
	if err != nil {
		return nil, fmt.Errorf("read metadata flags: %w", err)
	}
	if flags&flagInfiniteLifespan == 0 {
		if _, err := ReadLong(r); err != nil {
			return nil, fmt.Errorf("read creation: %w", err)
		}
		if _, err := ReadVInt(r); err != nil {
			return nil, fmt.Errorf("read lifespan: %w", err)
		}
	}
	if flags&flagInfiniteMaxIdle == 0 {
		if _, err := ReadLong(r); err != nil {
			return nil, fmt.Errorf("read lastUsed: %w", err)
		}
		if _, err := ReadVInt(r); err != nil {
			return nil, fmt.Errorf("read maxIdle: %w", err)
		}
	}
	if _, err := ReadLong(r); err != nil {
		return nil, fmt.Errorf("read version: %w", err)
	}
	return ReadLPBytes(r)
}

func SkipMediaType(r io.Reader) error {
	kind, err := ReadU1(r)
	if err != nil {
		return fmt.Errorf("skip media type kind: %w", err)
	}
	if kind == MediaTypeNone {
		return nil
	}
	if kind == MediaTypePredefined {
		if _, err := ReadVInt(r); err != nil {
			return fmt.Errorf("skip media type predefined id: %w", err)
		}
	} else if kind == MediaTypeCustom {
		if _, err := ReadLPString(r); err != nil {
			return fmt.Errorf("skip media type custom string: %w", err)
		}
	}
	paramCount, err := ReadVInt(r)
	if err != nil {
		return fmt.Errorf("skip media type param count: %w", err)
	}
	for i := int32(0); i < paramCount; i++ {
		if _, err := ReadLPString(r); err != nil {
			return fmt.Errorf("skip media type param key: %w", err)
		}
		if _, err := ReadLPString(r); err != nil {
			return fmt.Errorf("skip media type param value: %w", err)
		}
	}
	return nil
}
