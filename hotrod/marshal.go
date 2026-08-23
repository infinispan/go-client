package hotrod

import "infinispan.org/go-client/internal/codec"

// Media type identifiers for cache encoding.
const (
	MediaTypeJSON        = codec.MediaIDJSON
	MediaTypeTextPlain   = codec.MediaIDTextPlain
	MediaTypeProtostream = codec.MediaIDProtostream
	MediaTypeOctetStream = codec.MediaIDOctetStream
)

// Marshaller handles serialization of keys and values for cache operations.
type Marshaller interface {
	MarshalKey(key any) ([]byte, error)
	UnmarshalKey(data []byte, target any) error
	MarshalValue(value any) ([]byte, error)
	UnmarshalValue(data []byte, target any) error
	MediaType() int32
}
