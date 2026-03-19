package expirable

import (
	"container/list"
	"sync"
	"time"
)

type EvictCallback[K comparable, V any] func(key K, value V)

type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

type LRU[K comparable, V any] struct {
	mu      sync.Mutex
	size    int
	ttl     time.Duration
	onEvict EvictCallback[K, V]
	list    *list.List
	items   map[K]*list.Element
}

func NewLRU[K comparable, V any](size int, onEvict EvictCallback[K, V], ttl time.Duration) *LRU[K, V] {
	if size <= 0 {
		size = 1
	}

	return &LRU[K, V]{
		size:    size,
		ttl:     ttl,
		onEvict: onEvict,
		list:    list.New(),
		items:   make(map[K]*list.Element, size),
	}
}

func (l *LRU[K, V]) Add(key K, value V) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	l.removeExpiredLocked(now)

	if elem, ok := l.items[key]; ok {
		item := elem.Value.(*entry[K, V])
		item.value = value
		item.expiresAt = l.expiresAt(now)
		l.list.MoveToFront(elem)
		return false
	}

	elem := l.list.PushFront(&entry[K, V]{
		key:       key,
		value:     value,
		expiresAt: l.expiresAt(now),
	})
	l.items[key] = elem

	evicted := false
	if l.list.Len() > l.size {
		l.removeOldestLocked()
		evicted = true
	}
	return evicted
}

func (l *LRU[K, V]) Get(key K) (V, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	var zero V
	elem, ok := l.items[key]
	if !ok {
		return zero, false
	}

	item := elem.Value.(*entry[K, V])
	if l.expired(item, time.Now()) {
		l.removeElementLocked(elem)
		return zero, false
	}

	l.list.MoveToFront(elem)
	return item.value, true
}

func (l *LRU[K, V]) expiresAt(now time.Time) time.Time {
	if l.ttl <= 0 {
		return time.Time{}
	}
	return now.Add(l.ttl)
}

func (l *LRU[K, V]) expired(item *entry[K, V], now time.Time) bool {
	return !item.expiresAt.IsZero() && !item.expiresAt.After(now)
}

func (l *LRU[K, V]) removeExpiredLocked(now time.Time) {
	for elem := l.list.Back(); elem != nil; {
		prev := elem.Prev()
		item := elem.Value.(*entry[K, V])
		if l.expired(item, now) {
			l.removeElementLocked(elem)
		}
		elem = prev
	}
}

func (l *LRU[K, V]) removeOldestLocked() {
	if elem := l.list.Back(); elem != nil {
		l.removeElementLocked(elem)
	}
}

func (l *LRU[K, V]) removeElementLocked(elem *list.Element) {
	item := elem.Value.(*entry[K, V])
	delete(l.items, item.key)
	l.list.Remove(elem)
	if l.onEvict != nil {
		l.onEvict(item.key, item.value)
	}
}
