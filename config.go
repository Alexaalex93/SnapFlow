package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const currentConfigVersion = 1

type AppConfig struct {
	Version          int    `json:"version"`
	ProLicenseKey    string `json:"pro_license_key,omitempty"`
	UpgradeEntrySeen bool   `json:"upgrade_entry_seen,omitempty"`
}

type ConfigStore struct {
	path string
}

func NewConfigStore() (*ConfigStore, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("query user config dir: %w", err)
	}
	appDir := filepath.Join(dir, ProductName)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return nil, fmt.Errorf("create app config dir: %w", err)
	}
	return &ConfigStore{path: filepath.Join(appDir, "config.json")}, nil
}

func (s *ConfigStore) Load() (*AppConfig, error) {
	cfg := defaultConfig()
	b, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	migrated, changed := migrateConfig(cfg)
	if changed {
		if err := s.Save(migrated); err != nil {
			return nil, err
		}
	}
	return migrated, nil
}

func (s *ConfigStore) Save(cfg *AppConfig) error {
	cfg.Version = currentConfigVersion
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(s.path, b, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func defaultConfig() *AppConfig {
	return &AppConfig{Version: currentConfigVersion}
}

func migrateConfig(cfg *AppConfig) (*AppConfig, bool) {
	if cfg == nil {
		return defaultConfig(), true
	}
	changed := false
	if cfg.Version <= 0 {
		cfg.Version = 1
		changed = true
	}
	if cfg.Version != currentConfigVersion {
		cfg.Version = currentConfigVersion
		changed = true
	}
	return cfg, changed
}
