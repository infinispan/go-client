# Infinispan Go Client

[![Go Reference](https://pkg.go.dev/badge/infinispan.org/go-client.svg)](https://pkg.go.dev/infinispan.org/go-client)
[![License](https://img.shields.io/github/license/infinispan/go-client)](https://github.com/infinispan/go-client/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/infinispan/go-client)](https://github.com/infinispan/go-client/releases/latest)

A Go client for [Infinispan](https://infinispan.org/) using the Hot Rod binary protocol (version 4.1).

## Features

- Hot Rod 4.1 binary protocol over TCP
- SCRAM-SHA-256, PLAIN, OAUTHBEARER and EXTERNAL (certificate) authentication
- TLS and mutual TLS (mTLS)
- Topology-aware and hash-distribution-aware routing
- Multiplexed connections (concurrent operations over a single TCP connection)
- Type-safe cache with Protocol Buffers marshalling
- Cache entry event listeners (created, modified, removed, expired)
- Queries and continuous queries with Ickle query language
- Operation flags (force return value, skip cache load, skip indexing, etc.)

## Installation

```
go get infinispan.org/go-client
```

## Quick Start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"infinispan.org/go-client/hotrod"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := hotrod.NewClient(ctx, "hotrod://admin:password@localhost:11222")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	cache := client.Cache("my-cache")

	// Put
	if err := cache.Put(ctx, []byte("greeting"), []byte("hello world")); err != nil {
		log.Fatal(err)
	}

	// Get
	val, found, err := cache.Get(ctx, []byte("greeting"))
	if err != nil {
		log.Fatal(err)
	}
	if found {
		fmt.Println(string(val)) // hello world
	}

	// Remove
	if err := cache.Remove(ctx, []byte("greeting")); err != nil {
		log.Fatal(err)
	}
}
```

## Connection URI

The client connects using a URI of the form:

```
hotrod://username:password@host:port
hotrods://username:password@host:port   (TLS)
```

Query parameters for TLS configuration:

| Parameter               | Description                           |
|-------------------------|---------------------------------------|
| `trust_store_file_name` | Path to CA certificate PEM file       |
| `sni_host_name`         | SNI hostname for TLS                  |
| `trust_ca`              | Alias for `trust_store_file_name`     |
| `client_cert`           | Path to client certificate PEM (mTLS) |
| `client_key`            | Path to client private key PEM (mTLS) |
| `sni_host`              | Alias for `sni_host_name`             |

## Options

Client-level options are passed to `NewClient`:

```go
client, err := hotrod.NewClient(ctx, uri,
	hotrod.WithClientIntelligence(hotrod.IntelligenceBasic),
	hotrod.WithAuth("SCRAM-SHA-256", "admin", "password"),
	hotrod.WithConnectTimeout(5*time.Second),
	hotrod.WithTrustStore("/path/to/ca.pem"),
)
```

Per-operation options:

```go
// Put with expiration
cache.Put(ctx, key, value,
	hotrod.WithLifespan(60*time.Second),
	hotrod.WithMaxIdle(30*time.Second),
)

// Put and get the previous value
prev, err := cache.GetAndPut(ctx, key, value)

// Remove and get the previous value
prev, existed, err := cache.GetAndRemove(ctx, key)
```

## Typed Cache with Protocol Buffers

For type-safe operations using Protocol Buffers:

```go
import (
	"infinispan.org/go-client/hotrod"
	pb "your/module/proto"
)

// Register the .proto schema with the server
client.Schemas().Register(ctx, "person.proto", personProtoContent)

// Create a typed cache
cache := hotrod.NewTypedCache[string, *pb.Person](
	client, "people",
	hotrod.ProtoStreamMarshaller(),
	func() *pb.Person { return &pb.Person{} },
)

cache.Put(ctx, "john", &pb.Person{Name: "John", Age: 30})

person, found, err := cache.Get(ctx, "john")
```

## Event Listeners

Listen for cache entry events:

```go
listener, err := cache.AddListener(ctx,
	hotrod.WithListenerInterests(hotrod.EventCreated, hotrod.EventRemoved),
)
defer cache.RemoveListener(ctx, listener)

for ev := range listener.Events {
	fmt.Printf("event=%v key=%s\n", ev.Type, ev.Key)
}
```

## Queries

Execute Ickle queries against an indexed cache:

```go
result, err := cache.Query(ctx, "FROM test.Person WHERE age > :minAge",
	hotrod.WithQueryParam("minAge", int32(25)),
)
if err != nil {
	log.Fatal(err)
}

for _, entry := range result.Entries {
	p := &pb.Person{}
	proto.Unmarshal(entry.Value, p)
	fmt.Printf("%s (age %d)\n", p.Name, p.Age)
}
```

Queries require an indexed cache with a registered protobuf schema.

## Continuous Queries

Register a continuous query using the Ickle query language:

```go
cq, err := cache.ContinuousQuery(ctx, "FROM test.Person WHERE age >= :minAge",
	hotrod.WithCQParam("minAge", int32(18)),
)
defer cache.RemoveContinuousQuery(ctx, cq)

for ev := range cq.Events {
	switch ev.Type {
	case hotrod.CQJoining:
		fmt.Printf("matched: %s\n", ev.Key)
	case hotrod.CQLeaving:
		fmt.Printf("no longer matches: %s\n", ev.Key)
	}
}
```

Continuous queries require an indexed cache with a registered protobuf schema.

## Building

```
go build ./...
```

## Testing

Unit tests (no server required):

```
go test -short ./...
```

Integration tests (requires Docker for [testcontainers](https://golang.testcontainers.org/)):

```
go test -timeout 120s ./test/...
```

## Project Structure

```
hotrod/           Source code (package hotrod)
test/             Integration and public-API tests
internal/         Internal packages (codec, connection pool, auth, hashing)
documentation/    User guide and protocol implementation guide
```

## License

Apache License 2.0
