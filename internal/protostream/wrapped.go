package protostream

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Protobuf wire types.
const (
	wireVarint          = 0
	wireFixed64         = 1
	wireLengthDelimited = 2
	wireFixed32         = 5
)

// WrappedMessage field numbers (from org.infinispan.protostream.WrappedMessage).
const (
	fieldWrappedDouble   = 1
	fieldWrappedFloat    = 2
	fieldWrappedInt64    = 3
	fieldWrappedUInt64   = 4
	fieldWrappedInt32    = 5
	fieldWrappedFixed64  = 6
	fieldWrappedFixed32  = 7
	fieldWrappedBool     = 8
	fieldWrappedString   = 9
	fieldWrappedBytes    = 10
	fieldWrappedUInt32   = 11
	fieldWrappedSFixed32 = 12
	fieldWrappedSFixed64 = 13
	fieldWrappedSInt32   = 14
	fieldWrappedSInt64   = 15

	fieldWrappedDescriptorFullName = 16
	fieldWrappedMessage            = 17
	fieldWrappedDescriptorTypeID   = 19

	fieldWrappedContainerSize     = 27
	fieldWrappedContainerTypeName = 28
	fieldWrappedContainerMessage  = 30
)

// WrapMessage wraps already-serialized protobuf message bytes with its type name.
func WrapMessage(messageBytes []byte, typeName string) []byte {
	size := tagSize(fieldWrappedDescriptorFullName) + lenDelimitedSize(len(typeName)) +
		tagSize(fieldWrappedMessage) + lenDelimitedSize(len(messageBytes))
	dst := make([]byte, 0, size)
	dst = appendLenDelimited(dst, fieldWrappedDescriptorFullName, []byte(typeName))
	dst = appendLenDelimited(dst, fieldWrappedMessage, messageBytes)
	return dst
}

// WrapString wraps a string value in a WrappedMessage (field 9).
func WrapString(s string) []byte {
	size := tagSize(fieldWrappedString) + lenDelimitedSize(len(s))
	dst := make([]byte, 0, size)
	dst = appendLenDelimited(dst, fieldWrappedString, []byte(s))
	return dst
}

// WrapInt32 wraps an int32 value in a WrappedMessage (field 5).
func WrapInt32(v int32) []byte {
	dst := make([]byte, 0, tagSize(fieldWrappedInt32)+binary.MaxVarintLen32)
	dst = appendVarintField(dst, fieldWrappedInt32, uint64(v))
	return dst
}

// WrapInt64 wraps an int64 value in a WrappedMessage (field 3).
func WrapInt64(v int64) []byte {
	dst := make([]byte, 0, tagSize(fieldWrappedInt64)+binary.MaxVarintLen64)
	dst = appendVarintField(dst, fieldWrappedInt64, uint64(v))
	return dst
}

// WrapByTypeID wraps already-serialized protobuf message bytes with a numeric type ID
// (field 19) instead of a type name (field 16). Used for query request/response messages.
func WrapByTypeID(messageBytes []byte, typeID int32) []byte {
	size := tagSize(fieldWrappedDescriptorTypeID) + uvarintSize(uint64(typeID)) +
		tagSize(fieldWrappedMessage) + lenDelimitedSize(len(messageBytes))
	dst := make([]byte, 0, size)
	dst = appendVarintField(dst, fieldWrappedDescriptorTypeID, uint64(typeID))
	dst = appendLenDelimited(dst, fieldWrappedMessage, messageBytes)
	return dst
}

// WrapBool wraps a bool value in a WrappedMessage (field 8).
func WrapBool(v bool) []byte {
	val := uint64(0)
	if v {
		val = 1
	}
	dst := make([]byte, 0, tagSize(fieldWrappedBool)+1)
	dst = appendVarintField(dst, fieldWrappedBool, val)
	return dst
}

// WrapDouble wraps a float64 value in a WrappedMessage (field 1).
func WrapDouble(v float64) []byte {
	dst := make([]byte, 0, tagSize(fieldWrappedDouble)+8)
	dst = appendTag(dst, fieldWrappedDouble, wireFixed64)
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v))
	dst = append(dst, buf[:]...)
	return dst
}

