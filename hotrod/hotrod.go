package hotrod

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"infinispan.org/go-client/internal/auth"
	"infinispan.org/go-client/internal/connection"
	"infinispan.org/go-client/internal/operation"
	"infinispan.org/go-client/internal/protostream"
)

// Client is a Hot Rod client connected to an Infinispan cluster.
type Client struct {
	pool *connection.Pool
}

// NewClient creates a new Hot Rod client from a URI (e.g. "hotrod://user:pass@host:port") and optional configuration.
func NewClient(ctx context.Context, uri string, opts ...Option) (*Client, error) {
	parsed, err := parseURI(uri)
	if err != nil {
		return nil, fmt.Errorf("parse URI: %w", err)
	}

	cfg := &clientConfig{}
	for _, o := range opts {
		o(cfg)
	}

	if cfg.logger == nil {
		cfg.logger = slog.Default()
	}

	tlsConfig, err := buildTLSConfig(parsed, cfg)
	if err != nil {
		return nil, err
	}

	var authMech auth.Mechanism
	mechName := cfg.authMechanism
	username := cfg.authUsername
	password := cfg.authPassword
	if parsed.username != "" && username == "" {
		username = parsed.username
		password = parsed.password
	}
	token := cfg.authToken
	if parsed.token != "" && token == "" {
		token = parsed.token
	}
	if mechName == "" {
		if parsed.saslMechanism != "" {
			mechName = parsed.saslMechanism
		}
	}
	if mechName == "" {
		if username != "" {
			mechName = "SCRAM-SHA-256"
		} else if token != "" {
			mechName = "OAUTHBEARER"
		} else if cfg.clientCertPath != "" || parsed.clientCertPath != "" {
			mechName = "EXTERNAL"
		}
	}
	switch mechName {
	case "SCRAM-SHA-256":
		authMech = auth.NewScramSHA256(username, password)
	case "PLAIN":
		authMech = auth.NewPlain(username, password)
	case "EXTERNAL":
		authMech = auth.NewExternal()
	case "OAUTHBEARER":
		authMech = auth.NewOAuthBearer(token)
	case "":
		// no auth
	default:
		return nil, fmt.Errorf("unsupported auth mechanism: %s", mechName)
	}

	addrs := make([]string, len(parsed.servers))
	for i, s := range parsed.servers {
		addrs[i] = s.String()
	}

	connectTimeout := parsed.connectTimeout
	if cfg.connectTimeout > 0 {
		connectTimeout = cfg.connectTimeout
	}
	socketTimeout := parsed.socketTimeout
	if cfg.socketTimeout > 0 {
		socketTimeout = cfg.socketTimeout
	}
	tcpNoDelay := parsed.tcpNoDelay
	if cfg.tcpNoDelay != nil {
		tcpNoDelay = cfg.tcpNoDelay
	}
	tcpKeepAlive := parsed.tcpKeepAlive
	if cfg.tcpKeepAlive != nil {
		tcpKeepAlive = cfg.tcpKeepAlive
	}

	pool := connection.NewPool(connection.PoolConfig{
		InitialServers:     addrs,
		TLSConfig:          tlsConfig,
		AuthMechanism:      authMech,
		ClientIntelligence: cfg.clientIntelligence,
		ProtocolVersion:    cfg.protocolVersion,
		ConnectTimeout:     connectTimeout,
		SocketTimeout:      socketTimeout,
		TCPNoDelay:         tcpNoDelay,
		TCPKeepAlive:       tcpKeepAlive,
		Logger:             cfg.logger,
	})

	if err := pool.Connect(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return &Client{pool: pool}, nil
}

// Cache returns a RemoteCache handle for the named cache.
func (c *Client) Cache(name string) *RemoteCache {
	return &RemoteCache{client: c, name: name}
}

// Close shuts down the client and releases all connections.
func (c *Client) Close() error {
	return c.pool.Close()
}

// TopologyServers returns the server addresses known from the current cluster topology.
func (c *Client) TopologyServers() []string {
	return c.pool.TopologyServers()
}

// HasConsistentHash reports whether the client has received a consistent hash topology for the named cache.
func (c *Client) HasConsistentHash(cacheName string) bool {
	return c.pool.ConsistentHash(cacheName) != nil
}

// ConsistentHashOwnerCount returns the number of distinct owners in the consistent hash for the named cache.
func (c *Client) ConsistentHashOwnerCount(cacheName string) int {
	ch := c.pool.ConsistentHash(cacheName)
	if ch == nil {
		return 0
	}
	owners := ch.Owners()
	uniqueOwners := make(map[string]bool)
	for _, segOwners := range owners {
		for _, addr := range segOwners {
			if addr != "" {
				uniqueOwners[addr] = true
			}
		}
	}
	return len(uniqueOwners)
}

// RemoteCache provides operations on a single Infinispan cache.
type RemoteCache struct {
	client *Client
	name   string
}

// Put stores a key-value pair in the cache, overwriting any existing entry.
func (rc *RemoteCache) Put(ctx context.Context, key, value []byte, opts ...PutOption) error {
	return rc.PutRaw(ctx, key, value, 0, opts...)
}

// Get retrieves the value for a key. Returns (nil, false, nil) if the key does not exist.
func (rc *RemoteCache) Get(ctx context.Context, key []byte) ([]byte, bool, error) {
	return rc.GetRaw(ctx, key, 0)
}

// MetadataValue holds a cache entry's value along with its server-side metadata.
type MetadataValue struct {
	Value    []byte
	Created  int64
	Lifespan int32
	LastUsed int64
	MaxIdle  int32
	Version  int64
}

// GetWithMetadata retrieves the value and metadata (version, lifespan, timestamps) for a key.
func (rc *RemoteCache) GetWithMetadata(ctx context.Context, key []byte) (*MetadataValue, bool, error) {
	result, err := rc.client.pool.Execute(ctx, &operation.GetWithMetadataOp{
		Cache: rc.name,
		Key:   key,
	})
	if err != nil {
		return nil, false, err
	}
	resp := result.(*operation.GetWithMetadataResponse)
	if !resp.Found {
		return nil, false, nil
	}
	return &MetadataValue{
		Value:    resp.Value,
		Created:  resp.Metadata.Created,
		Lifespan: resp.Metadata.Lifespan,
		LastUsed: resp.Metadata.LastUsed,
		MaxIdle:  resp.Metadata.MaxIdle,
		Version:  resp.Metadata.Version,
	}, true, nil
}

// VersionedValue holds a cache entry's value and its version number.
type VersionedValue struct {
	Value   []byte
	Version int64
}

// Deprecated: Use GetWithMetadata instead, which returns the same version along with full metadata.
func (rc *RemoteCache) GetWithVersion(ctx context.Context, key []byte) (*VersionedValue, bool, error) {
	result, err := rc.client.pool.Execute(ctx, &operation.GetWithVersionOp{
		Cache: rc.name,
		Key:   key,
	})
	if err != nil {
		return nil, false, err
	}
	resp := result.(*operation.GetWithVersionResponse)
	if !resp.Found {
		return nil, false, nil
	}
	return &VersionedValue{
		Value:   resp.Value,
		Version: resp.Version,
	}, true, nil
}

// ReplaceIfUnmodified atomically replaces the value for a key only if its version matches.
// Returns true if the replacement succeeded, false if the version has changed.
func (rc *RemoteCache) ReplaceIfUnmodified(ctx context.Context, key, value []byte, version int64, opts ...PutOption) (bool, error) {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	result, err := rc.client.pool.Execute(ctx, &operation.ReplaceIfUnmodifiedOp{
		Cache:    rc.name,
		Key:      key,
		Value:    value,
		Version:  version,
		Lifespan: cfg.lifespan,
		MaxIdle:  cfg.maxIdle,
		OpFlags:  cfg.flags,
	})
	if err != nil {
		return false, err
	}
	return result.(*operation.CASResponse).Success, nil
}

// RemoveIfUnmodified atomically removes a key only if its version matches.
// Returns true if the removal succeeded, false if the version has changed.
func (rc *RemoteCache) RemoveIfUnmodified(ctx context.Context, key []byte, version int64) (bool, error) {
	result, err := rc.client.pool.Execute(ctx, &operation.RemoveIfUnmodifiedOp{
		Cache:   rc.name,
		Key:     key,
		Version: version,
	})
	if err != nil {
		return false, err
	}
	return result.(*operation.CASResponse).Success, nil
}

// PutRaw stores a key-value pair with an explicit media type identifier.
func (rc *RemoteCache) PutRaw(ctx context.Context, key, value []byte, mediaType int32, opts ...PutOption) error {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	_, err := rc.client.pool.Execute(ctx, &operation.PutOp{
		Cache:     rc.name,
		Key:       key,
		Value:     value,
		Lifespan:  cfg.lifespan,
		MaxIdle:   cfg.maxIdle,
		MediaType: mediaType,
		OpFlags:   cfg.flags,
	})
	return err
}

// GetAndPut stores a key-value pair and returns the previous value, or nil if none existed.
func (rc *RemoteCache) GetAndPut(ctx context.Context, key, value []byte, opts ...PutOption) ([]byte, error) {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	cfg.flags |= int32(FlagForceReturnValue)
	result, err := rc.client.pool.Execute(ctx, &operation.PutOp{
		Cache:    rc.name,
		Key:      key,
		Value:    value,
		Lifespan: cfg.lifespan,
		MaxIdle:  cfg.maxIdle,
		OpFlags:  cfg.flags,
	})
	if err != nil {
		return nil, err
	}
	if resp, ok := result.(*operation.PutResponse); ok && len(resp.PreviousValue) > 0 {
		return resp.PreviousValue, nil
	}
	return nil, nil
}

// Remove deletes the entry for a key from the cache.
func (rc *RemoteCache) Remove(ctx context.Context, key []byte, opts ...RemoveOption) error {
	cfg := &removeConfig{}
	for _, o := range opts {
		o(cfg)
	}
	_, err := rc.client.pool.Execute(ctx, &operation.RemoveOp{
		Cache:     rc.name,
		Key:       key,
		MediaType: cfg.mediaType,
		OpFlags:   cfg.flags,
	})
	return err
}

// GetAndRemove deletes the entry for a key and returns its previous value.
func (rc *RemoteCache) GetAndRemove(ctx context.Context, key []byte, opts ...RemoveOption) ([]byte, bool, error) {
	cfg := &removeConfig{}
	for _, o := range opts {
		o(cfg)
	}
	cfg.flags |= int32(FlagForceReturnValue)
	result, err := rc.client.pool.Execute(ctx, &operation.RemoveOp{
		Cache:     rc.name,
		Key:       key,
		MediaType: cfg.mediaType,
		OpFlags:   cfg.flags,
	})
	if err != nil {
		return nil, false, err
	}
	resp := result.(*operation.RemoveResponse)
	return resp.PreviousValue, resp.Existed, nil
}

// PutIfAbsent stores a key-value pair only if the key does not already exist.
// Returns true if the entry was stored, false if the key was already present.
func (rc *RemoteCache) PutIfAbsent(ctx context.Context, key, value []byte, opts ...PutOption) (bool, error) {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	result, err := rc.client.pool.Execute(ctx, &operation.PutIfAbsentOp{
		Cache:    rc.name,
		Key:      key,
		Value:    value,
		Lifespan: cfg.lifespan,
		MaxIdle:  cfg.maxIdle,
		OpFlags:  cfg.flags,
	})
	if err != nil {
		return false, err
	}
	return result.(*operation.CASResponse).Success, nil
}

// PutIfAbsentWithPrevious stores a key-value pair only if the key does not already exist.
// Returns the existing value and false if the key was already present, or nil and true if stored.
func (rc *RemoteCache) PutIfAbsentWithPrevious(ctx context.Context, key, value []byte, opts ...PutOption) ([]byte, bool, error) {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	cfg.flags |= int32(FlagForceReturnValue)
	result, err := rc.client.pool.Execute(ctx, &operation.PutIfAbsentOp{
		Cache:    rc.name,
		Key:      key,
		Value:    value,
		Lifespan: cfg.lifespan,
		MaxIdle:  cfg.maxIdle,
		OpFlags:  cfg.flags,
	})
	if err != nil {
		return nil, false, err
	}
	resp := result.(*operation.CASResponse)
	return resp.PreviousValue, resp.Success, nil
}

// Replace replaces an existing entry's value only if the key already exists.
// Returns true if the replacement succeeded, false if the key did not exist.
func (rc *RemoteCache) Replace(ctx context.Context, key, value []byte, opts ...PutOption) (bool, error) {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	result, err := rc.client.pool.Execute(ctx, &operation.ReplaceOp{
		Cache:    rc.name,
		Key:      key,
		Value:    value,
		Lifespan: cfg.lifespan,
		MaxIdle:  cfg.maxIdle,
		OpFlags:  cfg.flags,
	})
	if err != nil {
		return false, err
	}
	return result.(*operation.CASResponse).Success, nil
}

// ReplaceWithPrevious replaces an existing entry's value and returns the previous value.
// Returns the previous value and true if the key existed, or nil and false otherwise.
func (rc *RemoteCache) ReplaceWithPrevious(ctx context.Context, key, value []byte, opts ...PutOption) ([]byte, bool, error) {
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}
	cfg.flags |= int32(FlagForceReturnValue)
	result, err := rc.client.pool.Execute(ctx, &operation.ReplaceOp{
		Cache:    rc.name,
		Key:      key,
		Value:    value,
		Lifespan: cfg.lifespan,
		MaxIdle:  cfg.maxIdle,
		OpFlags:  cfg.flags,
	})
	if err != nil {
		return nil, false, err
	}
	resp := result.(*operation.CASResponse)
	return resp.PreviousValue, resp.Success, nil
}

