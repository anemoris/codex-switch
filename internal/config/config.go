package config

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
)

type Config struct {
	Version        int                `json:"version"`
	DefaultProfile string             `json:"default_profile,omitempty"`
	Profiles       map[string]Profile `json:"profiles"`
}

type Profile struct {
	Label     string `json:"label,omitempty"`
	CodexHome string `json:"codex_home"`
	Shortcut  string `json:"shortcut,omitempty"`
}

func Default() Config {
	return Config{
		Version:  1,
		Profiles: map[string]Profile{},
	}
}

var profileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
var shortcutPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

func Validate(cfg Config) error {
	if cfg.Profiles == nil {
		return fmt.Errorf("profiles cannot be null")
	}

	if cfg.DefaultProfile != "" {
		if _, ok := cfg.Profiles[cfg.DefaultProfile]; !ok {
			return fmt.Errorf("default profile %q does not exist", cfg.DefaultProfile)
		}
	}

	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)

	seenShortcuts := map[string]string{}
	seenHomes := map[string]string{}
	for _, name := range names {
		profile := cfg.Profiles[name]
		if err := ValidateProfileName(name); err != nil {
			return err
		}
		if profile.CodexHome == "" {
			return fmt.Errorf("profile %q has empty codex_home", name)
		}
		if !filepath.IsAbs(profile.CodexHome) {
			return fmt.Errorf("profile %q codex_home must be absolute", name)
		}
		cleanHome := filepath.Clean(profile.CodexHome)
		if owner, ok := seenHomes[cleanHome]; ok {
			return fmt.Errorf("codex_home %q is used by both %q and %q", cleanHome, owner, name)
		}
		seenHomes[cleanHome] = name
		if profile.Shortcut != "" {
			if err := ValidateShortcut(profile.Shortcut); err != nil {
				return fmt.Errorf("profile %q: %w", name, err)
			}
			if owner, ok := seenShortcuts[profile.Shortcut]; ok {
				return fmt.Errorf("shortcut %q is used by both %q and %q", profile.Shortcut, owner, name)
			}
			seenShortcuts[profile.Shortcut] = name
		}
	}

	return nil
}

func ValidateProfileName(name string) error {
	if !profileNamePattern.MatchString(name) {
		return fmt.Errorf("invalid profile name %q", name)
	}
	return nil
}

func ValidateShortcut(shortcut string) error {
	if !shortcutPattern.MatchString(shortcut) {
		return fmt.Errorf("invalid shortcut %q", shortcut)
	}
	if shortcut == "codex" || shortcut == "codex-switch" {
		return fmt.Errorf("shortcut %q is reserved", shortcut)
	}
	return nil
}
