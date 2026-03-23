package lru

import (
	"sync"
)

type node struct {
	key   string
	value string
	prev  *node
	next  *node
}

// LRU is a thread-safe LRU cache with O(1) get and put.
type LRU struct {
	mu       sync.RWMutex
	capacity int
	size     int
	cache    map[string]*node
	head     *node
	tail     *node
}

// New creates a new LRU with the given capacity.
func New(capacity int) *LRU {
	if capacity < 1 {
		capacity = 1
	}
	l := &LRU{
		capacity: capacity,
		cache:    make(map[string]*node),
		head:     &node{},
		tail:     &node{},
	}
	l.head.next = l.tail
	l.tail.prev = l.head
	return l
}

// Get returns the value for key and true if found, moving it to front. Otherwise ("", false).
func (l *LRU) Get(key string) (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	n, ok := l.cache[key]
	if !ok {
		return "", false
	}
	l.moveToFront(n)
	return n.value, true
}

// Put adds or updates a key-value pair, moving it to front.
// Returns the evicted key (if any) so the caller can sync other data structures.
func (l *LRU) Put(key, value string) (evictedKey string, evicted bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if n, ok := l.cache[key]; ok {
		n.value = value
		l.moveToFront(n)
		return "", false
	}

	n := &node{key: key, value: value}
	l.cache[key] = n
	l.addToFront(n)
	l.size++

	if l.size > l.capacity {
		tail := l.removeTail()
		delete(l.cache, tail.key)
		l.size--
		return tail.key, true
	}
	return "", false
}

// Evict removes and returns the least recently used key, or ("", false) if empty.
func (l *LRU) Evict() (string, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.size == 0 {
		return "", false
	}
	n := l.removeTail()
	delete(l.cache, n.key)
	l.size--
	return n.key, true
}

// Delete removes a key from the LRU.
func (l *LRU) Delete(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	n, ok := l.cache[key]
	if !ok {
		return
	}
	l.removeNode(n)
	delete(l.cache, key)
	l.size--
}

// Size returns the number of items in the LRU.
func (l *LRU) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.size
}

func (l *LRU) moveToFront(n *node) {
	l.removeNode(n)
	l.addToFront(n)
}

func (l *LRU) addToFront(n *node) {
	n.prev = l.head
	n.next = l.head.next
	l.head.next.prev = n
	l.head.next = n
}

func (l *LRU) removeNode(n *node) {
	n.prev.next = n.next
	n.next.prev = n.prev
}

func (l *LRU) removeTail() *node {
	n := l.tail.prev
	if n == l.head {
		return nil
	}
	l.removeNode(n)
	return n
}