// WrapBytes wraps a byte slice in a WrappedMessage (field 10).
func WrapBytes(v []byte) []byte {
	size := tagSize(fieldWrappedBytes) + lenDelimitedSize(len(v))
	dst := make([]byte, 0, size)
	dst = appendLenDelimited(dst, fieldWrappedBytes, v)
	return dst
}

// WrapUInt32 wraps a uint32 value in a WrappedMessage (field 11).
func WrapUInt32(v uint32) []byte {
	dst := make([]byte, 0, tagSize(fieldWrappedUInt32)+binary.MaxVarintLen32)
	dst = appendVarintField(dst, fieldWrappedUInt32, uint64(v))
	return dst
}

// WrapUInt64 wraps a uint64 value in a WrappedMessage (field 4).
func WrapUInt64(v uint64) []byte {
	dst := make([]byte, 0, tagSize(fieldWrappedUInt64)+binary.MaxVarintLen64)
	dst = appendVarintField(dst, fieldWrappedUInt64, v)
	return dst
}

// WrapFloat wraps a float32 value in a WrappedMessage (field 2 = wrappedFloat).
func WrapFloat(v float32) []byte {
	dst := make([]byte, 0, tagSize(fieldWrappedFloat)+4)
	dst = appendTag(dst, fieldWrappedFloat, wireFixed32)
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
	dst = append(dst, buf[:]...)
	return dst
}

const floatArrayTypeName = "org.infinispan.protostream.commons.FloatArray"

// WrapFloatArray wraps a []float32 in a WrappedMessage using the container encoding.
func WrapFloatArray(values []float32) []byte {
	dst := appendLenDelimited(nil, fieldWrappedContainerTypeName, []byte(floatArrayTypeName))
	dst = appendVarintField(dst, fieldWrappedContainerSize, uint64(len(values)))
	dst = appendLenDelimited(dst, fieldWrappedContainerMessage, []byte{})
	for _, v := range values {
		dst = appendTag(dst, fieldWrappedFloat, wireFixed32)
		var buf [4]byte
		binary.LittleEndian.PutUint32(buf[:], math.Float32bits(v))
		dst = append(dst, buf[:]...)
	}
	return dst
}

// UnwrapMessage extracts the inner message bytes and type name from a WrappedMessage.
func UnwrapMessage(data []byte) (messageBytes []byte, typeName string, err error) {
	var foundMsg, foundName bool
	err = ScanFields(data, func(fieldNumber int, wireType int, value []byte) error {
		switch fieldNumber {
		case fieldWrappedMessage:
			if wireType != wireLengthDelimited {
				return fmt.Errorf("wrappedMessage: expected wire type 2, got %d", wireType)
			}
			messageBytes = value
			foundMsg = true
		case fieldWrappedDescriptorFullName:
			if wireType != wireLengthDelimited {
				return fmt.Errorf("wrappedDescriptorFullName: expected wire type 2, got %d", wireType)
			}
			typeName = string(value)
			foundName = true
		}
		return nil
	})
	if err != nil {
		return nil, "", err
	}
	if !foundMsg {
		return nil, "", errors.New("WrappedMessage: missing wrappedMessage field (17)")
	}
	if !foundName {
		return nil, "", errors.New("WrappedMessage: missing wrappedDescriptorFullName field (16)")
	}
	return messageBytes, typeName, nil
}

