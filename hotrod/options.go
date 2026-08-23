package hotrod

import (
	"crypto/tls"
	"log/slog"
	"time"

	"infinispan.org/go-client/internal/codec"
)

const (
	IntelligenceBasic         = codec.IntelligenceBasic
	IntelligenceTopologyAware = codec.IntelligenceTopologyAware
	IntelligenceHashDistAware = codec.IntelligenceHashDistAware

	ProtocolVersion41 = codec.Version41
)

// Flag controls optional Hot Rod operation behavior such as forcing return values or skipping indexing.
type Flag int32

const (
	FlagForceReturnValue         Flag = 0x0001
	FlagDefaultLifespan          Flag = 0x0002
	FlagDefaultMaxIdle           Flag = 0x0004
	FlagSkipCacheLoad            Flag = 0x0008
	FlagSkipIndexing             Flag = 0x0010
	FlagSkipListenerNotification Flag = 0x0020
)

// Option configures a Client.
type Option func(*clientConfig)

type clientConfig struct {
	tls                   *tls.Config
	authMechanism         string
	authUsername          string
	authPassword          string
	authToken             string
	connectTimeout        time.Duration
	socketTimeout         time.Duration
	tcpNoDelay            *bool
	tcpKeepAlive          *bool
	trustStorePath        string
	clientCertPath        string
	clientKeyPath         string
	sniHostName           string
	sslHostnameValidation *bool
	clientIntelligence    byte
	protocolVersion       byte
	logger                *slog.Logger
}

// WithTLS sets a custom TLS configuration for server connections.
func WithTLS(c *tls.Config) Option {
	return func(cfg *clientConfig) {
		cfg.tls = c
	}
}

// WithTrustStore sets the path to a PEM-encoded CA certificate file for TLS verification.
func WithTrustStore(path string) Option {
	return func(cfg *clientConfig) {
		cfg.trustStorePath = path
	}
}

// WithClientCert sets the client certificate and private key for mutual TLS authentication.
func WithClientCert(certPath, keyPath string) Option {
	return func(cfg *clientConfig) {
		cfg.clientCertPath = certPath
		cfg.clientKeyPath = keyPath
	}
}

// WithSNIHostName sets the TLS Server Name Indication (SNI) hostname.
func WithSNIHostName(name string) Option {
	return func(cfg *clientConfig) {
		cfg.sniHostName = name
	}
}

// WithSSLHostnameValidation enables or disables TLS hostname verification.
func WithSSLHostnameValidation(b bool) Option {
	return func(cfg *clientConfig) {
		cfg.sslHostnameValidation = &b
	}
}

// WithAuth sets the SASL authentication mechanism and credentials.
func WithAuth(mechanism, username, password string) Option {
	return func(cfg *clientConfig) {
		cfg.authMechanism = mechanism
		cfg.authUsername = username
		cfg.authPassword = password
	}
}

// WithToken sets the OAuth2 bearer token for OAUTHBEARER SASL authentication.
func WithToken(token string) Option {
	return func(cfg *clientConfig) {
		cfg.authToken = token
	}
}

// WithConnectTimeout sets the TCP connection timeout.
func WithConnectTimeout(d time.Duration) Option {
	return func(cfg *clientConfig) {
		cfg.connectTimeout = d
	}
}

// WithSocketTimeout sets the timeout for individual socket read/write operations.
func WithSocketTimeout(d time.Duration) Option {
	return func(cfg *clientConfig) {
		cfg.socketTimeout = d
	}
}

// WithTCPNoDelay enables or disables Nagle's algorithm on connections.
func WithTCPNoDelay(b bool) Option {
	return func(cfg *clientConfig) {
		cfg.tcpNoDelay = &b
	}
}

// WithTCPKeepAlive enables or disables TCP keep-alive probes on connections.
func WithTCPKeepAlive(b bool) Option {
	return func(cfg *clientConfig) {
		cfg.tcpKeepAlive = &b
	}
}

// WithClientIntelligence sets the client intelligence level (basic, topology-aware, or hash-distribution-aware).
func WithClientIntelligence(level byte) Option {
	return func(cfg *clientConfig) {
		cfg.clientIntelligence = level
	}
}

