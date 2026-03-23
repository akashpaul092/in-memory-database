package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"my-project/api"
	"my-project/configs"
	"my-project/internal/bloom"
	"my-project/internal/lru"
	"my-project/internal/service"
	"my-project/internal/store"
	"my-project/internal/pubsub"
	"my-project/internal/ttl"
	"my-project/internal/wal"
	"my-project/pkg/logger"
)

func main() {
	cfg, err := configs.Load(os.Getenv("CONFIG_PATH"))
	if err != nil {
		panic(err)
	}

	// Init logger
	level := parseLogLevel(cfg.LogLevel)
	logger.Init(level)

	// Init components
	s := store.New()

	// Replay WAL before creating other components
	if err := wal.Replay(cfg.WAL.Path, s); err != nil {
		logger.Error("WAL replay failed", "error", err)
		panic(err)
	}

	// Bloom and LRU (repopulate LRU from store after replay)
	b := bloom.NewBloomFilter(cfg.Bloom.ExpectedKeys, cfg.Bloom.FalsePositiveRate)
	l := lru.New(cfg.Store.MaxKeys)
	for _, key := range s.Keys() {
		if value, ok := s.Get(key); ok {
			b.Add(key)
			l.Put(key, value)
		}
	}

	w, err := wal.New(cfg.WAL.Path)
	if err != nil {
		logger.Error("WAL init failed", "error", err)
		panic(err)
	}
	defer w.Close()

	svc := service.New(s, b, l, w)
	ps := pubsub.New()
	h := api.NewHandler(svc, ps)

	// TTL worker
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ttl.StartTTLWorker(ctx, s, cfg.Store.TTLInterval)

	// Router
	r := gin.Default()
	api.RegisterRoutes(r, h)

	// HTTP server with graceful shutdown
	srv := &http.Server{
		Addr:    ":" + cfg.Server.Port,
		Handler: r,
	}

	go func() {
		logger.Info("Starting server", "port", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", "error", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown error", "error", err)
	}

	logger.Info("Server stopped")
}

func parseLogLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
