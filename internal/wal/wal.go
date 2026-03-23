package wal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"my-project/internal/store"
)

const (
	opSet = "SET"
	opDel = "DEL"
)

// Entry represents a single WAL log entry.
type Entry struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// WAL is a Write-Ahead Log for persistence.
type WAL struct {
	mu     sync.Mutex
	file   *os.File
	path   string
	writer *bufio.Writer
}

// New creates or opens a WAL file at the given path.
func New(path string) (*WAL, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create wal dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("open wal file: %w", err)
	}
	return &WAL{
		file:   f,
		path:   path,
		writer: bufio.NewWriter(f),
	}, nil
}

// Append writes an operation to the log.
func (w *WAL) Append(op, key, value string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	entry := Entry{Op: op, Key: key, Value: value}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	if _, err := w.writer.Write(append(data, '\n')); err != nil {
		return err
	}
	return w.writer.Flush()
}

// Sync flushes the file to disk.
func (w *WAL) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Sync()
}

// Close closes the WAL file.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if err := w.writer.Flush(); err != nil {
		return err
	}
	return w.file.Close()
}

// Replay replays the WAL file into the store.
func Replay(path string, s *store.Store) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no WAL yet
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return fmt.Errorf("parse wal entry %q: %w", line, err)
		}
		switch entry.Op {
		case opSet:
			s.SetWithTTL(entry.Key, entry.Value, 0)
		case opDel:
			s.Delete(entry.Key)
		default:
			return fmt.Errorf("unknown wal op: %s", entry.Op)
		}
	}
	return scanner.Err()
}