// ContainsKey returns true if the cache contains an entry for the given key.
func (rc *RemoteCache) ContainsKey(ctx context.Context, key []byte) (bool, error) {
	result, err := rc.client.pool.Execute(ctx, &operation.ContainsKeyOp{
		Cache: rc.name,
		Key:   key,
	})
	if err != nil {
		return false, err
	}
	return result.(bool), nil
}

// Clear removes all entries from the cache.
func (rc *RemoteCache) Clear(ctx context.Context) error {
	_, err := rc.client.pool.Execute(ctx, &operation.ClearOp{
		Cache: rc.name,
	})
	return err
}

// Stats returns a map of server-side cache statistics (e.g. "timeSinceStart", "currentNumberOfEntries").
func (rc *RemoteCache) Stats(ctx context.Context) (map[string]string, error) {
	result, err := rc.client.pool.Execute(ctx, &operation.StatsOp{
		Cache: rc.name,
	})
	if err != nil {
		return nil, err
	}
	return result.(map[string]string), nil
}

// Size returns the number of entries in the cache.
func (rc *RemoteCache) Size(ctx context.Context) (int32, error) {
	result, err := rc.client.pool.Execute(ctx, &operation.SizeOp{
		Cache: rc.name,
	})
	if err != nil {
		return 0, err
	}
	return result.(int32), nil
}

