package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestStrongCounter_Add(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "add-counter")

	// Define counter with initial value 0
	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StorageVolatile,
		InitialValue: 0,
	}

	created, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	if !created {
		t.Fatal("expected counter to be created")
	}

	counter := client.Counters().Counter(counterName)

	// Get initial value
	val, err := counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 0 {
		t.Errorf("expected 0, got %d", val)
	}

	// AddAndGet positive value
	val, err = counter.AddAndGet(ctx, 5)
	if err != nil {
		t.Fatalf("AddAndGet(5): %v", err)
	}
	if val != 5 {
		t.Errorf("expected 5, got %d", val)
	}

	// AddAndGet negative value
	val, err = counter.AddAndGet(ctx, -2)
	if err != nil {
		t.Fatalf("AddAndGet(-2): %v", err)
	}
	if val != 3 {
		t.Errorf("expected 3, got %d", val)
	}

	// AddAndGet large value
	val, err = counter.AddAndGet(ctx, 100)
	if err != nil {
		t.Fatalf("AddAndGet(100): %v", err)
	}
	if val != 103 {
		t.Errorf("expected 103, got %d", val)
	}
}

func TestStrongCounter_CompareAndSwap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "cas-counter")

	// Define counter
	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StorageVolatile,
		InitialValue: 0,
	}

	_, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}

	counter := client.Counters().Counter(counterName)

	// Reset to ensure starting at 0
	if err := counter.Reset(ctx); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	// CAS should succeed: 0 -> 10
	oldVal, success, err := counter.CompareAndSwap(ctx, 0, 10)
	if err != nil {
		t.Fatalf("CompareAndSwap(0, 10): %v", err)
	}
	if !success {
		t.Fatal("expected CAS to succeed")
	}
	if oldVal != 0 {
		t.Errorf("expected old value 0, got %d", oldVal)
	}

	// Verify new value
	val, err := counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 10 {
		t.Errorf("expected 10, got %d", val)
	}

	// CAS should fail: expecting 0 but value is 10
	oldVal, success, err = counter.CompareAndSwap(ctx, 0, 20)
	if err != nil {
		t.Fatalf("CompareAndSwap(0, 20): %v", err)
	}
	if success {
		t.Fatal("expected CAS to fail")
	}
	if oldVal != 10 {
		t.Errorf("expected old value 10, got %d", oldVal)
	}

	// Value should still be 10
	val, err = counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 10 {
		t.Errorf("expected 10, got %d", val)
	}

	// CAS should succeed: 10 -> 100
	oldVal, success, err = counter.CompareAndSwap(ctx, 10, 100)
	if err != nil {
		t.Fatalf("CompareAndSwap(10, 100): %v", err)
	}
	if !success {
		t.Fatal("expected CAS to succeed")
	}
	if oldVal != 10 {
		t.Errorf("expected old value 10, got %d", oldVal)
	}
}

func TestStrongCounter_Boundaries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "bounded-counter")

	// Define bounded counter [0, 10]
	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StorageVolatile,
		InitialValue: 0,
		Bounded:      true,
		LowerBound:   0,
		UpperBound:   10,
	}

	_, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}

	counter := client.Counters().Counter(counterName)

	// Check initial value
	val, err := counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 0 {
		t.Errorf("expected 0, got %d", val)
	}

	// Try to add 10 to reach upper bound exactly
	val, err = counter.AddAndGet(ctx, 10)
	if err != nil {
		t.Fatalf("AddAndGet(10): %v", err)
	}
	if val != 10 {
		t.Errorf("expected 10 (upper bound), got %d", val)
	}

	// Try to add 1 more, should fail (exceeds upper bound)
	_, err = counter.AddAndGet(ctx, 1)
	if err == nil {
		t.Error("expected error when exceeding upper bound")
	}

	// Reset to test lower bound
	if err := counter.Reset(ctx); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	// Try to add -5, should fail (exceeds lower bound)
	_, err = counter.AddAndGet(ctx, -5)
	if err == nil {
		t.Error("expected error when exceeding lower bound")
	}

	// Add 5 to get to 5 (within bounds)
	val, err = counter.AddAndGet(ctx, 5)
	if err != nil {
		t.Fatalf("AddAndGet(5): %v", err)
	}
	if val != 5 {
		t.Errorf("expected 5, got %d", val)
	}

	// Add 3 to get to 8 (within bounds)
	val, err = counter.AddAndGet(ctx, 3)
	if err != nil {
		t.Fatalf("AddAndGet(3): %v", err)
	}
	if val != 8 {
		t.Errorf("expected 8, got %d", val)
	}
}

