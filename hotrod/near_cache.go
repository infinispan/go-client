package hotrod

import (
	"context"
	"crypto/rand"
	"sync/atomic"

	"infinispan.org/go-client/internal/hash"
	"infinispan.org/go-client/internal/operation"
)

// NearCacheOption configures a NearCache.
type NearCacheOption func(*nearCacheConfig)

type nearCacheConfig struct {
	maxEntries int
}

// WithMaxNearCacheEntries sets the maximum number of entries in the near cache.
func WithMaxNearCacheEntries(n int) NearCacheOption {
	return func(cfg *nearCacheConfig) {
		cfg.maxEntries = n
	}
}

// NearCache is a client-side LRU cache backed by a remote Infinispan cache with bloom-filter-based invalidation.
type NearCache struct {
	remote          *RemoteCache
	store           *lruStore
	listenerID      []byte
	bloomFilterBits int
	removalCount    atomic.Int32
	bloomDirty      atomic.Bool
	updateThreshold int32
	done            chan struct{}
	events          chan *CacheEntryEvent
}

// NewNearCache creates a near cache for the named remote cache with bloom-filter-based invalidation.
func NewNearCache(ctx context.Context, client *Client, cacheName string, opts ...NearCacheOption) (*NearCache, error) {
	cfg := &nearCacheConfig{maxEntries: 1000}
	for _, o := range opts {
		o(cfg)
	}

	remote := client.Cache(cacheName)
	bloomFilterBits := cfg.maxEntries * 4
	updateThreshold := int32(cfg.maxEntries/16 + 3)

	nc := &NearCache{
		remote:          remote,
		bloomFilterBits: bloomFilterBits,
		updateThreshold: updateThreshold,
		done:            make(chan struct{}),
		events:          make(chan *CacheEntryEvent, 64),
	}

	nc.store = newLRU(cfg.maxEntries, nc.onEntryRemoved)

	listenerID := make([]byte, 16)
	if _, err := rand.Read(listenerID); err != nil {
		return nil, err
	}
	nc.listenerID = listenerID

	op := &operation.AddBloomNearCacheListenerOp{
		Cache:           cacheName,
		ListenerID:      listenerID,
		BloomFilterBits: int32(bloomFilterBits),
	}

	if err := client.pool.AddBloomListener(ctx, op, nc.events); err != nil {
		return nil, err
	}

	go nc.invalidationLoop(client)

	return nc, nil
}

// Get retrieves a value, returning it from the local cache if present, otherwise fetching from the server.
func (nc *NearCache) Get(ctx context.Context, key []byte) ([]byte, bool, error) {
	if val, ok := nc.store.Get(string(key)); ok {
		return val, true, nil
	}
	val, found, err := nc.remote.Get(ctx, key)
	if err != nil {
		return nil, false, err
	}
	if found {
		nc.store.Put(string(key), val)
		nc.scheduleBloomUpdate()
	}
	return val, found, nil
}

// Put stores a key-value pair on the server and invalidates the local near cache entry.
func (nc *NearCache) Put(ctx context.Context, key, value []byte, opts ...PutOption) error {
	if err := nc.remote.Put(ctx, key, value, opts...); err != nil {
		return err
	}
	nc.store.Remove(string(key))
	return nil
}

// Remove deletes an entry from the server and invalidates the local near cache entry.
func (nc *NearCache) Remove(ctx context.Context, key []byte, opts ...RemoveOption) error {
	if err := nc.remote.Remove(ctx, key, opts...); err != nil {
		return err
	}
	nc.store.Remove(string(key))
	return nil
}

// Close removes the server-side listener and clears the local cache.
func (nc *NearCache) Close() error {
	close(nc.done)
	op := &operation.RemoveClientListenerOp{
		Cache:      nc.remote.name,
		ListenerID: nc.listenerID,
	}
	err := nc.remote.client.pool.RemoveListener(context.Background(), op)
	nc.store.Clear()
	return err
}

func (nc *NearCache) invalidationLoop(client *Client) {
	for {
		select {
		case ev, ok := <-nc.events:
			if !ok {
				return
			}
			nc.store.Remove(string(ev.Key))
		case <-nc.done:
			return
		}
	}
}

func (nc *NearCache) scheduleBloomUpdate() {
	if nc.bloomDirty.CompareAndSwap(false, true) {
		go func() {
			nc.updateBloomFilter()
			nc.bloomDirty.Store(false)
		}()
	}
}

func (nc *NearCache) onEntryRemoved(key string) {
	for {
		removals := nc.removalCount.Load()
		if removals >= nc.updateThreshold {
			if nc.removalCount.CompareAndSwap(removals, 0) {
				go nc.updateBloomFilter()
				return
			}
		} else if nc.removalCount.CompareAndSwap(removals, removals+1) {
			return
		}
	}
}

func (nc *NearCache) updateBloomFilter() {
	bf := hash.NewBloomFilter(nc.bloomFilterBits)
	for _, key := range nc.store.Keys() {
		bf.Add([]byte(key))
	}
	bits := bf.ToBitSet()

	ctx := context.Background()
	op := &operation.UpdateBloomFilterOp{
		Cache:     nc.remote.name,
		BloomBits: bits,
	}
	_, _ = nc.remote.client.pool.ExecuteOnListener(ctx, nc.listenerID, op)
}
