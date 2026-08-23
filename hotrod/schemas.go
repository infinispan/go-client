package hotrod

import "context"

const protobufMetadataCache = "___protobuf_metadata"

// SchemaManager handles protobuf schema registration with the Infinispan server.
type SchemaManager struct {
	client *Client
}

// Schemas returns a SchemaManager for registering .proto schemas.
func (c *Client) Schemas() *SchemaManager {
	return &SchemaManager{client: c}
}

// Register uploads a .proto schema to the server's protobuf metadata cache.
// name is the schema filename (e.g., "person.proto").
// content is the raw .proto file content.
func (sm *SchemaManager) Register(ctx context.Context, name string, content string) error {
	cache := sm.client.Cache(protobufMetadataCache).WithEncoding(MediaTypeTextPlain)
	return cache.Put(ctx, []byte(name), []byte(content))
}

// Get retrieves a registered schema by name.
func (sm *SchemaManager) Get(ctx context.Context, name string) (string, bool, error) {
	cache := sm.client.Cache(protobufMetadataCache).WithEncoding(MediaTypeTextPlain)
	data, found, err := cache.Get(ctx, []byte(name))
	if err != nil {
		return "", false, err
	}
	return string(data), found, nil
}
