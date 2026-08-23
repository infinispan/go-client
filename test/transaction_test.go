package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

const txCacheConfig = `<distributed-cache>
  <transaction mode="NON_XA" locking="OPTIMISTIC"/>
</distributed-cache>`

func createTxCache(t *testing.T, client *hotrod.Client, name string) {
	t.Helper()
	ctx := context.Background()
	if err := client.Admin().GetOrCreateCache(ctx, name, txCacheConfig); err != nil {
		t.Fatalf("create tx cache %q: %v", name, err)
	}
}

func TestTransaction_PutAndGet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	createTxCache(t, client, "tx-put")

	err = client.WithTransaction(ctx, "tx-put", func(tc *hotrod.TxCache) error {
		if err := tc.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}

	val, found, err := client.Cache("tx-put").Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after commit: %v", err)
	}
	if !found {
		t.Fatal("expected key to be found after commit")
	}
	if string(val) != "v1" {
		t.Fatalf("got %q, want %q", string(val), "v1")
	}
}

func TestTransaction_Rollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	createTxCache(t, client, "tx-rb")

	cache := client.Cache("tx-rb")
	if err := cache.Put(ctx, []byte("k1"), []byte("original")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	txErr := client.WithTransaction(ctx, "tx-rb", func(tc *hotrod.TxCache) error {
		if err := tc.Put(ctx, []byte("k1"), []byte("changed")); err != nil {
			return err
		}
		return fmt.Errorf("intentional rollback")
	})
	if txErr == nil {
		t.Fatal("expected error from rolled-back transaction")
	}

	val, found, err := cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after rollback: %v", err)
	}
	if !found {
		t.Fatal("expected key to exist after rollback")
	}
	if string(val) != "original" {
		t.Fatalf("got %q, want %q after rollback", string(val), "original")
	}
}

func TestTransaction_OptimisticLocking(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	createTxCache(t, client, "tx-opt")

	cache := client.Cache("tx-opt")
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	txErr := client.WithTransaction(ctx, "tx-opt", func(tc *hotrod.TxCache) error {
		_, _, err := tc.Get(ctx, []byte("k1"))
		if err != nil {
			return err
		}
		// Modify externally while transaction is open.
		if err := cache.Put(ctx, []byte("k1"), []byte("external")); err != nil {
			return fmt.Errorf("external put: %w", err)
		}
		return tc.Put(ctx, []byte("k1"), []byte("tx-value"))
	})
	if txErr == nil {
		t.Fatal("expected transaction to fail due to version conflict")
	}
	t.Logf("transaction correctly rejected: %v", txErr)

	val, _, err := cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(val) != "external" {
		t.Fatalf("got %q, want %q", string(val), "external")
	}
}

func TestTransaction_ReadOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	createTxCache(t, client, "tx-ro")

	cache := client.Cache("tx-ro")
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	err = client.WithTransaction(ctx, "tx-ro", func(tc *hotrod.TxCache) error {
		val, found, err := tc.Get(ctx, []byte("k1"))
		if err != nil {
			return err
		}
		if !found || string(val) != "v1" {
			return fmt.Errorf("unexpected: found=%v val=%q", found, string(val))
		}
		return nil
	})
	if err != nil {
		t.Fatalf("read-only transaction should succeed: %v", err)
	}
}

func TestTransaction_PutWithoutRead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	createTxCache(t, client, "tx-blind")

	err = client.WithTransaction(ctx, "tx-blind", func(tc *hotrod.TxCache) error {
		if err := tc.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
			return err
		}
		if err := tc.Put(ctx, []byte("k2"), []byte("v2")); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}

	cache := client.Cache("tx-blind")
	val, found, err := cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get k1: %v", err)
	}
	if !found || string(val) != "v1" {
		t.Fatalf("k1: found=%v val=%q", found, string(val))
	}

	val, found, err = cache.Get(ctx, []byte("k2"))
	if err != nil {
		t.Fatalf("Get k2: %v", err)
	}
	if !found || string(val) != "v2" {
		t.Fatalf("k2: found=%v val=%q", found, string(val))
	}
}

func TestTransaction_Remove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	createTxCache(t, client, "tx-rm")

	cache := client.Cache("tx-rm")
	if err := cache.Put(ctx, []byte("k1"), []byte("v1")); err != nil {
		t.Fatalf("Put: %v", err)
	}

	err = client.WithTransaction(ctx, "tx-rm", func(tc *hotrod.TxCache) error {
		_, _, err := tc.Get(ctx, []byte("k1"))
		if err != nil {
			return err
		}
		return tc.Remove(ctx, []byte("k1"))
	})
	if err != nil {
		t.Fatalf("WithTransaction: %v", err)
	}

	_, found, err := cache.Get(ctx, []byte("k1"))
	if err != nil {
		t.Fatalf("Get after tx remove: %v", err)
	}
	if found {
		t.Fatal("expected key to be removed after commit")
	}
}
