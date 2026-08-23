package connection

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"infinispan.org/go-client/internal/auth"
	"infinispan.org/go-client/internal/codec"
	"infinispan.org/go-client/internal/hash"
	"infinispan.org/go-client/internal/operation"
)

type PoolConfig struct {
	InitialServers     []string
	TLSConfig          *tls.Config
	AuthMechanism      auth.Mechanism
	ClientIntelligence byte
	ProtocolVersion    byte
	ConnectTimeout     time.Duration
	SocketTimeout      time.Duration
	TCPNoDelay         *bool
	TCPKeepAlive       *bool
	Logger             *slog.Logger
}

type listenerState struct {
	addr string
}

type Pool struct {
	mu                 sync.RWMutex
	conns              map[string]*Conn
	balancer           *Balancer
	consistentHashes   map[string]*hash.ConsistentHash
	listenersMu        sync.Mutex
	listenerStates     map[string]*listenerState // hex(listenerID) → state
	topoID             int32
	clientIntelligence byte
	protocolVersion    byte
	connectTimeout     time.Duration
	socketTimeout      time.Duration
	tcpNoDelay         bool
	tcpKeepAlive       bool
	tlsConfig          *tls.Config
	authMech           auth.Mechanism
	logger             *slog.Logger
}

func NewPool(cfg PoolConfig) *Pool {
	ci := cfg.ClientIntelligence
	if ci == 0 {
		ci = codec.IntelligenceHashDistAware
	}
	tcpNoDelay := true
	if cfg.TCPNoDelay != nil {
		tcpNoDelay = *cfg.TCPNoDelay
	}
	var tcpKeepAlive bool
	if cfg.TCPKeepAlive != nil {
		tcpKeepAlive = *cfg.TCPKeepAlive
	}
	p := &Pool{
		conns:              make(map[string]*Conn),
		balancer:           NewBalancer(cfg.InitialServers),
		consistentHashes:   make(map[string]*hash.ConsistentHash),
		listenerStates:     make(map[string]*listenerState),
		clientIntelligence: ci,
		protocolVersion:    cfg.ProtocolVersion,
		connectTimeout:     cfg.ConnectTimeout,
		socketTimeout:      cfg.SocketTimeout,
		tcpNoDelay:         tcpNoDelay,
		tcpKeepAlive:       tcpKeepAlive,
		tlsConfig:          cfg.TLSConfig,
		authMech:           cfg.AuthMechanism,
		logger:             cfg.Logger,
	}
	return p
}

func (p *Pool) Connect(ctx context.Context) error {
	for _, addr := range p.balancer.Servers() {
		if err := p.ensureConn(ctx, addr); err != nil {
			return fmt.Errorf("connect to %s: %w", addr, err)
		}
	}
	return nil
}

func (p *Pool) Execute(ctx context.Context, op operation.Operation) (any, error) {
	if p.socketTimeout > 0 {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, p.socketTimeout)
			defer cancel()
		}
	}
	addr := p.routeOperation(op)
	if addr == "" {
		return nil, fmt.Errorf("no servers available")
	}
	if err := p.ensureConn(ctx, addr); err != nil {
		return nil, err
	}
	p.mu.RLock()
	conn := p.conns[addr]
	p.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("no connection to %s", addr)
	}
	return conn.Execute(ctx, op)
}

func (p *Pool) routeOperation(op operation.Operation) string {
	if keyed, ok := op.(operation.KeyedOperation); ok {
		p.mu.RLock()
		ch := p.consistentHashes[string(op.CacheName())]
		p.mu.RUnlock()
		if ch != nil {
			if addr := ch.PrimaryOwner(keyed.KeyBytes()); addr != "" {
				return addr
			}
		}
	}
	return p.balancer.Next()
}

func (p *Pool) ensureConn(ctx context.Context, addr string) error {
	p.mu.RLock()
	_, exists := p.conns[addr]
	p.mu.RUnlock()
	if exists {
		return nil
	}

	// Dial without holding mu — the readLoop's topology callback needs mu.
	conn, err := p.dialAndSetup(ctx, addr)
	if err != nil {
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	if _, exists := p.conns[addr]; exists {
		conn.Close()
		return nil
	}
	p.conns[addr] = conn
	return nil
}

func (p *Pool) dialAndSetup(ctx context.Context, addr string) (*Conn, error) {
	dialCtx := ctx
	if p.connectTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, p.connectTimeout)
		defer cancel()
	}

	d := net.Dialer{}
	if !p.tcpKeepAlive {
		d.KeepAlive = -1
	}
	rawConn, err := d.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	if tc, ok := rawConn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(p.tcpNoDelay)
	}

	var netConn net.Conn = rawConn
	if p.tlsConfig != nil {
		tlsConn := tls.Client(rawConn, p.tlsConfig)
		if err := tlsConn.HandshakeContext(dialCtx); err != nil {
			rawConn.Close()
			return nil, fmt.Errorf("tls handshake %s: %w", addr, err)
		}
		netConn = tlsConn
	}

	conn := NewConn(netConn, addr, p.handleTopologyUpdate, p.clientIntelligence, p.protocolVersion, p.logger)

	// Ping
	if _, err := conn.Execute(ctx, &operation.PingOp{}); err != nil {
		conn.Close()
		return nil, fmt.Errorf("ping %s: %w", addr, err)
	}

	// Auth
	if p.authMech != nil {
		mechCopy := p.newAuthMechanism()
		if err := conn.Authenticate(ctx, mechCopy); err != nil {
			conn.Close()
			return nil, fmt.Errorf("auth %s: %w", addr, err)
		}
	}

	return conn, nil
}

