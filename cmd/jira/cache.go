package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"gbm/internal/jira"
	"os"
	"path/filepath"
)

// fileCacheStore persists IssuesCache + user to a JSON file under the user's
// cache dir. It satisfies jira.CacheStore.
type fileCacheStore struct {
	path string
}

type cachePayload struct {
	Cache *jira.IssuesCache `json:"cache"`
	User  string            `json:"user"`
}

func newFileCacheStore() (*fileCacheStore, error) {
	dir, err := cacheDir()
	if err != nil {
		return nil, err
	}
	return &fileCacheStore{path: filepath.Join(dir, "cache.json")}, nil
}

// cacheDir returns $XDG_CACHE_HOME/gbm-jira or $HOME/.cache/gbm-jira.
func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine cache dir: %w", err)
	}
	dir := filepath.Join(base, "gbm-jira")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create cache dir %s: %w", dir, err)
	}
	return dir, nil
}

func (s *fileCacheStore) Load() (*jira.IssuesCache, string, error) {
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("read cache: %w", err)
	}
	var payload cachePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, "", fmt.Errorf("parse cache: %w", err)
	}
	return payload.Cache, payload.User, nil
}

func (s *fileCacheStore) Save(cache *jira.IssuesCache, user string) error {
	data, err := json.MarshalIndent(cachePayload{Cache: cache, User: user}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode cache: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write cache: %w", err)
	}
	return nil
}
