package hotrod

import (
	"fmt"

	"google.golang.org/protobuf/proto"

	"infinispan.org/go-client/internal/codec"
	"infinispan.org/go-client/internal/protostream"
)

type protoStreamMarshaller struct{}

// ProtoStreamMarshaller returns a Marshaller that encodes values as ProtoStream
// WrappedMessage envelopes, compatible with Infinispan's server-side indexing
// and Ickle queries.
//
// Keys can be string, int32, int64, or proto.Message.
// Values must implement proto.Message.
func ProtoStreamMarshaller() Marshaller {
	return &protoStreamMarshaller{}
}

func (m *protoStreamMarshaller) MediaType() int32 {
	return codec.MediaIDProtostream
}

func (m *protoStreamMarshaller) MarshalKey(key any) ([]byte, error) {
	switch k := key.(type) {
	case string:
		return protostream.WrapString(k), nil
	case int32:
		return protostream.WrapInt32(k), nil
	case int64:
		return protostream.WrapInt64(k), nil
	case proto.Message:
		b, err := proto.Marshal(k)
		if err != nil {
			return nil, fmt.Errorf("marshal proto key: %w", err)
		}
		typeName := string(k.ProtoReflect().Descriptor().FullName())
		return protostream.WrapMessage(b, typeName), nil
	default:
		return nil, fmt.Errorf("unsupported key type: %T (expected string, int32, int64, or proto.Message)", key)
	}
}

func (m *protoStreamMarshaller) UnmarshalKey(data []byte, target any) error {
	switch t := target.(type) {
	case *string:
		s, err := protostream.UnwrapString(data)
		if err != nil {
			return err
		}
		*t = s
		return nil
	case proto.Message:
		msgBytes, _, err := protostream.UnwrapMessage(data)
		if err != nil {
			return err
		}
		return proto.Unmarshal(msgBytes, t)
	default:
		return fmt.Errorf("unsupported key target type: %T", target)
	}
}

func (m *protoStreamMarshaller) MarshalValue(value any) ([]byte, error) {
	msg, ok := value.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("value must implement proto.Message, got %T", value)
	}
	b, err := proto.Marshal(msg)
	if err != nil {
		return nil, fmt.Errorf("marshal proto value: %w", err)
	}
	typeName := string(msg.ProtoReflect().Descriptor().FullName())
	return protostream.WrapMessage(b, typeName), nil
}

func (m *protoStreamMarshaller) UnmarshalValue(data []byte, target any) error {
	msg, ok := target.(proto.Message)
	if !ok {
		return fmt.Errorf("target must implement proto.Message, got %T", target)
	}
	msgBytes, _, err := protostream.UnwrapMessage(data)
	if err != nil {
		return err
	}
	return proto.Unmarshal(msgBytes, msg)
}
