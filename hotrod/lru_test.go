package hotrod

import (
	"sync/atomic"
	"testing"
)

func TestLRU_GetPut(t *testing.T) {
	l := newLRU(10, nil)
	l.Put("a", []byte("1"))
	l.Put("b", []byte("2"))

	val, ok := l.Get("a")
	if !ok || string(val) != "1" {
		t.Errorf("Get(a) = %q, %v; want %q, true", val, ok, "1")
	}

	_, ok = l.Get("missing")
	if ok {
		t.Error("Get(missing) should return false")
	}
}

func TestLRU_Eviction(t *testing.T) {
	l := newLRU(3, nil)
	l.Put("a", []byte("1"))
	l.Put("b", []byte("2"))
	l.Put("c", []byte("3"))
	l.Put("d", []byte("4"))

	if _, ok := l.Get("a"); ok {
		t.Error("expected 'a' to be evicted")
	}
	if val, ok := l.Get("d"); !ok || string(val) != "4" {
		t.Errorf("Get(d) = %q, %v; want %q, true", val, ok, "4")
	}
	if l.Len() != 3 {
		t.Errorf("Len() = %d, want 3", l.Len())
	}
}

func TestLRU_Remove(t *testing.T) {
	l := newLRU(10, nil)
	l.Put("a", []byte("1"))
	l.Remove("a")

	if _, ok := l.Get("a"); ok {
		t.Error("Get(a) should return false after Remove")
	}
	if l.Len() != 0 {
		t.Errorf("Len() = %d after Remove, want 0", l.Len())
	}

	l.Remove("nonexistent")
}

func TestLRU_AccessOrder(t *testing.T) {
	l := newLRU(3, nil)
	l.Put("a", []byte("1"))
	l.Put("b", []byte("2"))
	l.Put("c", []byte("3"))

	l.Get("a")

	l.Put("d", []byte("4"))

	if _, ok := l.Get("b"); ok {
		t.Error("expected 'b' to be evicted (least recently used)")
	}
	if _, ok := l.Get("a"); !ok {
		t.Error("expected 'a' to still be present (was accessed)")
	}
}

func TestLRU_OnRemoveCallback(t *testing.T) {
	var count atomic.Int32
	l := newLRU(2, func(key string) {
		count.Add(1)
	})

	l.Put("a", []byte("1"))
	l.Put("b", []byte("2"))

	l.Remove("a")
	if count.Load() != 1 {
		t.Errorf("callback count = %d after Remove, want 1", count.Load())
	}

	l.Put("c", []byte("3"))
	l.Put("d", []byte("4"))
	if count.Load() != 2 {
		t.Errorf("callback count = %d after eviction, want 2", count.Load())
	}
}

func TestLRU_Update(t *testing.T) {
	l := newLRU(10, nil)
	l.Put("a", []byte("1"))
	l.Put("a", []byte("2"))

	val, ok := l.Get("a")
	if !ok || string(val) != "2" {
		t.Errorf("Get(a) = %q, %v; want %q, true", val, ok, "2")
	}
	if l.Len() != 1 {
		t.Errorf("Len() = %d, want 1 after update", l.Len())
	}
}

func TestLRU_Keys(t *testing.T) {
	l := newLRU(10, nil)
	l.Put("a", []byte("1"))
	l.Put("b", []byte("2"))
	l.Put("c", []byte("3"))

	keys := l.Keys()
	if len(keys) != 3 {
		t.Fatalf("Keys() returned %d keys, want 3", len(keys))
	}
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	for _, expected := range []string{"a", "b", "c"} {
		if !keySet[expected] {
			t.Errorf("Keys() missing %q", expected)
		}
	}
}
