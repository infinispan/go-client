package hotrod_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"infinispan.org/go-client/hotrod"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

func startClusterNode(t *testing.T, ctx context.Context, nw *testcontainers.DockerNetwork, alias string) testcontainers.Container {
	t.Helper()
	req := testcontainers.ContainerRequest{
		Image:        serverImage(),
		ExposedPorts: []string{"11222/tcp"},
		Env: map[string]string{
			"USER": "admin",
			"PASS": "password",
		},
		WaitingFor: wait.ForLog("ISPN080001").WithStartupTimeout(120 * time.Second),
	}
	customizer := network.WithNetwork([]string{alias}, nw)
	genReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}
	if err := customizer(&genReq); err != nil {
		t.Fatalf("apply network customizer: %v", err)
	}
	container, err := testcontainers.GenericContainer(ctx, genReq)
	if err != nil {
		t.Fatalf("start infinispan node %s: %v", alias, err)
	}
	return container
}

func nodeAddr(t *testing.T, ctx context.Context, container testcontainers.Container) string {
	t.Helper()
	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("get host: %v", err)
	}
	port, err := container.MappedPort(ctx, "11222")
	if err != nil {
		t.Fatalf("get port: %v", err)
	}
	return fmt.Sprintf("%s:%s", host, port.Port())
}

func waitForClusterSize(t *testing.T, ctx context.Context, container testcontainers.Container, expectedSize int) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		_, output, err := container.Exec(ctx, []string{
			"bash", "-c",
			"curl -s --digest -u admin:password http://localhost:11222/rest/v2/cluster?action=distribution",
		})
		if err == nil {
			buf := make([]byte, 4096)
			n, _ := output.Read(buf)
			body := string(buf[:n])
			t.Logf("cluster distribution: %s", body)

			// Count node_name occurrences to determine cluster size
			members := 0
			for i := 0; i < len(body); i++ {
				if i+9 <= len(body) && body[i:i+9] == "node_name" {
					members++
				}
			}
			if members >= expectedSize {
				t.Logf("cluster size reached %d", members)
				return
			}
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("timed out waiting for cluster size %d", expectedSize)
}

func waitForTopologyServers(t *testing.T, client *hotrod.Client, ctx context.Context, cache *hotrod.RemoteCache, expectedCount int) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	probe := 0
	for time.Now().Before(deadline) {
		// Use different keys to hit different servers / hash segments
		for j := 0; j < 5; j++ {
			key := fmt.Sprintf("topo-probe-%d", probe)
			probe++
			cache.Put(ctx, []byte(key), []byte("x"))
		}

		servers := client.TopologyServers()
		if len(servers) == expectedCount {
			t.Logf("client topology has %d servers: %v", len(servers), servers)
			return
		}
		t.Logf("client topology has %d servers (want %d): %v", len(servers), expectedCount, servers)
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("timed out waiting for client to see %d topology servers", expectedCount)
}

