package ratelimit

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"mycourse-io-be/internal/shared/xdgx"
)

const cliRateLimitDir = "mycourse"
const cliRateLimitFile = "cli_rate_limit.json"

type fileEntry struct {
	WindowStart int64 `json:"windowStart"`
	WindowSec   int64 `json:"windowSec"`
	Count       int   `json:"count"`
}

type filePayload struct {
	Entries map[string]fileEntry `json:"entries"`
}

// FileStore persists fixed-window counters to a JSON file (APPCLI one-shot processes).
type FileStore struct {
	path string
	mu   sync.Mutex
}

// NewFileStore returns a file-backed store at path.
func NewFileStore(path string) *FileStore {
	return &FileStore{path: path}
}

// DefaultCLIFileStore uses $XDG_CONFIG_HOME/mycourse/cli_rate_limit.json (override via MYCOURSE_CLI_RATE_LIMIT_PATH).
func DefaultCLIFileStore() (*FileStore, error) {
	path, err := cliRateLimitPath()
	if err != nil {
		return nil, err
	}
	return NewFileStore(path), nil
}

func cliRateLimitPath() (string, error) {
	if override := os.Getenv("MYCOURSE_CLI_RATE_LIMIT_PATH"); override != "" {
		return override, nil
	}
	base, err := xdgx.ConfigHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, cliRateLimitDir, cliRateLimitFile), nil
}

// Allow records one attempt for key and returns whether it is within attempts for the window.
func (s *FileStore) Allow(key string, windowSec, windowStart int64, attempts int) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	payload, err := s.readPayload()
	if err != nil {
		return false, err
	}
	if payload.Entries == nil {
		payload.Entries = make(map[string]fileEntry)
	}

	ent, ok := payload.Entries[key]
	if !ok || ent.WindowStart != windowStart || ent.WindowSec != windowSec {
		ent = fileEntry{WindowStart: windowStart, WindowSec: windowSec, Count: 1}
		payload.Entries[key] = ent
		if err := s.writePayload(payload); err != nil {
			return false, err
		}
		return true, nil
	}
	ent.Count++
	payload.Entries[key] = ent
	if err := s.writePayload(payload); err != nil {
		return false, err
	}
	return ent.Count <= attempts, nil
}

func (s *FileStore) readPayload() (filePayload, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return filePayload{Entries: make(map[string]fileEntry)}, nil
		}
		return filePayload{}, err
	}
	var payload filePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return filePayload{}, err
	}
	if payload.Entries == nil {
		payload.Entries = make(map[string]fileEntry)
	}
	return payload, nil
}

func (s *FileStore) writePayload(payload filePayload) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0o600)
}

// AllowCLI is a convenience helper using default CLI file path and minute-based windows.
func AllowCLI(key string, attempts, minutes int) (bool, error) {
	if attempts < 1 || minutes < 1 {
		return true, nil
	}
	store, err := DefaultCLIFileStore()
	if err != nil {
		return false, err
	}
	windowSec := int64(minutes) * 60
	now := time.Now().Unix()
	windowStart := WindowStart(now, windowSec)
	return store.Allow(key, windowSec, windowStart, attempts)
}
