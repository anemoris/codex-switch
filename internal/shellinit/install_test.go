package shellinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".zshrc")
	block, err := ManagedBlock("zsh")
	if err != nil {
		t.Fatal(err)
	}

	changed, err := Install(rcPath, block)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected initial install to change file")
	}

	changed, err = Install(rcPath, block)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected second install to be idempotent")
	}
}

func TestInstallReplacesExistingManagedBlock(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".zshrc")
	initial := strings.Join([]string{
		"export PATH=$PATH:/tmp/bin",
		StartMarker,
		"eval \"$(old aliases --shell zsh)\"",
		EndMarker,
		"",
	}, "\n")
	if err := os.WriteFile(rcPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	block, err := ManagedBlock("zsh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(rcPath, block); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Count(text, StartMarker) != 1 {
		t.Fatalf("expected one managed block, got: %s", text)
	}
	if !strings.Contains(text, `eval "$('codex-switch' aliases --shell zsh)"`) {
		t.Fatalf("expected updated block, got: %s", text)
	}
}

func TestUninstallRemovesManagedBlockOnly(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".zshrc")
	initial := strings.Join([]string{
		"export PATH=$PATH:/tmp/bin",
		"",
		StartMarker,
		`eval "$(codex-switch aliases --shell zsh)"`,
		EndMarker,
		"",
		"alias gs='git status'",
		"",
	}, "\n")
	if err := os.WriteFile(rcPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := Uninstall(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected uninstall to change file")
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if strings.Contains(text, StartMarker) || strings.Contains(text, EndMarker) {
		t.Fatalf("managed block was not removed: %s", text)
	}
	if !strings.Contains(text, "export PATH=$PATH:/tmp/bin") || !strings.Contains(text, "alias gs='git status'") {
		t.Fatalf("user content should remain: %s", text)
	}
}

func TestUninstallKeepsAdjacentUserLinesSeparated(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".zshrc")
	initial := strings.Join([]string{
		"export PATH=$PATH:/tmp/bin",
		StartMarker,
		`eval "$(codex-switch aliases --shell zsh)"`,
		EndMarker,
		"alias gs='git status'",
		"",
	}, "\n")
	if err := os.WriteFile(rcPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Uninstall(rcPath); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "export PATH=$PATH:/tmp/bin\nalias gs='git status'\n"; got != want {
		t.Fatalf("unexpected file contents after uninstall:\nwant: %q\ngot:  %q", want, got)
	}
}

func TestUninstallIsIdempotent(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(rcPath, []byte("export PATH=$PATH:/tmp/bin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	changed, err := Uninstall(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("expected uninstall without managed block to be unchanged")
	}
}

func TestHasManagedBlock(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".zshrc")
	block, err := ManagedBlock("zsh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rcPath, []byte(block), 0o644); err != nil {
		t.Fatal(err)
	}

	ok, err := HasManagedBlock(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected managed block to be detected")
	}
}

func TestInstallAndUninstallPreserveExistingFilePermissions(t *testing.T) {
	dir := t.TempDir()
	rcPath := filepath.Join(dir, ".zshrc")
	if err := os.WriteFile(rcPath, []byte("export PATH=$PATH:/tmp/bin\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	block, err := ManagedBlock("zsh")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(rcPath, block); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected install to preserve mode 0600, got %#o", got)
	}

	if _, err := Uninstall(rcPath); err != nil {
		t.Fatal(err)
	}

	info, err = os.Stat(rcPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected uninstall to preserve mode 0600, got %#o", got)
	}
}
