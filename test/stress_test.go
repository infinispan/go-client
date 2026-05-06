package hotrod_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
)

func TestPipelineStress(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "stress")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("stress")

	const goroutines = 20
	const opsPerGoroutine = 50

	var wg sync.WaitGroup
	errCh := make(chan error, goroutines*opsPerGoroutine)

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < opsPerGoroutine; i++ {
				key := []byte(fmt.Sprintf("g%d-k%d", id, i))
				val := []byte(fmt.Sprintf("g%d-v%d", id, i))

				if err := cache.Put(ctx, key, val); err != nil {
					errCh <- fmt.Errorf("goroutine %d put %d: %w", id, i, err)
					return
				}

				got, found, err := cache.Get(ctx, key)
				if err != nil {
					errCh <- fmt.Errorf("goroutine %d get %d: %w", id, i, err)
					return
				}
				if !found {
					errCh <- fmt.Errorf("goroutine %d get %d: key not found", id, i)
					return
				}
				if string(got) != string(val) {
					errCh <- fmt.Errorf("goroutine %d get %d: value mismatch: got %q want %q", id, i, got, val)
					return
				}
			}
		}(g)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}

	t.Logf("completed %d operations across %d goroutines with no cross-talk", goroutines*opsPerGoroutine*2, goroutines)
}

func TestContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	createTestCache(t, sharedContainer, "cancel")

	uri := fmt.Sprintf("hotrod://admin:password@%s", sharedAddr)
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer setupCancel()

	client, err := hotrod.NewClient(setupCtx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("cancel")

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()

	err = cache.Put(cancelledCtx, []byte("k"), []byte("v"))
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	if cancelledCtx.Err() == nil {
		t.Fatal("expected context to be cancelled")
	}
}