func TestStrongCounter_GetAndSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "getset-counter")

	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StorageVolatile,
		InitialValue: 5,
	}

	_, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}

	counter := client.Counters().Counter(counterName)

	// GetAndSet should return old value and set new value
	oldVal, err := counter.GetAndSet(ctx, 100)
	if err != nil {
		t.Fatalf("GetAndSet: %v", err)
	}
	if oldVal != 5 {
		t.Errorf("expected old value 5, got %d", oldVal)
	}

	// Verify new value
	val, err := counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}
}

func TestStrongCounter_Reset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "reset-counter")

	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StorageVolatile,
		InitialValue: 42,
	}

	_, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}

	counter := client.Counters().Counter(counterName)

	// Modify the counter
	_, err = counter.AddAndGet(ctx, 10)
	if err != nil {
		t.Fatalf("AddAndGet: %v", err)
	}

	// Verify it changed
	val, err := counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if val != 52 {
		t.Errorf("expected 52, got %d", val)
	}

	// Reset should restore initial value
	if err := counter.Reset(ctx); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	val, err = counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get after reset: %v", err)
	}
	if val != 42 {
		t.Errorf("expected 42 (initial value), got %d", val)
	}
}

func TestCounter_NameAndConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "config-counter")

	// Define counter with specific configuration
	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StoragePersistent,
		InitialValue: 10,
		Bounded:      true,
		LowerBound:   -100,
		UpperBound:   200,
	}

	created, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	if !created {
		t.Fatal("expected counter to be created")
	}

	// Check if counter is defined
	isDefined, err := client.Counters().IsDefined(ctx, counterName)
	if err != nil {
		t.Fatalf("IsDefined: %v", err)
	}
	if !isDefined {
		t.Error("expected counter to be defined")
	}

	// Get configuration
	retrievedCfg, err := client.Counters().GetConfiguration(ctx, counterName)
	if err != nil {
		t.Fatalf("GetConfiguration: %v", err)
	}
	if retrievedCfg == nil {
		t.Fatal("expected configuration to be returned")
	}

	// Verify configuration
	if retrievedCfg.Type != hotrod.CounterStrong {
		t.Errorf("expected type Strong, got %v", retrievedCfg.Type)
	}
	if retrievedCfg.Storage != hotrod.StoragePersistent {
		t.Errorf("expected storage Persistent, got %v", retrievedCfg.Storage)
	}
	if retrievedCfg.InitialValue != 10 {
		t.Errorf("expected initial value 10, got %d", retrievedCfg.InitialValue)
	}
	if !retrievedCfg.Bounded {
		t.Error("expected counter to be bounded")
	}
	if retrievedCfg.LowerBound != -100 {
		t.Errorf("expected lower bound -100, got %d", retrievedCfg.LowerBound)
	}
	if retrievedCfg.UpperBound != 200 {
		t.Errorf("expected upper bound 200, got %d", retrievedCfg.UpperBound)
	}

	// Verify counter name
	counter := client.Counters().Counter(counterName)
	if counter.Name() != counterName {
		t.Errorf("expected name %q, got %q", counterName, counter.Name())
	}
}

