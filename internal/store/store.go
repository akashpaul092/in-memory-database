package store

import (
	"sync"
	"time"
)

// Store is a thread-safe in-memory key-value store.
type Store struct {
	mu   sync.RWMutex
	data map[string]Item
}

// New creates a new Store.
func New() *Store {
	return &Store{
		data: make(map[string]Item),
	}
}

// Set stores a key-value pair with no expiry.
func (s *Store) Set(key, value string) {
	s.SetWithTTL(key, value, 0)
}

// SetWithTTL stores a key-value pair with optional TTL.
// If ttlSeconds is 0, the key never expires.
func (s *Store) SetWithTTL(key, value string, ttlSeconds int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	expiry := int64(0)
	if ttlSeconds > 0 {
		expiry = time.Now().Unix() + ttlSeconds
	}

	s.data[key] = Item{
		Value:  value,
		Expiry: expiry,
	}
}

// Get retrieves a value by key. Returns (value, true) if found and not expired,
// ( "", false) otherwise.
func (s *Store) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.data[key]
	if !ok {
		return "", false
	}
	if item.Expiry > 0 && time.Now().Unix() >= item.Expiry {
		return "", false
	}
	return item.Value, true
}

// Delete removes a key from the store.
func (s *Store) Delete(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, key)
}

// Keys returns a copy of all keys in the store (for TTL scanner and WAL replay).
func (s *Store) Keys() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}
	return keys
}

// GetExpiry returns the expiry timestamp for a key, or 0 if not found.
func (s *Store) GetExpiry(key string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.data[key]
	if !ok {
		return 0
	}
	return item.Expiry
}
