package hotrod

// Marshaller handles serialization of keys and values for cache operations.
type Marshaller interface {
	MarshalKey(key any) ([]byte, error)
	UnmarshalKey(data []byte, target any) error
	MarshalValue(value any) ([]byte, error)
	UnmarshalValue(data []byte, target any) error
	MediaType() int32
}
