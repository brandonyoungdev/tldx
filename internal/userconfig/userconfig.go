package userconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type UserConfig struct {
	Presets map[string]PresetEntry `toml:"presets"`
}

type PresetEntry struct {
	TLDs []string `toml:"tlds"`
}

func ConfigPath() (string, error) {
	if override := os.Getenv("TLDX_CONFIG"); override != "" {
		return override, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("userconfig: cannot determine config directory: %w", err)
	}
	return filepath.Join(dir, "tldx", "presets.toml"), nil
}

func Load() (*UserConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	cfg := &UserConfig{Presets: make(map[string]PresetEntry)}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("userconfig: read %s: %w", path, err)
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("userconfig: parse %s: %w", path, err)
	}

	if cfg.Presets == nil {
		cfg.Presets = make(map[string]PresetEntry)
	}

	return cfg, nil
}

func Save(cfg *UserConfig) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("userconfig: create config dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("userconfig: write %s: %w", path, err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("userconfig: encode %s: %w", path, err)
	}

	return nil
}
