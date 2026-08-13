package userconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/BurntSushi/toml"
	"github.com/brandonyoungdev/tldx/internal/config"
)

const ConfigFileName = "config.toml"

// LegacyConfigFileName is the original, presets-only file name. Still read and
// written back to when it exists.
const LegacyConfigFileName = "presets.toml"

type UserConfig struct {
	Defaults Defaults               `toml:"defaults"`
	Presets  map[string]PresetEntry `toml:"presets"`
}

// Defaults are applied to every run unless the matching flag is passed on the
// command line. A zero value means "not set".
type Defaults struct {
	TLDs      []string `toml:"tlds,omitempty"`
	TLDPreset string   `toml:"tld_preset,omitempty"`
	Prefixes  []string `toml:"prefixes,omitempty"`
	Suffixes  []string `toml:"suffixes,omitempty"`
	Format    string   `toml:"format,omitempty"`
	// Pointers because omitempty alone does not drop zero ints on save.
	MaxDomainLength *int `toml:"max_domain_length,omitempty"`
	Limit           *int `toml:"limit,omitempty"`
	OnlyAvailable   bool `toml:"only_available,omitempty"`
	ForSale         bool `toml:"for_sale,omitempty"`
	OnlyForSale     bool `toml:"only_for_sale,omitempty"`
	ShowStats       bool `toml:"show_stats,omitempty"`
	NoColor         bool `toml:"no_color,omitempty"`
	Verbose         bool `toml:"verbose,omitempty"`
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

	preferred := filepath.Join(dir, "tldx", ConfigFileName)
	if _, err := os.Stat(preferred); err == nil {
		return preferred, nil
	}

	legacy := filepath.Join(dir, "tldx", LegacyConfigFileName)
	if _, err := os.Stat(legacy); err == nil {
		return legacy, nil
	}

	return preferred, nil
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

// ApplyTo copies configured defaults onto cfg. isSet reports whether a flag was
// passed on the command line; those win over the config file.
func (d Defaults) ApplyTo(cfg *config.TldxConfigOptions, isSet func(flag string) bool) {
	if isSet == nil {
		isSet = func(string) bool { return false }
	}

	// Either flag replaces both configured values rather than merging.
	if !isSet("tlds") && !isSet("tld-preset") {
		if len(d.TLDs) > 0 {
			cfg.TLDs = slices.Clone(d.TLDs)
		}
		if d.TLDPreset != "" {
			cfg.TLDPreset = d.TLDPreset
		}
	}

	if !isSet("prefixes") && len(d.Prefixes) > 0 {
		cfg.Prefixes = slices.Clone(d.Prefixes)
	}
	if !isSet("suffixes") && len(d.Suffixes) > 0 {
		cfg.Suffixes = slices.Clone(d.Suffixes)
	}
	if !isSet("max-domain-length") && d.MaxDomainLength != nil {
		cfg.MaxDomainLength = *d.MaxDomainLength
	}
	if !isSet("format") && d.Format != "" {
		cfg.OutputFormat = d.Format
	}
	if !isSet("limit") && d.Limit != nil {
		cfg.Limit = *d.Limit
	}
	if !isSet("only-available") && d.OnlyAvailable {
		cfg.OnlyAvailable = true
	}
	if !isSet("for-sale") && d.ForSale {
		cfg.CheckForSale = true
	}
	// only_for_sale implies for_sale; applied in the root command's PreRunE.
	if !isSet("only-for-sale") && d.OnlyForSale {
		cfg.OnlyForSale = true
	}
	if !isSet("show-stats") && d.ShowStats {
		cfg.ShowStats = true
	}
	if !isSet("no-color") && d.NoColor {
		cfg.NoColor = true
	}
	if !isSet("verbose") && d.Verbose {
		cfg.Verbose = true
	}
}