func (p *Pool) newAuthMechanism() auth.Mechanism {
	switch m := p.authMech.(type) {
	case *auth.ScramSHA256:
		return auth.NewScramSHA256(m.Username(), m.Password())
	case *auth.Plain:
		return auth.NewPlain(m.Username(), m.Password())
	case *auth.External:
		return auth.NewExternal()
	case *auth.OAuthBearer:
		return auth.NewOAuthBearer(m.Token())
	default:
		return p.authMech
	}
}

func (p *Pool) AddListener(ctx context.Context, op *operation.AddClientListenerOp, ch chan<- *operation.CacheEntryEvent) error {
	addr := p.balancer.Next()
	if addr == "" {
		return fmt.Errorf("no servers available")
	}
	if err := p.ensureConn(ctx, addr); err != nil {
		return err
	}
	p.mu.RLock()
	conn := p.conns[addr]
	p.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("no connection to %s", addr)
	}

	entry := &ListenerEntry{Ch: ch}
	if err := conn.ExecuteListener(ctx, op, op.ListenerID, entry); err != nil {
		return err
	}

	key := hex.EncodeToString(op.ListenerID)
	p.listenersMu.Lock()
	p.listenerStates[key] = &listenerState{addr: addr}
	p.listenersMu.Unlock()
	return nil
}

func (p *Pool) AddCustomListener(ctx context.Context, op *operation.AddClientListenerOp, ch chan<- []byte) error {
	addr := p.balancer.Next()
	if addr == "" {
		return fmt.Errorf("no servers available")
	}
	if err := p.ensureConn(ctx, addr); err != nil {
		return err
	}
	p.mu.RLock()
	conn := p.conns[addr]
	p.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("no connection to %s", addr)
	}

	entry := &ListenerEntry{CustomCh: ch}
	if err := conn.ExecuteListener(ctx, op, op.ListenerID, entry); err != nil {
		return err
	}

	key := hex.EncodeToString(op.ListenerID)
	p.listenersMu.Lock()
	p.listenerStates[key] = &listenerState{addr: addr}
	p.listenersMu.Unlock()
	return nil
}

func (p *Pool) AddCounterListener(ctx context.Context, op *operation.CounterAddListenerOp, ch chan<- *operation.CounterEvent) error {
	addr := p.balancer.Next()
	if addr == "" {
		return fmt.Errorf("no servers available")
	}
	if err := p.ensureConn(ctx, addr); err != nil {
		return err
	}
	p.mu.RLock()
	conn := p.conns[addr]
	p.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("no connection to %s", addr)
	}

	entry := &ListenerEntry{CounterCh: ch}
	if err := conn.ExecuteListener(ctx, op, op.ListenerID, entry); err != nil {
		return err
	}

	key := hex.EncodeToString(op.ListenerID)
	p.listenersMu.Lock()
	p.listenerStates[key] = &listenerState{addr: addr}
	p.listenersMu.Unlock()
	return nil
}

func (p *Pool) RemoveCounterListener(ctx context.Context, op *operation.CounterRemoveListenerOp) error {
	key := hex.EncodeToString(op.ListenerID)
	p.listenersMu.Lock()
	state, ok := p.listenerStates[key]
	if ok {
		delete(p.listenerStates, key)
	}
	p.listenersMu.Unlock()
	if !ok {
		return fmt.Errorf("listener not found")
	}

	p.mu.RLock()
	conn := p.conns[state.addr]
	p.mu.RUnlock()

	if conn != nil {
		conn.UnregisterListener(key)
		_, err := conn.Execute(ctx, op)
		return err
	}
	return nil
}

func (p *Pool) AddBloomListener(ctx context.Context, op *operation.AddBloomNearCacheListenerOp, ch chan<- *operation.CacheEntryEvent) error {
	addr := p.balancer.Next()
	if addr == "" {
		return fmt.Errorf("no servers available")
	}
	if err := p.ensureConn(ctx, addr); err != nil {
		return err
	}
	p.mu.RLock()
	conn := p.conns[addr]
	p.mu.RUnlock()
	if conn == nil {
		return fmt.Errorf("no connection to %s", addr)
	}

	entry := &ListenerEntry{Ch: ch}
	if err := conn.ExecuteListener(ctx, op, op.ListenerID, entry); err != nil {
		return err
	}

	key := hex.EncodeToString(op.ListenerID)
	p.listenersMu.Lock()
	p.listenerStates[key] = &listenerState{addr: addr}
	p.listenersMu.Unlock()
	return nil
}