// UnwrapString extracts a string from a WrappedMessage (field 9).
func UnwrapString(data []byte) (string, error) {
	var result string
	var found bool
	err := ScanFields(data, func(fieldNumber int, wireType int, value []byte) error {
		if fieldNumber == fieldWrappedString {
			if wireType != wireLengthDelimited {
				return fmt.Errorf("wrappedString: expected wire type 2, got %d", wireType)
			}
			result = string(value)
			found = true
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if !found {
		return "", errors.New("WrappedMessage: missing wrappedString field (9)")
	}
	return result, nil
}

func appendTag(dst []byte, fieldNumber int, wireType int) []byte {
	return appendUvarint(dst, uint64(fieldNumber<<3|wireType))
}

func appendLenDelimited(dst []byte, fieldNumber int, data []byte) []byte {
	dst = appendTag(dst, fieldNumber, wireLengthDelimited)
	dst = appendUvarint(dst, uint64(len(data)))
	dst = append(dst, data...)
	return dst
}

func appendVarintField(dst []byte, fieldNumber int, value uint64) []byte {
	dst = appendTag(dst, fieldNumber, wireVarint)
	dst = appendUvarint(dst, value)
	return dst
}

func appendUvarint(dst []byte, v uint64) []byte {
	for v >= 0x80 {
		dst = append(dst, byte(v)|0x80)
		v >>= 7
	}
	dst = append(dst, byte(v))
	return dst
}

func tagSize(fieldNumber int) int {
	return uvarintSize(uint64(fieldNumber << 3))
}

func lenDelimitedSize(dataLen int) int {
	return uvarintSize(uint64(dataLen)) + dataLen
}

func uvarintSize(v uint64) int {
	n := 1
	for v >= 0x80 {
		v >>= 7
		n++
	}
	return n
}

// UnwrapBytes extracts the inner message bytes (field 17) from a WrappedMessage
// without requiring the type name (field 16) or type ID (field 19) to be present.
func UnwrapBytes(data []byte) ([]byte, error) {
	var msgBytes []byte
	err := ScanFields(data, func(fieldNumber int, wireType int, value []byte) error {
		if fieldNumber == fieldWrappedMessage {
			if wireType != wireLengthDelimited {
				return fmt.Errorf("wrappedMessage: expected wire type 2, got %d", wireType)
			}
			msgBytes = value
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if msgBytes == nil {
		return nil, errors.New("WrappedMessage: missing wrappedMessage field (17)")
	}
	return msgBytes, nil
}

// UnwrapValue extracts a typed Go value from a WrappedMessage by inspecting which
// field is set. Returns float64, float32, int64, uint64, int32, uint32, bool, string, or []byte.
func UnwrapValue(data []byte) (any, error) {
	var result any
	err := ScanFields(data, func(fieldNumber int, wireType int, value []byte) error {
		switch fieldNumber {
		case fieldWrappedDouble: // 1 - fixed64
			if len(value) != 8 {
				return fmt.Errorf("UnwrapValue: expected 8 bytes for double, got %d", len(value))
			}
			result = math.Float64frombits(binary.LittleEndian.Uint64(value))
		case fieldWrappedFloat: // 2 - fixed32
			if len(value) != 4 {
				return fmt.Errorf("UnwrapValue: expected 4 bytes for float, got %d", len(value))
			}
			result = math.Float32frombits(binary.LittleEndian.Uint32(value))
		case fieldWrappedInt64: // 3 - varint
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("UnwrapValue: invalid int64 varint")
			}
			result = int64(v)
		case fieldWrappedUInt64: // 4 - varint
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("UnwrapValue: invalid uint64 varint")
			}
			result = v
		case fieldWrappedInt32: // 5 - varint
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("UnwrapValue: invalid int32 varint")
			}
			result = int32(v)
		case fieldWrappedFixed64: // 6 - fixed64
			if len(value) != 8 {
				return fmt.Errorf("UnwrapValue: expected 8 bytes for fixed64, got %d", len(value))
			}
			result = binary.LittleEndian.Uint64(value)
		case fieldWrappedFixed32: // 7 - fixed32
			if len(value) != 4 {
				return fmt.Errorf("UnwrapValue: expected 4 bytes for fixed32, got %d", len(value))
			}
			result = binary.LittleEndian.Uint32(value)
		case fieldWrappedBool: // 8 - varint
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("UnwrapValue: invalid bool varint")
			}
			result = v != 0
		case fieldWrappedString: // 9
			result = string(value)
		case fieldWrappedBytes: // 10
			result = append([]byte(nil), value...)
		case fieldWrappedUInt32: // 11 - varint
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("UnwrapValue: invalid uint32 varint")
			}
			result = uint32(v)
		case fieldWrappedSFixed32: // 12 - fixed32
			if len(value) != 4 {
				return fmt.Errorf("UnwrapValue: expected 4 bytes for sfixed32, got %d", len(value))
			}
			result = int32(binary.LittleEndian.Uint32(value))
		case fieldWrappedSFixed64: // 13 - fixed64
			if len(value) != 8 {
				return fmt.Errorf("UnwrapValue: expected 8 bytes for sfixed64, got %d", len(value))
			}
			result = int64(binary.LittleEndian.Uint64(value))
		case fieldWrappedSInt32: // 14 - varint (zigzag)
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("UnwrapValue: invalid sint32 varint")
			}
			result = decodeZigZag32(v)
		case fieldWrappedSInt64: // 15 - varint (zigzag)
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("UnwrapValue: invalid sint64 varint")
			}
			result = decodeZigZag64(v)
		case fieldWrappedMessage: // 17
			result = append([]byte(nil), value...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, errors.New("UnwrapValue: no recognized value field found")
	}
	return result, nil
}

func decodeZigZag32(v uint64) int32 {
	return int32(uint32(v)>>1) ^ -int32(uint32(v)&1)
}

func decodeZigZag64(v uint64) int64 {
	return int64(v>>1) ^ -int64(v&1)
}

// CQResultType represents the type of a continuous query result.
type CQResultType byte

const (
	CQJoining CQResultType = 1
	CQUpdated CQResultType = 2
	CQLeaving CQResultType = 3
)

// CQResult represents a decoded ContinuousQueryResult protobuf message.
type CQResult struct {
	ResultType  CQResultType
	Key         []byte
	Value       []byte
	Projections [][]byte
}

// DecodeCQResult decodes a ContinuousQueryResult protobuf message.
func DecodeCQResult(data []byte) (*CQResult, error) {
	r := &CQResult{}
	var foundType, foundKey bool
	err := ScanFields(data, func(fieldNumber int, wireType int, value []byte) error {
		switch fieldNumber {
		case 1: // resultType
			v, n := decodeUvarint(value)
			if n <= 0 {
				return errors.New("CQResult: invalid resultType varint")
			}
			r.ResultType = CQResultType(v)
			foundType = true
		case 2: // key
			r.Key = value
			foundKey = true
		case 3: // value
			r.Value = value
		case 4: // projection (repeated)
			r.Projections = append(r.Projections, value)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if !foundType {
		return nil, errors.New("CQResult: missing resultType field (1)")
	}
	if !foundKey {
		return nil, errors.New("CQResult: missing key field (2)")
	}
	return r, nil
}

// ScanFields iterates over protobuf fields in data, calling fn for each.
// For length-delimited fields, value contains the raw bytes.
// For varint fields, value contains the varint-encoded bytes.
func ScanFields(data []byte, fn func(fieldNumber, wireType int, value []byte) error) error {
	pos := 0
	for pos < len(data) {
		tag, n := decodeUvarint(data[pos:])
		if n <= 0 {
			return errors.New("invalid tag varint")
		}
		pos += n

		fieldNumber := int(tag >> 3)
		wireType := int(tag & 0x7)

		switch wireType {
		case wireVarint:
			start := pos
			val, n := decodeUvarint(data[pos:])
			if n <= 0 {
				return fmt.Errorf("field %d: invalid varint", fieldNumber)
			}
			_ = val
			pos += n
			if err := fn(fieldNumber, wireType, data[start:pos]); err != nil {
				return err
			}
		case wireFixed64:
			if pos+8 > len(data) {
				return fmt.Errorf("field %d: truncated fixed64", fieldNumber)
			}
			if err := fn(fieldNumber, wireType, data[pos:pos+8]); err != nil {
				return err
			}
			pos += 8
		case wireLengthDelimited:
			length, n := decodeUvarint(data[pos:])
			if n <= 0 {
				return fmt.Errorf("field %d: invalid length varint", fieldNumber)
			}
			pos += n
			if pos+int(length) > len(data) {
				return fmt.Errorf("field %d: truncated data (need %d, have %d)", fieldNumber, length, len(data)-pos)
			}
			if err := fn(fieldNumber, wireType, data[pos:pos+int(length)]); err != nil {
				return err
			}
			pos += int(length)
		case wireFixed32:
			if pos+4 > len(data) {
				return fmt.Errorf("field %d: truncated fixed32", fieldNumber)
			}
			if err := fn(fieldNumber, wireType, data[pos:pos+4]); err != nil {
				return err
			}
			pos += 4
		default:
			return fmt.Errorf("unsupported wire type %d for field %d", wireType, fieldNumber)
		}
	}
	return nil
}

func decodeUvarint(data []byte) (uint64, int) {
	var v uint64
	for i, b := range data {
		if i >= binary.MaxVarintLen64 {
			return 0, -1
		}
		v |= uint64(b&0x7F) << (7 * i)
		if b < 0x80 {
			return v, i + 1
		}
	}
	return 0, -1
}