// PutAll stores multiple key-value pairs in a single bulk operation.
// When hash-distribution-aware, entries are grouped by owner and dispatched in parallel.
func (rc *RemoteCache) PutAll(ctx context.Context, entries map[string][]byte, opts ...PutOption) error {
	if len(entries) == 0 {
		return nil
	}
	cfg := &putConfig{}
	for _, o := range opts {
		o(cfg)
	}

	ch := rc.client.pool.ConsistentHash(rc.name)
	groups := make(map[string][]operation.PutAllEntry)
	for k, v := range entries {
		keyBytes := []byte(k)
		addr := ""
		if ch != nil {
			addr = ch.GetPrimaryOwner(keyBytes)
		}
		groups[addr] = append(groups[addr], operation.PutAllEntry{Key: keyBytes, Value: v})
	}

	return rc.dispatchPutAll(ctx, groups, cfg)
}

func (rc *RemoteCache) dispatchPutAll(ctx context.Context, groups map[string][]operation.PutAllEntry, cfg *putConfig) error {
	if len(groups) == 1 {
		for addr, entries := range groups {
			op := &operation.PutAllOp{
				Cache:    rc.name,
				Entries:  entries,
				Lifespan: cfg.lifespan,
				MaxIdle:  cfg.maxIdle,
			}
			if addr != "" {
				_, err := rc.client.pool.ExecuteOn(ctx, addr, op)
				return err
			}
			_, err := rc.client.pool.Execute(ctx, op)
			return err
		}
	}

	var wg sync.WaitGroup
	errs := make(chan error, len(groups))
	for addr, entries := range groups {
		wg.Add(1)
		go func(addr string, entries []operation.PutAllEntry) {
			defer wg.Done()
			op := &operation.PutAllOp{
				Cache:    rc.name,
				Entries:  entries,
				Lifespan: cfg.lifespan,
				MaxIdle:  cfg.maxIdle,
			}
			if addr != "" {
				_, err := rc.client.pool.ExecuteOn(ctx, addr, op)
				if err != nil {
					errs <- err
				}
			} else {
				_, err := rc.client.pool.Execute(ctx, op)
				if err != nil {
					errs <- err
				}
			}
		}(addr, entries)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		return err
	}
	return nil
}

