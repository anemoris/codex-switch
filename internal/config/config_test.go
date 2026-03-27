package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateNilProfiles(t *testing.T) {
	cfg := Config{Version: 1, Profiles: nil}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for nil profiles")
	}
}

func TestValidateDefaultProfileNotExist(t *testing.T) {
	cfg := Config{
		Version:        1,
		DefaultProfile: "missing",
		Profiles:       map[string]Profile{},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for missing default profile")
	}
}

func TestValidateEmptyCodexHome(t *testing.T) {
	cfg := Config{
		Version: 1,
		Profiles: map[string]Profile{
			"work": {CodexHome: ""},
		},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for empty codex_home")
	}
}

func TestValidateRelativeCodexHome(t *testing.T) {
	cfg := Config{
		Version: 1,
		Profiles: map[string]Profile{
			"work": {CodexHome: "relative/path"},
		},
	}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected error for relative codex_home")
	}
}

func TestValidateDuplicateShortcut(t *testing.T) {
	cfg := Config{
		Version: 1,
		Profiles: map[string]Profile{
			"alpha": {CodexHome: "/a", Shortcut: "same"},
			"beta":  {CodexHome: "/b", Shortcut: "same"},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate shortcut")
	}
	// Due to sorted iteration, error should always mention alpha first
	want := `shortcut "same" is used by both "alpha" and "beta"`
	if err.Error() != want {
		t.Fatalf("unexpected error: %q (want %q)", err.Error(), want)
	}
}

func TestValidateDuplicateCodexHome(t *testing.T) {
	cfg := Config{
		Version: 1,
		Profiles: map[string]Profile{
			"alpha": {CodexHome: "/same/path"},
			"beta":  {CodexHome: "/same/path/"},
		},
	}
	err := Validate(cfg)
	if err == nil {
		t.Fatal("expected error for duplicate codex_home")
	}
	want := `codex_home "/same/path" is used by both "alpha" and "beta"`
	if err.Error() != want {
		t.Fatalf("unexpected error: %q (want %q)", err.Error(), want)
	}
}

func TestValidateValidConfig(t *testing.T) {
	cfg := Config{
		Version:        1,
		DefaultProfile: "work",
		Profiles: map[string]Profile{
			"work":     {CodexHome: "/home/user/.codex-switch/profiles/work", Shortcut: "cwork"},
			"personal": {CodexHome: "/home/user/.codex-switch/profiles/personal"},
		},
	}
	if err := Validate(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProfileName(t *testing.T) {
	valid := []string{"work", "Work", "work-1", "work.prod", "w_1"}
	for _, name := range valid {
		if err := ValidateProfileName(name); err != nil {
			t.Errorf("expected %q to be valid: %v", name, err)
		}
	}
	invalid := []string{"", "-start", ".start", "has space", "special!"}
	for _, name := range invalid {
		if err := ValidateProfileName(name); err == nil {
			t.Errorf("expected %q to be invalid", name)
		}
	}
}

func TestValidateShortcut(t *testing.T) {
	valid := []string{"cwork", "c-work", "c_work", "Cw1"}
	for _, s := range valid {
		if err := ValidateShortcut(s); err != nil {
			t.Errorf("expected %q to be valid: %v", s, err)
		}
	}
	invalid := []string{"", "1start", "-start", "codex", "codex-switch"}
	for _, s := range invalid {
		if err := ValidateShortcut(s); err == nil {
			t.Errorf("expected %q to be invalid", s)
		}
	}
}

func TestExpandPath(t *testing.T) {
	// Empty string
	result, err := ExpandPath("")
	if err != nil || result != "" {
		t.Fatalf("expected empty result for empty input, got %q (err=%v)", result, err)
	}

	// Absolute path
	result, err = ExpandPath("/absolute/path")
	if err != nil || result != "/absolute/path" {
		t.Fatalf("expected /absolute/path, got %q (err=%v)", result, err)
	}

	// Tilde
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	result, err = ExpandPath("~")
	if err != nil || result != home {
		t.Fatalf("expected %q for ~, got %q (err=%v)", home, result, err)
	}

	// Tilde with subpath
	result, err = ExpandPath("~/subdir")
	if err != nil || result != filepath.Join(home, "subdir") {
		t.Fatalf("expected %q, got %q (err=%v)", filepath.Join(home, "subdir"), result, err)
	}
}

func TestStoreLoadSaveRoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "config.json"))

	// Loading non-existent file returns default config
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 1 || len(cfg.Profiles) != 0 {
		t.Fatalf("unexpected default config: %+v", cfg)
	}

	// Save and reload
	cfg.DefaultProfile = "work"
	cfg.Profiles["work"] = Profile{
		CodexHome: "/home/user/work",
		Shortcut:  "cwork",
		Label:     "Work Account",
	}
	if err := store.Save(cfg); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DefaultProfile != "work" {
		t.Fatalf("expected default profile 'work', got %q", loaded.DefaultProfile)
	}
	p, ok := loaded.Profiles["work"]
	if !ok {
		t.Fatal("profile 'work' not found after reload")
	}
	if p.CodexHome != "/home/user/work" || p.Shortcut != "cwork" || p.Label != "Work Account" {
		t.Fatalf("unexpected profile: %+v", p)
	}
}

func TestStoreSaveCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "config.json")
	store := NewStore(path)

	cfg := Default()
	cfg.Profiles["test"] = Profile{CodexHome: "/tmp/test"}
	if err := store.Save(cfg); err != nil {
		t.Fatalf("save with nested dirs: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file should exist: %v", err)
	}
	if _, err := os.Stat(RootMarkerPath(filepath.Dir(path))); err != nil {
		t.Fatalf("root marker should exist: %v", err)
	}
}

func TestStoreSaveRejectsInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(filepath.Join(dir, "config.json"))

	cfg := Config{Version: 1, Profiles: nil}
	if err := store.Save(cfg); err == nil {
		t.Fatal("expected save to refuse invalid config")
	}
}

func TestStoreSaveDoesNotCommitConfigWhenMarkerWriteFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(RootMarkerPath(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	store := NewStore(filepath.Join(dir, "config.json"))

	cfg := Default()
	cfg.Profiles["test"] = Profile{CodexHome: "/tmp/test"}
	if err := store.Save(cfg); err == nil {
		t.Fatal("expected save to fail when root marker path is blocked")
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); !os.IsNotExist(err) {
		t.Fatalf("config file should not be committed on marker write failure, stat err=%v", err)
	}
}

func TestStoreSaveRollsBackNewMarkerWhenConfigCommitFails(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")
	if err := os.Mkdir(configPath, 0o755); err != nil {
		t.Fatal(err)
	}
	store := NewStore(configPath)

	cfg := Default()
	cfg.Profiles["test"] = Profile{CodexHome: "/tmp/test"}
	if err := store.Save(cfg); err == nil {
		t.Fatal("expected save to fail when config path is a directory")
	}
	if _, err := os.Stat(RootMarkerPath(dir)); !os.IsNotExist(err) {
		t.Fatalf("new marker should be rolled back on config commit failure, stat err=%v", err)
	}
}

func TestStoreSaveKeepsExistingMarkerWhenConfigCommitFails(t *testing.T) {
	dir := t.TempDir()
	if err := EnsureRootMarker(dir); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(dir, "config.json")
	if err := os.Mkdir(configPath, 0o755); err != nil {
		t.Fatal(err)
	}
	store := NewStore(configPath)

	cfg := Default()
	cfg.Profiles["test"] = Profile{CodexHome: "/tmp/test"}
	if err := store.Save(cfg); err == nil {
		t.Fatal("expected save to fail when config path is a directory")
	}
	if _, err := os.Stat(RootMarkerPath(dir)); err != nil {
		t.Fatalf("existing marker should be preserved on config commit failure: %v", err)
	}
}
