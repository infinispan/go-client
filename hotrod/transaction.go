package hotrod

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync/atomic"
	"time"

	"infinispan.org/go-client/internal/codec"
	"infinispan.org/go-client/internal/operation"
)

const xidFormatID int32 = 0x48525458 // "HRTX"

var txCounter atomic.Int64

// TxCache provides transactional Get, Put, and Remove operations within a
// transaction scope. Changes are tracked locally and sent to the server as a
// single prepare request when the transaction commits.
type TxCache struct {
	client    *Client
	cacheName string
	entries   map[string]*txEntry
}

type txEntry struct {
	key         []byte
	value       []byte
	versionRead int64
	control     byte
	lifespan    time.Duration
	maxIdle     time.Duration
	modified    bool
}

// Get retrieves the value for a key within the transaction. The first read for
// a key fetches from the server and records the version for optimistic locking.
// Subsequent reads return the locally cached value.
func (tc *TxCache) Get(ctx context.Context, key []byte) ([]byte, bool, error) {
	hex := hexKey(key)
	if e, ok := tc.entries[hex]; ok {
		if e.control&operation.CtrlRemoveOp != 0 {
			return nil, false, nil
		}
		return e.value, e.value != nil, nil
	}
	result, err := tc.client.pool.Execute(ctx, &operation.GetWithVersionOp{
		Cache: tc.cacheName,
		Key:   key,
	})
	if err != nil {
		return nil, false, err
	}
	resp := result.(*operation.GetWithVersionResponse)
	e := &txEntry{key: dup(key)}
	if resp.Found {
		e.value = resp.Value
		e.versionRead = resp.Version
	} else {
		e.control = operation.CtrlNonExisting
	}
	tc.entries[hex] = e
	return e.value, resp.Found, nil
}

// Put stores a key-value pair within the transaction. If the key was not
// previously read, the modification is marked as NOT_READ. Optional PutOption
// values control lifespan and max-idle expiration.
func (tc *TxCache) Put(ctx context.Context, key, value []byte, opts ...PutOption) error {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	hex := hexKey(key)
	e, ok := tc.entries[hex]
	if !ok {
		e = &txEntry{
			key:     dup(key),
			control: operation.CtrlNotRead,
		}
		tc.entries[hex] = e
	}
	e.value = dup(value)
	e.control &^= operation.CtrlRemoveOp
	e.lifespan = cfg.lifespan
	e.maxIdle = cfg.maxIdle
	e.modified = true
	return nil
}

// Remove marks a key for deletion within the transaction. If the key was not
// previously read, the modification is marked as NOT_READ.
func (tc *TxCache) Remove(ctx context.Context, key []byte) error {
	hex := hexKey(key)
	e, ok := tc.entries[hex]
	if !ok {
		e = &txEntry{
			key:     dup(key),
			control: operation.CtrlNotRead,
		}
		tc.entries[hex] = e
	}
	e.control |= operation.CtrlRemoveOp
	e.value = nil
	e.modified = true
	return nil
}

// WithTransaction executes fn within a transaction on the named cache. If fn
// returns nil, the transaction commits. If fn returns an error (or panics),
// the transaction rolls back. Uses one-phase commit for single-cache sync mode.
func (c *Client) WithTransaction(ctx context.Context, cacheName string, fn func(tc *TxCache) error, opts ...TxOption) error {
	cfg := &txConfig{timeout: 60 * time.Second}
	for _, o := range opts {
		o(cfg)
	}

	globalTxID, branchQual := generateXID()

	tc := &TxCache{
		client:    c,
		cacheName: cacheName,
		entries:   make(map[string]*txEntry),
	}

	var fnErr error
	func() {
		defer func() {
			if r := recover(); r != nil {
				fnErr = fmt.Errorf("transaction panic: %v", r)
			}
		}()
		fnErr = fn(tc)
	}()

	if fnErr != nil {
		_ = rollback(ctx, c, globalTxID, branchQual)
		return fnErr
	}

	mods := buildModifications(tc.entries)
	if len(mods) == 0 {
		return nil
	}

	result, err := c.pool.Execute(ctx, &operation.PrepareTx2Op{
		Cache:          cacheName,
		FormatID:       xidFormatID,
		GlobalTxID:     globalTxID,
		BranchQual:     branchQual,
		OnePhaseCommit: true,
		TimeoutMs:      cfg.timeout.Milliseconds(),
		Modifications:  mods,
	})
	if err != nil {
		_ = rollback(ctx, c, globalTxID, branchQual)
		return fmt.Errorf("prepare transaction: %w", err)
	}

	xaCode := result.(int32)
	if xaCode != codec.XaOk && xaCode != codec.XaRdOnly {
		_ = rollback(ctx, c, globalTxID, branchQual)
		return fmt.Errorf("transaction rejected: XA code %d", xaCode)
	}
	return nil
}

func rollback(ctx context.Context, c *Client, globalTxID, branchQual []byte) error {
	_, err := c.pool.Execute(ctx, &operation.RollbackTxOp{
		FormatID:   xidFormatID,
		GlobalTxID: globalTxID,
		BranchQual: branchQual,
	})
	return err
}

func buildModifications(entries map[string]*txEntry) []operation.TxModification {
	var mods []operation.TxModification
	for _, e := range entries {
		if !e.modified {
			continue
		}
		mods = append(mods, operation.TxModification{
			Key:         e.key,
			Value:       e.value,
			VersionRead: e.versionRead,
			Control:     e.control,
			Lifespan:    e.lifespan,
			MaxIdle:     e.maxIdle,
		})
	}
	return mods
}

func generateXID() (globalTxID, branchQual []byte) {
	globalTxID = make([]byte, 32)
	branchQual = make([]byte, 32)

	_, _ = rand.Read(globalTxID[:16])
	binary.BigEndian.PutUint64(globalTxID[16:24], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint64(globalTxID[24:32], uint64(txCounter.Add(1)))

	_, _ = rand.Read(branchQual[:16])
	binary.BigEndian.PutUint64(branchQual[16:24], uint64(time.Now().UnixNano()))
	binary.BigEndian.PutUint64(branchQual[24:32], uint64(txCounter.Add(1)))

	return globalTxID, branchQual
}

func hexKey(b []byte) string {
	const hextable = "0123456789abcdef"
	dst := make([]byte, len(b)*2)
	for i, v := range b {
		dst[i*2] = hextable[v>>4]
		dst[i*2+1] = hextable[v&0x0f]
	}
	return string(dst)
}

func dup(b []byte) []byte {
	if b == nil {
		return nil
	}
	c := make([]byte, len(b))
	copy(c, b)
	return c
}