// GetAll retrieves multiple keys in a single bulk operation.
// When hash-distribution-aware, keys are grouped by owner and fetched in parallel.
func (rc *RemoteCache) GetAll(ctx context.Context, keys []string) (map[string][]byte, error) {
	if len(keys) == 0 {
		return map[string][]byte{}, nil
	}

	ch := rc.client.pool.ConsistentHash(rc.name)
	groups := make(map[string][][]byte)
	for _, k := range keys {
		keyBytes := []byte(k)
		addr := ""
		if ch != nil {
			addr = ch.GetPrimaryOwner(keyBytes)
		}
		groups[addr] = append(groups[addr], keyBytes)
	}

	return rc.dispatchGetAll(ctx, groups)
}

func (rc *RemoteCache) dispatchGetAll(ctx context.Context, groups map[string][][]byte) (map[string][]byte, error) {
	if len(groups) == 1 {
		for addr, keys := range groups {
			op := &operation.GetAllOp{Cache: rc.name, Keys: keys}
			var result any
			var err error
			if addr != "" {
				result, err = rc.client.pool.ExecuteOn(ctx, addr, op)
			} else {
				result, err = rc.client.pool.Execute(ctx, op)
			}
			if err != nil {
				return nil, err
			}
			return entriesToMap(result.([]operation.GetAllEntry)), nil
		}
	}

	type getResult struct {
		entries []operation.GetAllEntry
		err     error
	}

	results := make(chan getResult, len(groups))
	var wg sync.WaitGroup
	for addr, keys := range groups {
		wg.Add(1)
		go func(addr string, keys [][]byte) {
			defer wg.Done()
			op := &operation.GetAllOp{Cache: rc.name, Keys: keys}
			var result any
			var err error
			if addr != "" {
				result, err = rc.client.pool.ExecuteOn(ctx, addr, op)
			} else {
				result, err = rc.client.pool.Execute(ctx, op)
			}
			if err != nil {
				results <- getResult{err: err}
				return
			}
			results <- getResult{entries: result.([]operation.GetAllEntry)}
		}(addr, keys)
	}
	wg.Wait()
	close(results)

	merged := make(map[string][]byte)
	for r := range results {
		if r.err != nil {
			return nil, r.err
		}
		for _, e := range r.entries {
			merged[string(e.Key)] = e.Value
		}
	}
	return merged, nil
}