func TestTopologyScaleUpAndDown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()

	// Create a shared Docker network for the cluster
	nw, err := network.New(ctx)
	if err != nil {
		t.Fatalf("create network: %v", err)
	}
	t.Cleanup(func() { nw.Remove(ctx) })

	// Start the first node
	t.Log("--- Starting node 1 ---")
	node1 := startClusterNode(t, ctx, nw, "ispn-node1")
	t.Cleanup(func() { node1.Terminate(ctx) })

	addr1 := nodeAddr(t, ctx, node1)
	t.Logf("node1 address: %s", addr1)

	// Create a distributed cache on node1
	createTestCache(t, node1, "topo-test")

	// Connect the client to node1
	uri := fmt.Sprintf("hotrod://admin:password@%s", addr1)
	clientCtx, clientCancel := context.WithTimeout(ctx, 120*time.Second)
	defer clientCancel()

	client, err := hotrod.NewClient(clientCtx, uri)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.Close()

	cache := client.Cache("topo-test")

	// Put some initial data
	for i := 0; i < 20; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		val := []byte(fmt.Sprintf("val-%d", i))
		if err := cache.Put(clientCtx, key, val); err != nil {
			t.Fatalf("initial Put key-%d: %v", i, err)
		}
	}

	// Verify: client should see 1 topology server
	servers := client.TopologyServers()
	t.Logf("initial topology: %v", servers)

	// --- Scale up: add node2 ---
	t.Log("--- Starting node 2 ---")
	node2 := startClusterNode(t, ctx, nw, "ispn-node2")
	t.Cleanup(func() { node2.Terminate(ctx) })

	addr2 := nodeAddr(t, ctx, node2)
	t.Logf("node2 address: %s", addr2)

	// Wait for the server-side cluster to form
	waitForClusterSize(t, ctx, node1, 2)

	// Trigger topology update delivery to the client by performing operations
	waitForTopologyServers(t, client, clientCtx, cache, 2)

	// Verify consistent hash is updated with 2 owners
	ownerCount := client.ConsistentHashOwnerCount("topo-test")
	t.Logf("consistent hash owner count after scale-up: %d", ownerCount)
	if ownerCount < 2 {
		t.Errorf("expected at least 2 owners in consistent hash after adding node2, got %d", ownerCount)
	}

	// Verify data is still accessible after topology change
	for i := 0; i < 20; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		expected := fmt.Sprintf("val-%d", i)
		val, found, err := cache.Get(clientCtx, key)
		if err != nil {
			t.Fatalf("Get key-%d after scale-up: %v", i, err)
		}
		if !found {
			t.Errorf("key-%d not found after scale-up", i)
			continue
		}
		if string(val) != expected {
			t.Errorf("key-%d: got %q, want %q", i, string(val), expected)
		}
	}

	// Put more data with the 2-node topology
	for i := 20; i < 40; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		val := []byte(fmt.Sprintf("val-%d", i))
		if err := cache.Put(clientCtx, key, val); err != nil {
			t.Fatalf("Put key-%d with 2 nodes: %v", i, err)
		}
	}

	// --- Scale up: add node3 ---
	t.Log("--- Starting node 3 ---")
	node3 := startClusterNode(t, ctx, nw, "ispn-node3")
	t.Cleanup(func() { node3.Terminate(ctx) })

	addr3 := nodeAddr(t, ctx, node3)
	t.Logf("node3 address: %s", addr3)

	waitForClusterSize(t, ctx, node1, 3)
	waitForTopologyServers(t, client, clientCtx, cache, 3)

	ownerCount = client.ConsistentHashOwnerCount("topo-test")
	t.Logf("consistent hash owner count after 3rd node: %d", ownerCount)
	if ownerCount < 3 {
		t.Errorf("expected 3 owners in consistent hash, got %d", ownerCount)
	}

	// Verify all 40 keys
	for i := 0; i < 40; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		expected := fmt.Sprintf("val-%d", i)
		val, found, err := cache.Get(clientCtx, key)
		if err != nil {
			t.Fatalf("Get key-%d with 3 nodes: %v", i, err)
		}
		if !found {
			t.Errorf("key-%d not found with 3 nodes", i)
			continue
		}
		if string(val) != expected {
			t.Errorf("key-%d: got %q, want %q", i, string(val), expected)
		}
	}

	// --- Scale down: remove node3 ---
	t.Log("--- Stopping node 3 ---")
	if err := node3.Terminate(ctx); err != nil {
		t.Fatalf("terminate node3: %v", err)
	}

	waitForClusterSize(t, ctx, node1, 2)
	waitForTopologyServers(t, client, clientCtx, cache, 2)

	ownerCount = client.ConsistentHashOwnerCount("topo-test")
	t.Logf("consistent hash owner count after removing node3: %d", ownerCount)
	if ownerCount != 2 {
		t.Errorf("expected 2 owners after removing node3, got %d", ownerCount)
	}

	// Verify all data is still accessible (DIST_SYNC with numOwners=2 means
	// all data should survive losing 1 of 3 nodes)
	for i := 0; i < 40; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		expected := fmt.Sprintf("val-%d", i)
		val, found, err := cache.Get(clientCtx, key)
		if err != nil {
			t.Fatalf("Get key-%d after scale-down: %v", i, err)
		}
		if !found {
			t.Errorf("key-%d not found after scale-down", i)
			continue
		}
		if string(val) != expected {
			t.Errorf("key-%d: got %q, want %q", i, string(val), expected)
		}
	}

	// --- Scale down: remove node2 ---
	t.Log("--- Stopping node 2 ---")
	if err := node2.Terminate(ctx); err != nil {
		t.Fatalf("terminate node2: %v", err)
	}

	waitForClusterSize(t, ctx, node1, 1)
	waitForTopologyServers(t, client, clientCtx, cache, 1)

	ownerCount = client.ConsistentHashOwnerCount("topo-test")
	t.Logf("consistent hash owner count after removing node2: %d", ownerCount)
	if ownerCount != 1 {
		t.Errorf("expected 1 owner after removing node2, got %d", ownerCount)
	}

	// All data should still be accessible (the remaining node has copies)
	for i := 0; i < 40; i++ {
		key := []byte(fmt.Sprintf("key-%d", i))
		expected := fmt.Sprintf("val-%d", i)
		val, found, err := cache.Get(clientCtx, key)
		if err != nil {
			t.Fatalf("Get key-%d single node: %v", i, err)
		}
		if !found {
			t.Errorf("key-%d not found on single node", i)
			continue
		}
		if string(val) != expected {
			t.Errorf("key-%d: got %q, want %q", i, string(val), expected)
		}
	}

	t.Log("--- Topology change test passed ---")
}
