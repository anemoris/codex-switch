package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const RootMarkerName = ".codex-switch-root"

type Store struct {
	path string
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func NewDefaultStore() (*Store, error) {
	path, err := DefaultStorePath()
	if err != nil {
		return nil, err
	}
	return NewStore(path), nil
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Load() (Config, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, err
	}

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Profiles == nil {
		cfg.Profiles = map[string]Profile{}
	}
	if err := Validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (s *Store) Save(cfg Config) error {
	if err := Validate(cfg); err != nil {
		return err
	}
	root := filepath.Dir(s.path)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	markerCreated, err := ensureRootMarker(root)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		if markerCreated {
			_ = os.Remove(RootMarkerPath(root))
		}
		return err
	}
	data = append(data, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		if markerCreated {
			_ = os.Remove(RootMarkerPath(root))
		}
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		_ = os.Remove(tmp)
		if markerCreated {
			_ = os.Remove(RootMarkerPath(root))
		}
		return err
	}
	return nil
}

func DefaultStorePath() (string, error) {
	root, err := RootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "config.json"), nil
}

func RootDir() (string, error) {
	if root := os.Getenv("CODEX_SWITCH_HOME"); root != "" {
		return filepath.Abs(root)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex-switch"), nil
}

func RootMarkerPath(root string) string {
	return filepath.Join(root, RootMarkerName)
}

func EnsureRootMarker(root string) error {
	return os.WriteFile(RootMarkerPath(root), []byte("codex-switch\n"), 0o644)
}

func IsLegacyConfigFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false, nil
	}
	if _, ok := raw["profiles"]; !ok {
		return false, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return false, nil
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Profiles == nil {
		return false, nil
	}
	if err := Validate(cfg); err != nil {
		return false, nil
	}
	return true, nil
}

func ensureRootMarker(root string) (bool, error) {
	path := RootMarkerPath(root)
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return false, os.ErrExist
		}
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, err
	}
	if err := EnsureRootMarker(root); err != nil {
		return false, err
	}
	return true, nil
}

func DefaultProfileHome(name string) (string, error) {
	root, err := RootDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "profiles", name), nil
}

func ExpandPath(input string) (string, error) {
	if input == "" {
		return "", nil
	}
	if input == "~" || (len(input) >= 2 && input[:2] == "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if input == "~" {
			input = home
		} else {
			input = filepath.Join(home, input[2:])
		}
	}
	return filepath.Abs(input)
}
