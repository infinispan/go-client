package codec

import (
	"fmt"
	"io"
)

func WriteVInt(w io.Writer, v int32) error {
	u := uint32(v)
	var buf [5]byte
	n := 0
	for u&^0x7F != 0 {
		buf[n] = byte(u&0x7F) | 0x80
		u >>= 7
		n++
	}
	buf[n] = byte(u)
	n++
	_, err := w.Write(buf[:n])
	return err
}

func WriteVLong(w io.Writer, v int64) error {
	u := uint64(v)
	var buf [9]byte
	n := 0
	for u&^0x7F != 0 {
		buf[n] = byte(u&0x7F) | 0x80
		u >>= 7
		n++
	}
	buf[n] = byte(u)
	n++
	_, err := w.Write(buf[:n])
	return err
}

func ReadVInt(r io.Reader) (int32, error) {
	var b [1]byte
	var result uint32
	var shift uint
	for range 5 {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, fmt.Errorf("read vint: %w", err)
		}
		result |= uint32(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			return int32(result), nil
		}
		shift += 7
	}
	return 0, fmt.Errorf("read vint: too many bytes")
}

func ReadVLong(r io.Reader) (int64, error) {
	var b [1]byte
	var result uint64
	var shift uint
	for range 9 {
		if _, err := io.ReadFull(r, b[:]); err != nil {
			return 0, fmt.Errorf("read vlong: %w", err)
		}
		result |= uint64(b[0]&0x7F) << shift
		if b[0]&0x80 == 0 {
			return int64(result), nil
		}
		shift += 7
	}
	return 0, fmt.Errorf("read vlong: too many bytes")
}

func AppendVInt(dst []byte, v int32) []byte {
	u := uint32(v)
	for u&^0x7F != 0 {
		dst = append(dst, byte(u&0x7F)|0x80)
		u >>= 7
	}
	return append(dst, byte(u))
}

func AppendVLong(dst []byte, v int64) []byte {
	u := uint64(v)
	for u&^0x7F != 0 {
		dst = append(dst, byte(u&0x7F)|0x80)
		u >>= 7
	}
	return append(dst, byte(u))
}