func (p *Pool) ExecuteOnListener(ctx context.Context, listenerID []byte, op operation.Operation) (any, error) {
	key := hex.EncodeToString(listenerID)
	p.listenersMu.Lock()
	state, ok := p.listenerStates[key]
	p.listenersMu.Unlock()
	if !ok {
		return nil, fmt.Errorf("listener not found")
	}

	p.mu.RLock()
	conn := p.conns[state.addr]
	p.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("no connection to %s", state.addr)
	}
	return conn.Execute(ctx, op)
}

func (p *Pool) RemoveListener(ctx context.Context, op *operation.RemoveClientListenerOp) error {
	key := hex.EncodeToString(op.ListenerID)
	p.listenersMu.Lock()
	state, ok := p.listenerStates[key]
	if ok {
		delete(p.listenerStates, key)
	}
	p.listenersMu.Unlock()
	if !ok {
		return fmt.Errorf("listener not found")
	}

	p.mu.RLock()
	conn := p.conns[state.addr]
	p.mu.RUnlock()

	if conn != nil {
		conn.UnregisterListener(key)
		_, err := conn.Execute(ctx, op)
		return err
	}
	return nil
}

func (p *Pool) ExecuteWithAddr(ctx context.Context, op operation.Operation) (any, string, error) {
	if p.socketTimeout > 0 {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, p.socketTimeout)
			defer cancel()
		}
	}
	addr := p.routeOperation(op)
	if addr == "" {
		return nil, "", fmt.Errorf("no servers available")
	}
	if err := p.ensureConn(ctx, addr); err != nil {
		return nil, "", err
	}
	p.mu.RLock()
	conn := p.conns[addr]
	p.mu.RUnlock()
	if conn == nil {
		return nil, "", fmt.Errorf("no connection to %s", addr)
	}
	result, err := conn.Execute(ctx, op)
	if err != nil {
		return nil, "", err
	}
	return result, addr, nil
}

func (p *Pool) ExecuteOn(ctx context.Context, addr string, op operation.Operation) (any, error) {
	if p.socketTimeout > 0 {
		if _, ok := ctx.Deadline(); !ok {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, p.socketTimeout)
			defer cancel()
		}
	}
	if err := p.ensureConn(ctx, addr); err != nil {
		return nil, err
	}
	p.mu.RLock()
	conn := p.conns[addr]
	p.mu.RUnlock()
	if conn == nil {
		return nil, fmt.Errorf("no connection to %s", addr)
	}
	return conn.Execute(ctx, op)
}

func (p *Pool) handleTopologyUpdate(update *codec.TopologyUpdate, cacheName string) {
	if update == nil {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()

	if update.ID == p.topoID {
		return
	}
	p.topoID = update.ID

	newAddrs := make([]string, len(update.Servers))
	newSet := make(map[string]bool, len(update.Servers))
	for i, s := range update.Servers {
		addr := s.Addr()
		newAddrs[i] = addr
		newSet[addr] = true
	}
	p.balancer.SetServers(newAddrs)

	for addr, conn := range p.conns {
		if !newSet[addr] {
			p.logger.Info("removing connection to departed server", "addr", addr)
			delete(p.conns, addr)
			conn.Drain()
		}
	}

	if cacheName != "" && update.SegmentOwners != nil && update.HashFunctionVersion > 0 {
		segmentOwners := make([][]string, len(update.SegmentOwners))
		for i, owners := range update.SegmentOwners {
			addrs := make([]string, len(owners))
			for j, idx := range owners {
				if idx < len(newAddrs) {
					addrs[j] = newAddrs[idx]
				}
			}
			segmentOwners[i] = addrs
		}
		p.consistentHashes[cacheName] = hash.NewConsistentHash(update.NumSegments, segmentOwners)
		p.logger.Info("consistent hash updated", "cache", cacheName, "segments", update.NumSegments)
	} else if cacheName != "" && update.HashFunctionVersion == 0 && update.NumSegments > 0 {
		delete(p.consistentHashes, cacheName)
	}

	p.logger.Info("topology updated", "id", update.ID, "servers", newAddrs)
}

func (p *Pool) TopologyServers() []string {
	return p.balancer.Servers()
}

func (p *Pool) ConsistentHash(cacheName string) *hash.ConsistentHash {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.consistentHashes[cacheName]
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for addr, conn := range p.conns {
		conn.Close()
		delete(p.conns, addr)
	}
	return nil
}
