package connection

import (
	"sync"
	"sync/atomic"
)

type Balancer struct {
	mu      sync.RWMutex
	servers []string
	index   atomic.Uint64
}

func NewBalancer(servers []string) *Balancer {
	return &Balancer{servers: servers}
}

func (b *Balancer) Next() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	if len(b.servers) == 0 {
		return ""
	}
	idx := b.index.Add(1) - 1
	return b.servers[idx%uint64(len(b.servers))]
}

func (b *Balancer) SetServers(servers []string) {
	b.mu.Lock()
	b.servers = servers
	b.mu.Unlock()
}

func (b *Balancer) Servers() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	result := make([]string, len(b.servers))
	copy(result, b.servers)
	return result
}
