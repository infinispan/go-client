package hotrod

import (
	"context"

	"infinispan.org/go-client/internal/operation"
)

// MultimapCache provides operations on an Infinispan multimap cache, where each key can map to multiple values.
type MultimapCache struct {
	client             *Client
	name               string
	supportsDuplicates bool
}

// Multimap returns a MultimapCache handle for the named multimap cache.
func (c *Client) Multimap(name string, opts ...MultimapOption) *MultimapCache {
	cfg := &multimapConfig{}
	for _, o := range opts {
		o(cfg)
	}
	return &MultimapCache{
		client:             c,
		name:               name,
		supportsDuplicates: cfg.supportsDuplicates,
	}
}

// Get retrieves all values associated with a key.
func (mc *MultimapCache) Get(ctx context.Context, key []byte) ([][]byte, error) {
	result, err := mc.client.pool.Execute(ctx, &operation.MultimapGetOp{
		Cache:              mc.name,
		Key:                key,
		SupportsDuplicates: mc.supportsDuplicates,
	})
	if err != nil {
		return nil, err
	}
	return result.([][]byte), nil
}

// MultimapMetadataCollection holds the values and server-side metadata for a multimap key.
type MultimapMetadataCollection struct {
	Values   [][]byte
	Created  int64
	Lifespan int32
	LastUsed int64
	MaxIdle  int32
	Version  int64
}

// GetWithMetadata retrieves all values and metadata (version, lifespan, timestamps) for a key.
func (mc *MultimapCache) GetWithMetadata(ctx context.Context, key []byte) (*MultimapMetadataCollection, bool, error) {
	result, err := mc.client.pool.Execute(ctx, &operation.MultimapGetWithMetadataOp{
		Cache:              mc.name,
		Key:                key,
		SupportsDuplicates: mc.supportsDuplicates,
	})
	if err != nil {
		return nil, false, err
	}
	resp := result.(*operation.MultimapGetWithMetadataResponse)
	if !resp.Found {
		return nil, false, nil
	}
	return &MultimapMetadataCollection{
		Values:   resp.Values,
		Created:  resp.Metadata.Created,
		Lifespan: resp.Metadata.Lifespan,
		LastUsed: resp.Metadata.LastUsed,
		MaxIdle:  resp.Metadata.MaxIdle,
		Version:  resp.Metadata.Version,
	}, true, nil
}

// Put adds one or more values for a key. Each value is stored as a separate entry.
func (mc *MultimapCache) Put(ctx context.Context, key []byte, values ...[]byte) error {
	return mc.PutWithOptions(ctx, key, values, nil)
}

// PutWithOptions adds one or more values for a key with expiration options.
func (mc *MultimapCache) PutWithOptions(ctx context.Context, key []byte, values [][]byte, opts []PutOption) error {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	for _, value := range values {
		_, err := mc.client.pool.Execute(ctx, &operation.MultimapPutOp{
			Cache:              mc.name,
			Key:                key,
			Value:              value,
			Lifespan:           cfg.lifespan,
			MaxIdle:            cfg.maxIdle,
			SupportsDuplicates: mc.supportsDuplicates,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// RemoveKey removes all values associated with a key. Returns true if the key existed.
func (mc *MultimapCache) RemoveKey(ctx context.Context, key []byte) (bool, error) {
	result, err := mc.client.pool.Execute(ctx, &operation.MultimapRemoveKeyOp{
		Cache:              mc.name,
		Key:                key,
		SupportsDuplicates: mc.supportsDuplicates,
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

// RemoveEntry removes a specific key-value pair. Returns true if the entry existed.
func (mc *MultimapCache) RemoveEntry(ctx context.Context, key, value []byte) (bool, error) {
	result, err := mc.client.pool.Execute(ctx, &operation.MultimapRemoveEntryOp{
		Cache:              mc.name,
		Key:                key,
		Value:              value,
		SupportsDuplicates: mc.supportsDuplicates,
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

// ContainsKey reports whether the multimap contains any values for the given key.
func (mc *MultimapCache) ContainsKey(ctx context.Context, key []byte) (bool, error) {
	result, err := mc.client.pool.Execute(ctx, &operation.MultimapContainsKeyOp{
		Cache:              mc.name,
		Key:                key,
		SupportsDuplicates: mc.supportsDuplicates,
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

// ContainsValue reports whether the multimap contains the given value under any key.
func (mc *MultimapCache) ContainsValue(ctx context.Context, value []byte) (bool, error) {
	result, err := mc.client.pool.Execute(ctx, &operation.MultimapContainsValueOp{
		Cache:              mc.name,
		Value:              value,
		SupportsDuplicates: mc.supportsDuplicates,
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

// ContainsEntry reports whether the multimap contains the specific key-value pair.
func (mc *MultimapCache) ContainsEntry(ctx context.Context, key, value []byte) (bool, error) {
	result, err := mc.client.pool.Execute(ctx, &operation.MultimapContainsEntryOp{
		Cache:              mc.name,
		Key:                key,
		Value:              value,
		SupportsDuplicates: mc.supportsDuplicates,
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

// Size returns the total number of key-value pairs in the multimap.
func (mc *MultimapCache) Size(ctx context.Context) (int64, error) {
	result, err := mc.client.pool.Execute(ctx, &operation.MultimapSizeOp{
		Cache:              mc.name,
		SupportsDuplicates: mc.supportsDuplicates,
	})
	if err != nil {
		return 0, err
	}
	return result.(int64), nil
}

// Name returns the cache name.
func (mc *MultimapCache) Name() string {
	return mc.name
}
