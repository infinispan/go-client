# Testing Instructions

## Test Organization

Tests are organized into two categories:

| Category         | Location                     | Guard                     | Command                              |
|------------------|------------------------------|---------------------------|--------------------------------------|
| Unit tests       | `*_test.go` in each package  | `testing.Short()` (none)  | `go test -short ./...`               |
| Integration tests| `test/*_test.go`             | `testing.Short()` skip    | `go test -timeout 120s ./test/...`   |

## Writing Unit Tests

Unit tests live alongside the code they test, in `_test.go` files within the same package.

### Conventions
- Test function names: `TestTypeName_MethodOrBehavior` (for example, `TestBloomFilter_AddAndBits`).
- Use `t.Fatal` / `t.Fatalf` for setup failures that make the rest of the test pointless.
- Use `t.Error` / `t.Errorf` for assertion failures where the test can continue checking other conditions.
- Use table-driven tests when testing the same logic with multiple inputs.
- Do not use third-party assertion libraries — use standard `testing` package functions.

### Example

```go
func TestLRU_EvictsOldest(t *testing.T) {
    lru := newLRU(2, nil)
    lru.Put("a", []byte("1"))
    lru.Put("b", []byte("2"))
    lru.Put("c", []byte("3"))

    if _, ok := lru.Get("a"); ok {
        t.Error("expected 'a' to be evicted")
    }
}
```

## Writing Integration Tests

Integration tests live in the `test/` directory and use the `hotrod_test` package. They require Docker for testcontainers.

### Conventions
- Always guard with `testing.Short()`:
  ```go
  if testing.Short() {
      t.Skip("skipping integration test")
  }
  ```
- Use `startInfinispanPublic(t)` to start a server container. It returns the `host:port` address and the container handle.
- Use `createTestCache(t, container, "cache-name")` for standard `DIST_SYNC` caches.
- Use `createIndexedCache(t, container, "cache-name")` for indexed caches (required for queries and continuous queries).
- Use `registerSchemaREST(t, container, "schema.proto", schemaContent)` to register protobuf schemas.
- Each test should use a unique cache name to avoid interference.
- Use `context.WithTimeout` with a generous timeout (30-60 seconds) since container operations can be slow.
- Always `defer client.Close()` and `defer cancel()`.
- For tests that wait for asynchronous events (listener invalidation, CQ events), use a polling loop with a deadline rather than a fixed `time.Sleep`.

### Example

```go
func TestMyFeature(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    addr, container := startInfinispanPublic(t)
    createTestCache(t, container, "my-feature")

    uri := fmt.Sprintf("hotrod://admin:password@%s", addr)
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    client, err := hotrod.NewClient(ctx, uri, hotrod.WithClientIntelligence(hotrod.IntelligenceBasic))
    if err != nil {
        t.Fatalf("NewClient: %v", err)
    }
    defer client.Close()

    cache := client.Cache("my-feature")
    // ... test logic
}
```

### Test Helpers

All test helpers are defined in `test/hotrod_test.go`:

- **`startInfinispanPublic(t)`** — starts an Infinispan container with admin/password credentials, returns `host:port` and container.
- **`createTestCache(t, container, name)`** — creates a `DIST_SYNC` cache via the CLI.
- **`createIndexedCache(t, container, name)`** — creates an indexed cache with ProtoStream encoding via the REST API.
- **`registerSchemaREST(t, container, name, schema)`** — registers a `.proto` schema via the REST API.

## Running Tests

```bash
# All unit tests (fast, no Docker)
go test -short ./...

# All integration tests (requires Docker)
go test -timeout 120s ./test/...

# A specific test with verbose output
go test -run TestQuery_BasicIckle -v -timeout 120s ./test/...

# Tests in a specific package
go test -v ./internal/protostream/

# Race detector
go test -race -short ./...
```