func TestCounter_Remove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "remove-counter")

	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StorageVolatile,
		InitialValue: 42,
	}

	_, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}

	counter := client.Counters().Counter(counterName)

	// Modify the counter
	val, err := counter.AddAndGet(ctx, 10)
	if err != nil {
		t.Fatalf("AddAndGet: %v", err)
	}
	if val != 52 {
		t.Errorf("expected 52, got %d", val)
	}

	// Remove counter (actually resets it to initial value)
	if err := client.Counters().Remove(ctx, counterName); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Verify counter still exists
	isDefined, err := client.Counters().IsDefined(ctx, counterName)
	if err != nil {
		t.Fatalf("IsDefined after remove: %v", err)
	}
	if !isDefined {
		t.Error("expected counter to still be defined after removal")
	}

	// Verify counter was reset to initial value
	val, err = counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get after remove: %v", err)
	}
	if val != 42 {
		t.Errorf("expected counter to be reset to initial value 42, got %d", val)
	}
}

func TestCounter_Names(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()

	// Create counters with unique names
	prefix := fmt.Sprintf("names-test-%d", time.Now().UnixNano())
	counter1 := fmt.Sprintf("%s-c1", prefix)
	counter2 := fmt.Sprintf("%s-c2", prefix)

	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StorageVolatile,
		InitialValue: 0,
	}

	_, err := client.Counters().Define(ctx, counter1, cfg)
	if err != nil {
		t.Fatalf("Define counter1: %v", err)
	}

	_, err = client.Counters().Define(ctx, counter2, cfg)
	if err != nil {
		t.Fatalf("Define counter2: %v", err)
	}

	// Get all counter names
	names, err := client.Counters().Names(ctx)
	if err != nil {
		t.Fatalf("Names: %v", err)
	}

	// Verify our counters are in the list
	found1, found2 := false, false
	for _, name := range names {
		if name == counter1 {
			found1 = true
		}
		if name == counter2 {
			found2 = true
		}
	}

	if !found1 {
		t.Errorf("expected to find counter %q in names list", counter1)
	}
	if !found2 {
		t.Errorf("expected to find counter %q in names list", counter2)
	}
}

func TestWeakCounter_Basic(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "weak-counter")

	// Define weak counter
	cfg := &hotrod.CounterConfiguration{
		Type:             hotrod.CounterWeak,
		Storage:          hotrod.StorageVolatile,
		InitialValue:     0,
		ConcurrencyLevel: 16, // Required for weak counters (must be > 0)
	}

	created, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}
	if !created {
		t.Fatal("expected counter to be created")
	}

	counter := client.Counters().Counter(counterName)

	// Weak counters support AddAndGet but return value may not be accurate
	// Just verify the operation doesn't error
	_, err = counter.AddAndGet(ctx, 10)
	if err != nil {
		t.Fatalf("AddAndGet: %v", err)
	}

	// Get should work
	_, err = counter.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	// Reset should work
	if err := counter.Reset(ctx); err != nil {
		t.Fatalf("Reset: %v", err)
	}
}

func TestCounter_Listener(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	client := setupClient(t)
	ctx := context.Background()
	counterName := uniqueName(t, "listener-counter")

	cfg := &hotrod.CounterConfiguration{
		Type:         hotrod.CounterStrong,
		Storage:      hotrod.StorageVolatile,
		InitialValue: 0,
	}

	_, err := client.Counters().Define(ctx, counterName, cfg)
	if err != nil {
		t.Fatalf("Define: %v", err)
	}

	counter := client.Counters().Counter(counterName)

	// Add listener
	listener, err := counter.AddListener(ctx)
	if err != nil {
		t.Fatalf("AddListener: %v", err)
	}
	defer counter.RemoveListener(ctx, listener)

	// Modify counter and expect events
	_, err = counter.AddAndGet(ctx, 5)
	if err != nil {
		t.Fatalf("AddAndGet: %v", err)
	}

	// Wait for event (with timeout)
	select {
	case event := <-listener.Events:
		if event == nil {
			t.Fatal("received nil event")
		}
		// Event received successfully
		t.Logf("Received counter event: old=%d, new=%d, oldState=%v, newState=%v",
			event.OldValue, event.NewValue, event.OldState, event.NewState)

	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for counter event")
	}
}

// Helper functions

func setupClient(t *testing.T) *hotrod.Client {
	t.Helper()

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
	})

	return client
}

func uniqueName(t *testing.T, prefix string) string {
	return fmt.Sprintf("%s-%s-%d", prefix, t.Name(), time.Now().UnixNano())
}
