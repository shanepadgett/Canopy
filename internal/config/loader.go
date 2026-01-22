package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/shanepadgett/canopy/internal/core"
)

// Load reads site.json from the given directory and returns a Config.
// If path is empty, it searches upward from cwd for site.json.
func Load(path string) (core.Config, error) {
	cfg := core.DefaultConfig()
	cfg.Search.Enabled = true

	if path == "" {
		var err error
		path, err = findConfig()
		if err != nil {
			return cfg, err
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing config: %w", err)
	}

	// Validate required fields
	if cfg.Name == "" {
		return cfg, errors.New("config: name is required")
	}
	if cfg.BaseURL == "" {
		return cfg, errors.New("config: baseURL is required")
	}

	// Apply defaults for empty fields
	if cfg.Title == "" {
		cfg.Title = cfg.Name
	}
	if cfg.Permalinks == nil {
		cfg.Permalinks = make(map[string]string)
	}
	if cfg.Sections == nil {
		cfg.Sections = make(map[string]core.SectionConfig)
	}
	if cfg.Params == nil {
		cfg.Params = make(map[string]any)
	}

	return cfg, nil
}

// Find searches upward from cwd for site.json and returns its path.
func Find() (string, error) {
	return findConfig()
}

// findConfig searches upward from cwd for site.json.
func findConfig() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, "site.json")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", errors.New("site.json not found (searched upward from cwd)")
}

// RootDir returns the directory containing site.json.
func RootDir(configPath string) string {
	return filepath.Dir(configPath)
}
