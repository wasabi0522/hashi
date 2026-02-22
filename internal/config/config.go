package config

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/rawbytes"
	"github.com/knadh/koanf/v2"
)

// Config represents the hashi configuration.
type Config struct {
	WorktreeDir string `koanf:"worktree_dir"`
	Hooks       Hooks  `koanf:"hooks"`
}

// Hooks defines lifecycle hooks.
type Hooks struct {
	CopyFiles []string `koanf:"copy_files"`
	PostNew   []string `koanf:"post_new"`
}

// Load reads configuration from the given YAML file path and environment variables.
// Missing file is not an error; defaults are used.
// Priority: environment variables > file > defaults.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	// 1. Defaults — confmap.Provider wraps an in-memory map and never fails.
	_ = k.Load(confmap.Provider(map[string]any{
		"worktree_dir": ".worktrees",
	}, "."), nil)

	// 2. YAML file (overrides defaults)
	if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("loading config %s: %w", path, err)
		}
	}

	// 3. Environment variables (highest priority)
	if err := k.Load(env.Provider("HASHI_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "HASHI_"))
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env config: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadFromReader reads configuration from an io.Reader containing YAML.
// Environment variables are not applied. Useful for testing.
func LoadFromReader(r io.Reader) (*Config, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	k := koanf.New(".")
	_ = k.Load(confmap.Provider(map[string]any{
		"worktree_dir": ".worktrees",
	}, "."), nil)

	if err := k.Load(rawbytes.Provider(data), yaml.Parser()); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if filepath.IsAbs(c.WorktreeDir) {
		return fmt.Errorf("worktree_dir must be a relative path: %s", c.WorktreeDir)
	}
	if strings.Contains(c.WorktreeDir, "..") {
		return fmt.Errorf("worktree_dir must not contain '..': %s", c.WorktreeDir)
	}
	if c.WorktreeDir == "." {
		return fmt.Errorf("worktree_dir must not be '.': worktrees would be created directly in the repository root")
	}
	return nil
}
