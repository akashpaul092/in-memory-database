package service

import (
	"my-project/internal/bloom"
	"my-project/internal/lru"
	"my-project/internal/store"
	"my-project/internal/wal"
)

// Service is the business logic layer that connects all modules.
type Service struct {
	store *store.Store
	bloom *bloom.Filter
	lru   *lru.LRU
	wal   *wal.WAL
}

// New creates a new Service. WAL can be nil to disable persistence.
func New(s *store.Store, b *bloom.Filter, l *lru.LRU, w *wal.WAL) *Service {
	return &Service{
		store: s,
		bloom: b,
		lru:   l,
		wal:   w,
	}
}

// Set stores a key-value pair. If ttl is non-nil and > 0, the key will expire.
func (svc *Service) Set(key, value string, ttl *int64) {
	ttlSec := int64(0)
	if ttl != nil && *ttl > 0 {
		ttlSec = *ttl
	}

	if svc.lru != nil {
		evictedKey, evicted := svc.lru.Put(key, value)
		if evicted {
			svc.store.Delete(evictedKey)
			if svc.wal != nil {
				_ = svc.wal.Append("DEL", evictedKey, "")
			}
		}
	}
	svc.store.SetWithTTL(key, value, ttlSec)
	if svc.bloom != nil {
		svc.bloom.Add(key)
	}
	if svc.wal != nil {
		_ = svc.wal.Append("SET", key, value)
	}
}

// Get retrieves a value by key.
// Uses Bloom filter to avoid store lookup when key definitely does not exist.
func (svc *Service) Get(key string) (string, bool) {
	if svc.bloom != nil && !svc.bloom.Contains(key) {
		return "", false
	}
	value, ok := svc.store.Get(key)
	if ok && svc.lru != nil {
		svc.lru.Get(key) // touch to update LRU order
	}
	return value, ok
}

// Delete removes a key.
func (svc *Service) Delete(key string) {
	svc.store.Delete(key)
	if svc.lru != nil {
		svc.lru.Delete(key)
	}
	if svc.wal != nil {
		_ = svc.wal.Append("DEL", key, "")
	}
	// Bloom stays as-is (acceptable; may have rare false positives)
}