func entriesToMap(entries []operation.GetAllEntry) map[string][]byte {
	m := make(map[string][]byte, len(entries))
	for _, e := range entries {
		m[string(e.Key)] = e.Value
	}
	return m
}

// GetRaw retrieves the value for a key with an explicit media type identifier.
func (rc *RemoteCache) GetRaw(ctx context.Context, key []byte, mediaType int32, opts ...GetOption) ([]byte, bool, error) {
	cfg := &getConfig{}
	for _, o := range opts {
		o(cfg)
	}
	result, err := rc.client.pool.Execute(ctx, &operation.GetOp{
		Cache:     rc.name,
		Key:       key,
		MediaType: mediaType,
		OpFlags:   cfg.flags,
	})
	if err != nil {
		return nil, false, err
	}
	resp := result.(*operation.GetResponse)
	return resp.Value, resp.Found, nil
}

// QueryEntry holds a single decoded query result.
// For entity queries, TypeName and Value are populated.
// For projection queries, Projections contains the typed column values.
type QueryEntry struct {
	TypeName    string
	Value       []byte
	Projections []any
}

// QueryResult holds the results of an Ickle query execution.
type QueryResult struct {
	NumResults     int
	ProjectionSize int
	Entries        []QueryEntry
	HitCount       int
	HitCountExact  bool
}

