package hotrod

import (
	"context"
	"crypto/rand"
	"fmt"

	"infinispan.org/go-client/internal/operation"
)

// CounterType distinguishes strong and weak counters.
type CounterType = operation.CounterType

// CounterStorage controls whether a counter survives server restarts.
type CounterStorage = operation.CounterStorage

// CounterState represents the state of a counter value relative to its bounds.
type CounterState = operation.CounterState

// CounterConfiguration holds the configuration of a distributed counter,
// including its type, storage mode, bounds, and initial value.
type CounterConfiguration = operation.CounterConfiguration

const (
	// CounterStrong identifies a strong counter that supports all operations
	// including compare-and-swap.
	CounterStrong CounterType = operation.CounterStrong
	// CounterWeak identifies a weak counter optimized for high write throughput.
	// Weak counters do not support compare-and-swap or get-and-set.
	CounterWeak CounterType = operation.CounterWeak

	// StorageVolatile means the counter value is lost on server restart.
	StorageVolatile CounterStorage = operation.StorageVolatile
	// StoragePersistent means the counter value survives server restarts.
	StoragePersistent CounterStorage = operation.StoragePersistent

	// CounterStateValid indicates the counter value is within its bounds.
	CounterStateValid CounterState = operation.CounterStateValid
	// CounterStateLowerBound indicates the counter has reached its lower bound.
	CounterStateLowerBound CounterState = operation.CounterStateLowerBound
	// CounterStateUpperBound indicates the counter has reached its upper bound.
	CounterStateUpperBound CounterState = operation.CounterStateUpperBound
)

// CounterEvent represents a counter value change event from the server.
type CounterEvent = operation.CounterEvent

// CounterManager provides operations for managing distributed counters.
type CounterManager struct {
	client *Client
}

// Counters returns a CounterManager for creating and managing distributed counters.
func (c *Client) Counters() *CounterManager {
	return &CounterManager{client: c}
}

// Define creates a counter with the given configuration. Returns true if created,
// false if a counter with that name already exists.
func (cm *CounterManager) Define(ctx context.Context, name string, config *CounterConfiguration) (bool, error) {
	return execute[bool](ctx, cm.client.pool, &operation.CounterCreateOp{
		Name:   name,
		Config: config,
	})
}

// Defined reports whether a counter with the given name exists.
func (cm *CounterManager) Defined(ctx context.Context, name string) (bool, error) {
	return execute[bool](ctx, cm.client.pool, &operation.CounterIsDefinedOp{
		Name: name,
	})
}

// Configuration returns the configuration of a named counter, or nil if it does not exist.
func (cm *CounterManager) Configuration(ctx context.Context, name string) (*CounterConfiguration, error) {
	return execute[*CounterConfiguration](ctx, cm.client.pool, &operation.CounterGetConfigurationOp{
		Name: name,
	})
}

// Remove resets a counter to its initial value. The counter definition remains.
func (cm *CounterManager) Remove(ctx context.Context, name string) error {
	success, err := execute[bool](ctx, cm.client.pool, &operation.CounterRemoveOp{
		Name: name,
	})
	if err != nil {
		return err
	}
	if !success {
		return fmt.Errorf("failed to reset counter %q", name)
	}
	return nil
}

// Names returns the names of all counters defined in the cluster.
func (cm *CounterManager) Names(ctx context.Context) ([]string, error) {
	return execute[[]string](ctx, cm.client.pool, &operation.CounterGetNamesOp{})
}

// Counter returns a Counter handle for operating on the named counter.
func (cm *CounterManager) Counter(name string) *Counter {
	return &Counter{client: cm.client, name: name}
}

// Counter provides operations on a single named distributed counter.
type Counter struct {
	client *Client
	name   string
}

// Name returns the counter name.
func (c *Counter) Name() string { return c.name }

// Get returns the current value of the counter.
func (c *Counter) Get(ctx context.Context) (int64, error) {
	return execute[int64](ctx, c.client.pool, &operation.CounterGetOp{
		Name: c.name,
	})
}

// GetAndSet sets the counter to the given value and returns the previous value.
func (c *Counter) GetAndSet(ctx context.Context, value int64) (int64, error) {
	return execute[int64](ctx, c.client.pool, &operation.CounterGetAndSetOp{
		Name:  c.name,
		Value: value,
	})
}

// AddAndGet adds a delta to the counter and returns the new value.
// For weak counters the returned value is always 0.
func (c *Counter) AddAndGet(ctx context.Context, delta int64) (int64, error) {
	return execute[int64](ctx, c.client.pool, &operation.CounterAddAndGetOp{
		Name:  c.name,
		Delta: delta,
	})
}

// CompareAndSwap atomically sets the counter to update if its current value equals expect.
// Returns the previous value and whether the swap succeeded.
func (c *Counter) CompareAndSwap(ctx context.Context, expect, update int64) (int64, bool, error) {
	cas, err := execute[*operation.CounterCASResult](ctx, c.client.pool, &operation.CounterCasOp{
		Name:   c.name,
		Expect: expect,
		Update: update,
	})
	if err != nil {
		return 0, false, err
	}
	return cas.OldValue, cas.Success, nil
}

// Reset resets the counter to its initial value.
func (c *Counter) Reset(ctx context.Context) error {
	_, err := c.client.pool.Execute(ctx, &operation.CounterResetOp{
		Name: c.name,
	})
	return err
}

// CounterListener receives counter change events on its Events channel.
// Create one with [Counter.AddListener] and remove it with [Counter.RemoveListener].
type CounterListener struct {
	// Events delivers counter value change notifications from the server.
	// The channel is closed if the underlying connection drops.
	Events <-chan *CounterEvent
	id     []byte
	ch     chan *CounterEvent
}

// AddListener registers a listener for value changes on this counter.
func (c *Counter) AddListener(ctx context.Context) (*CounterListener, error) {
	listenerID := make([]byte, 16)
	if _, err := rand.Read(listenerID); err != nil {
		return nil, err
	}

	ch := make(chan *CounterEvent, 64)

	op := &operation.CounterAddListenerOp{
		Name:       c.name,
		ListenerID: listenerID,
	}

	if err := c.client.pool.AddCounterListener(ctx, op, ch); err != nil {
		return nil, err
	}

	return &CounterListener{
		Events: ch,
		id:     listenerID,
		ch:     ch,
	}, nil
}

// RemoveListener unregisters a previously added counter listener.
func (c *Counter) RemoveListener(ctx context.Context, listener *CounterListener) error {
	op := &operation.CounterRemoveListenerOp{
		Name:       c.name,
		ListenerID: listener.id,
	}
	return c.client.pool.RemoveCounterListener(ctx, op)
}