// WithProtocolVersion sets the Hot Rod protocol version. Default is ProtocolVersion41.
func WithProtocolVersion(v byte) Option {
	return func(cfg *clientConfig) {
		cfg.protocolVersion = v
	}
}

// WithLogger sets a custom structured logger for the client.
func WithLogger(l *slog.Logger) Option {
	return func(cfg *clientConfig) {
		cfg.logger = l
	}
}

// PutOption configures a Put, PutAll, or ReplaceIfUnmodified operation.
type PutOption func(*putConfig)

type putConfig struct {
	lifespan time.Duration
	maxIdle  time.Duration
	flags    int32
}

// WithLifespan sets the maximum time an entry can live in the cache before expiring.
func WithLifespan(d time.Duration) PutOption {
	return func(cfg *putConfig) {
		cfg.lifespan = d
	}
}

// WithMaxIdle sets the maximum idle time before an entry is considered expired.
func WithMaxIdle(d time.Duration) PutOption {
	return func(cfg *putConfig) {
		cfg.maxIdle = d
	}
}

// WithPutFlag sets one or more operation flags on a put operation.
func WithPutFlag(flags ...Flag) PutOption {
	return func(cfg *putConfig) {
		for _, f := range flags {
			cfg.flags |= int32(f)
		}
	}
}

// GetOption configures a Get operation.
type GetOption func(*getConfig)

type getConfig struct {
	flags int32
}

// WithGetFlag sets one or more operation flags on a get operation.
func WithGetFlag(flags ...Flag) GetOption {
	return func(cfg *getConfig) {
		for _, f := range flags {
			cfg.flags |= int32(f)
		}
	}
}

// RemoveOption configures a Remove operation.
type RemoveOption func(*removeConfig)

type removeConfig struct {
	flags int32
}

// WithRemoveFlag sets one or more operation flags on a remove operation.
func WithRemoveFlag(flags ...Flag) RemoveOption {
	return func(cfg *removeConfig) {
		for _, f := range flags {
			cfg.flags |= int32(f)
		}
	}
}

// MultimapOption configures a MultimapCache.
type MultimapOption func(*multimapConfig)

type multimapConfig struct {
	supportsDuplicates bool
}

// WithDuplicates controls whether the multimap allows duplicate key-value pairs.
func WithDuplicates(b bool) MultimapOption {
	return func(cfg *multimapConfig) {
		cfg.supportsDuplicates = b
	}
}

// QueryOption configures a Query operation.
type QueryOption func(*queryConfig)

type queryConfig struct {
	maxResults  int32
	startOffset int64
	params      []queryParam
}

type queryParam struct {
	name  string
	value any
}

// WithQueryParam adds a named parameter binding to the Ickle query.
func WithQueryParam(name string, value any) QueryOption {
	return func(cfg *queryConfig) {
		cfg.params = append(cfg.params, queryParam{name: name, value: value})
	}
}

// WithQueryMaxResults limits the number of results returned by a query.
func WithQueryMaxResults(n int) QueryOption {
	return func(cfg *queryConfig) {
		cfg.maxResults = int32(n)
	}
}

// WithQueryStartOffset sets the starting offset for paginated query results.
func WithQueryStartOffset(offset int64) QueryOption {
	return func(cfg *queryConfig) {
		cfg.startOffset = offset
	}
}

// TxOption configures a WithTransaction call.
type TxOption func(*txConfig)

type txConfig struct {
	timeout time.Duration
}

// WithTxTimeout sets the server-side transaction timeout. Default is 60 seconds.
func WithTxTimeout(d time.Duration) TxOption {
	return func(cfg *txConfig) {
		cfg.timeout = d
	}
}

// IteratorOption configures a cache Iterator.
type IteratorOption func(*iteratorConfig)

type iteratorConfig struct {
	batchSize       int
	includeMetadata bool
}

// WithIteratorBatchSize sets the number of entries fetched per server round-trip. Default is 1000.
func WithIteratorBatchSize(n int) IteratorOption {
	return func(cfg *iteratorConfig) {
		cfg.batchSize = n
	}
}

// WithIteratorMetadata requests that the server include entry metadata (creation time, lifespan, version, etc.).
func WithIteratorMetadata() IteratorOption {
	return func(cfg *iteratorConfig) {
		cfg.includeMetadata = true
	}
}