// Query executes an Ickle query against the cache and returns the results.
func (rc *RemoteCache) Query(ctx context.Context, ickle string, opts ...QueryOption) (*QueryResult, error) {
	cfg := &queryConfig{maxResults: -1}
	for _, o := range opts {
		o(cfg)
	}

	var params []protostream.QueryParam
	for _, p := range cfg.params {
		wrapped, err := wrapQueryParam(p.value)
		if err != nil {
			return nil, fmt.Errorf("query param %q: %w", p.name, err)
		}
		params = append(params, protostream.QueryParam{Name: p.name, Value: wrapped})
	}

	reqBytes := protostream.EncodeQueryRequest(ickle, cfg.startOffset, cfg.maxResults, params)
	result, err := rc.client.pool.Execute(ctx, &operation.QueryOp{
		Cache:   rc.name,
		Request: reqBytes,
	})
	if err != nil {
		return nil, err
	}

	respBytes := result.([]byte)
	resp, err := protostream.DecodeQueryResponse(respBytes)
	if err != nil {
		return nil, fmt.Errorf("decode query response: %w", err)
	}

	entries := make([]QueryEntry, len(resp.Results))
	if resp.ProjectionSize > 0 {
		for i, r := range resp.Results {
			e, err := unwrapProjection(r, int(resp.ProjectionSize))
			if err != nil {
				return nil, fmt.Errorf("unwrap projection result %d: %w", i, err)
			}
			entries[i] = e
		}
	} else {
		for i, r := range resp.Results {
			e, err := unwrapEntity(r)
			if err != nil {
				return nil, fmt.Errorf("unwrap entity result %d: %w", i, err)
			}
			entries[i] = e
		}
	}

	return &QueryResult{
		NumResults:     int(resp.NumResults),
		ProjectionSize: int(resp.ProjectionSize),
		Entries:        entries,
		HitCount:       int(resp.HitCount),
		HitCountExact:  resp.HitCountExact,
	}, nil
}

func unwrapEntity(data []byte) (QueryEntry, error) {
	var typeName string
	var msgBytes []byte
	var wrappedBytes []byte
	err := protostream.ScanFields(data, func(fieldNumber, wireType int, value []byte) error {
		switch fieldNumber {
		case 10: // WRAPPED_BYTES: contains a nested WrappedMessage
			wrappedBytes = value
		case 16: // WRAPPED_TYPE_NAME
			typeName = string(value)
		case 17: // WRAPPED_MESSAGE
			msgBytes = value
		}
		return nil
	})
	if err != nil {
		return QueryEntry{}, err
	}
	if wrappedBytes != nil {
		inner, innerType, err := protostream.UnwrapMessage(wrappedBytes)
		if err != nil {
			return QueryEntry{}, err
		}
		return QueryEntry{TypeName: innerType, Value: inner}, nil
	}
	if msgBytes != nil {
		return QueryEntry{TypeName: typeName, Value: msgBytes}, nil
	}
	return QueryEntry{}, fmt.Errorf("no entity data in query result")
}

