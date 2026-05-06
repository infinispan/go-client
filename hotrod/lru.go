package hotrod

import (
	"container/list"
	"sync"
)

type lruEntry struct {
	key   string
	value []byte
}

type lruStore struct {
	mu       sync.Mutex
	max      int
	items    map[string]*list.Element
	eviction *list.List
	onRemove func(key string)
}

func newLRU(maxEntries int, onRemove func(key string)) *lruStore {
	return &lruStore{
		max:      maxEntries,
		items:    make(map[string]*list.Element),
		eviction: list.New(),
		onRemove: onRemove,
	}
}

func (l *lruStore) Get(key string) ([]byte, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	el, ok := l.items[key]
	if !ok {
		return nil, false
	}
	l.eviction.MoveToFront(el)
	return el.Value.(*lruEntry).value, true
}

func (l *lruStore) Put(key string, value []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if el, ok := l.items[key]; ok {
		el.Value.(*lruEntry).value = value
		l.eviction.MoveToFront(el)
		return
	}
	if l.eviction.Len() >= l.max {
		l.evictOldestLocked()
	}
	el := l.eviction.PushFront(&lruEntry{key: key, value: value})
	l.items[key] = el
}

func (l *lruStore) Remove(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	el, ok := l.items[key]
	if !ok {
		return
	}
	l.eviction.Remove(el)
	delete(l.items, key)
	if l.onRemove != nil {
		l.onRemove(key)
	}
}

func (l *lruStore) Clear() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.items = make(map[string]*list.Element)
	l.eviction.Init()
}

func (l *lruStore) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.items)
}

// Keys returns all keys currently in the cache. Caller must not hold l.mu.
func (l *lruStore) Keys() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	keys := make([]string, 0, len(l.items))
	for k := range l.items {
		keys = append(keys, k)
	}
	return keys
}

func (l *lruStore) evictOldestLocked() {
	el := l.eviction.Back()
	if el == nil {
		return
	}
	l.eviction.Remove(el)
	entry := el.Value.(*lruEntry)
	delete(l.items, entry.key)
	if l.onRemove != nil {
		l.onRemove(entry.key)
	}
}
