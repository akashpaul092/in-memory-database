package ttl

import (
	"context"
	"time"

	"my-project/internal/store"
	"my-project/pkg/logger"
)

// StartTTLWorker starts a background goroutine that periodically cleans expired keys.
// It runs until ctx is cancelled.
func StartTTLWorker(ctx context.Context, s *store.Store, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Info("TTL worker stopped")
				return
			case <-ticker.C:
				cleanExpired(s)
			}
		}
	}()
}

func cleanExpired(s *store.Store) {
	now := time.Now().Unix()
	keys := s.Keys()
	deleted := 0

	for _, key := range keys {
		expiry := s.GetExpiry(key)
		if expiry > 0 && now >= expiry {
			s.Delete(key)
			deleted++
		}
	}

	if deleted > 0 {
		logger.Debug("TTL worker deleted expired keys", "count", deleted)
	}
}