func unwrapProjection(data []byte, projSize int) (QueryEntry, error) {
	inner, err := protostream.UnwrapBytes(data)
	if err != nil {
		return QueryEntry{}, err
	}
	columns := make([]any, projSize)
	err = protostream.ScanFields(inner, func(fieldNumber, wireType int, value []byte) error {
		if fieldNumber < 1 || fieldNumber > projSize {
			return nil
		}
		if wireType != 2 {
			return fmt.Errorf("projection column %d: expected length-delimited, got wire type %d", fieldNumber, wireType)
		}
		v, err := protostream.UnwrapValue(value)
		if err != nil {
			return fmt.Errorf("projection column %d: %w", fieldNumber, err)
		}
		columns[fieldNumber-1] = v
		return nil
	})
	if err != nil {
		return QueryEntry{}, err
	}
	return QueryEntry{Projections: columns}, nil
}

func wrapQueryParam(value any) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return protostream.WrapString(v), nil
	case int32:
		return protostream.WrapInt32(v), nil
	case int:
		return protostream.WrapInt32(int32(v)), nil
	case int64:
		return protostream.WrapInt64(v), nil
	case float64:
		return protostream.WrapDouble(v), nil
	case float32:
		return protostream.WrapFloat(v), nil
	case []float32:
		return protostream.WrapFloatArray(v), nil
	case bool:
		return protostream.WrapBool(v), nil
	default:
		return nil, fmt.Errorf("unsupported type %T", value)
	}
}

func buildTLSConfig(parsed *parsedURI, cfg *clientConfig) (*tls.Config, error) {
	trustStorePath := parsed.trustStorePath
	if cfg.trustStorePath != "" {
		trustStorePath = cfg.trustStorePath
	}
	clientCertPath := parsed.clientCertPath
	if cfg.clientCertPath != "" {
		clientCertPath = cfg.clientCertPath
	}
	clientKeyPath := parsed.clientKeyPath
	if cfg.clientKeyPath != "" {
		clientKeyPath = cfg.clientKeyPath
	}
	sniHostName := parsed.sniHostName
	if cfg.sniHostName != "" {
		sniHostName = cfg.sniHostName
	}
	sslHostnameValidation := parsed.sslHostnameValidation
	if cfg.sslHostnameValidation != nil {
		sslHostnameValidation = cfg.sslHostnameValidation
	}

	hasTLSFields := trustStorePath != "" || clientCertPath != "" || sniHostName != "" || sslHostnameValidation != nil
	if !parsed.tls && cfg.tls == nil && !hasTLSFields {
		return nil, nil
	}

	var tlsConfig *tls.Config
	if cfg.tls != nil {
		tlsConfig = cfg.tls.Clone()
	} else {
		tlsConfig = &tls.Config{}
	}

	if trustStorePath != "" {
		pem, err := os.ReadFile(trustStorePath)
		if err != nil {
			return nil, fmt.Errorf("read trust store %q: %w", trustStorePath, err)
		}
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no valid certificates found in %q", trustStorePath)
		}
		tlsConfig.RootCAs = pool
	}

	if clientCertPath != "" {
		if clientKeyPath == "" {
			return nil, fmt.Errorf("client_cert requires client_key (key_store_password or client_key)")
		}
		cert, err := tls.LoadX509KeyPair(clientCertPath, clientKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load client certificate: %w", err)
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, cert)
	}

	if sniHostName != "" {
		tlsConfig.ServerName = sniHostName
	}

	if sslHostnameValidation != nil && !*sslHostnameValidation {
		tlsConfig.InsecureSkipVerify = true
	}

	return tlsConfig, nil
}
