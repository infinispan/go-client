package hotrod

import (
	"context"

	"infinispan.org/go-client/internal/operation"
)

// IteratorMetadata holds the server-side metadata for an iterated cache entry.
type IteratorMetadata struct {
	Created  int64
	Lifespan int32
	LastUsed int64
	MaxIdle  int32
	Version  int64
}

// CloseableIterator iterates over cache entries fetched in batches from the
// server. It must be closed when no longer needed to release the server-side
// cursor. It is not safe for concurrent use.
type CloseableIterator struct {
	client      *Client
	cacheName   string
	iterationID string
	serverAddr  string
	ctx         context.Context
	metadata    bool

	buf    []operation.IterEntry
	pos    int
	more   bool
	closed bool
	err    error
}

// Next advances the iterator to the next entry. It returns false when the
// iteration is exhausted or an error occurred. Call Err to distinguish between
// the two.
func (it *CloseableIterator) Next() bool {
	if it.err != nil || it.closed {
		return false
	}
	it.pos++
	if it.pos < len(it.buf) {
		return true
	}
	if !it.more {
		return false
	}
	result, err := it.client.pool.ExecuteOn(it.ctx, it.serverAddr, &operation.IterationNextOp{
		Cache:           it.cacheName,
		IterationID:     it.iterationID,
		IncludeMetadata: it.metadata,
	})
	if err != nil {
		it.err = err
		return false
	}
	resp := result.(*operation.IterationNextResponse)
	it.buf = resp.Entries
	it.pos = 0
	it.more = resp.HasMore
	return len(it.buf) > 0
}

// Entry returns the key and value of the current entry. Must only be called
// after Next returns true.
func (it *CloseableIterator) Entry() (key, value []byte) {
	e := it.buf[it.pos]
	return e.Key, e.Value
}

// Metadata returns the server-side metadata of the current entry, or nil if
// the iterator was not created with WithIteratorMetadata.
func (it *CloseableIterator) Metadata() *IteratorMetadata {
	e := it.buf[it.pos]
	if e.Metadata == nil {
		return nil
	}
	return &IteratorMetadata{
		Created:  e.Metadata.Created,
		Lifespan: e.Metadata.Lifespan,
		LastUsed: e.Metadata.LastUsed,
		MaxIdle:  e.Metadata.MaxIdle,
		Version:  e.Metadata.Version,
	}
}

// Err returns the error that caused Next to return false, if any.
func (it *CloseableIterator) Err() error {
	return it.err
}

// Close releases the server-side iteration cursor. It is safe to call multiple
// times. If the iteration finished naturally, Close is a no-op.
func (it *CloseableIterator) Close() error {
	if it.closed {
		return nil
	}
	it.closed = true
	if !it.more {
		return nil
	}
	_, err := it.client.pool.ExecuteOn(it.ctx, it.serverAddr, &operation.IterationEndOp{
		Cache:       it.cacheName,
		IterationID: it.iterationID,
	})
	return err
}

// Iterator returns a CloseableIterator that fetches all entries from the cache
// in batches. The caller must close the iterator when done.
func (rc *RemoteCache) Iterator(ctx context.Context, opts ...IteratorOption) (*CloseableIterator, error) {
	cfg := &iteratorConfig{batchSize: 1000}
	for _, o := range opts {
		o(cfg)
	}

	result, addr, err := rc.client.pool.ExecuteWithAddr(ctx, &operation.IterationStartOp{
		Cache:           rc.name,
		BatchSize:       int32(cfg.batchSize),
		IncludeMetadata: cfg.includeMetadata,
	})
	if err != nil {
		return nil, err
	}
	iterationID := result.(string)

	return &CloseableIterator{
		client:      rc.client,
		cacheName:   rc.name,
		iterationID: iterationID,
		serverAddr:  addr,
		ctx:         ctx,
		metadata:    cfg.includeMetadata,
		pos:         -1,
		more:        true,
	}, nil
}
