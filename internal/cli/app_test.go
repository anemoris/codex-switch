package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"codex-switch/internal/config"
	"codex-switch/internal/profile"
	"codex-switch/internal/shellinit"
	"codex-switch/internal/testutil"
)

type recordingRunner struct {
	last *exec.Cmd
	err  error
	run  func(cmd *exec.Cmd) error
}

func (r *recordingRunner) Run(cmd *exec.Cmd) error {
	r.last = cmd
	if r.run != nil {
		return r.run(cmd)
	}
	return r.err
}

func newTestApp(t *testing.T, cmdRunner *recordingRunner) (*App, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CODEX_SWITCH_HOME", root)
	prependFakeCodexToPath(t)
	store, err := config.NewDefaultStore()
	if err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	app := New(store, cmdRunner, "test", strings.NewReader(""), &stdout, &stderr)
	return app, root
}

func prependFakeCodexToPath(t *testing.T) {
	t.Helper()
	binDir := t.TempDir()
	codexPath := filepath.Join(binDir, "codex")
	if err := os.WriteFile(codexPath, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func loadTestConfig(t *testing.T, root string) config.Config {
	t.Helper()
	store := config.NewStore(filepath.Join(root, "config.json"))
	cfg, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func addProfileForTest(t *testing.T, app *App, args ...string) {
	t.Helper()
	runArgs := append([]string{"add"}, args...)
	if err := app.Run(runArgs); err != nil {
		t.Fatalf("add profile %v: %v", args, err)
	}
}

func initShellForTest(t *testing.T, app *App, shell string) string {
	t.Helper()
	rcPath := filepath.Join(t.TempDir(), "."+shell+"rc")
	if shell == "fish" {
		rcPath = filepath.Join(t.TempDir(), "config.fish")
	}
	if err := app.Run([]string{"init-shell", "--shell", shell, "--rc-file", rcPath}); err != nil {
		t.Fatalf("init-shell: %v", err)
	}
	return rcPath
}

func readFileForTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

func TestAddListAndShowProfile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)

	stdout := app.stdout.(*bytes.Buffer)
	if err := app.Run([]string{"add", "work", "--shortcut", "cwork", "--default"}); err != nil {
		t.Fatalf("add profile: %v", err)
	}

	cfg := loadTestConfig(t, root)
	if cfg.DefaultProfile != "work" {
		t.Fatalf("unexpected default profile: %q", cfg.DefaultProfile)
	}

	stdout.Reset()
	if err := app.Run([]string{"list"}); err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "work [default, shortcut=cwork, auth=missing]") {
		t.Fatalf("unexpected list output: %s", out)
	}
	if !strings.Contains(out, "USAGE_SNAPSHOT=unavailable") {
		t.Fatalf("expected missing usage snapshot in list output: %s", out)
	}

	stdout.Reset()
	if err := app.Run([]string{"show", "work"}); err != nil {
		t.Fatalf("show profile: %v", err)
	}
	if out := stdout.String(); !strings.Contains(out, "auth_present=false") {
		t.Fatalf("unexpected show output: %s", out)
	}
}

func TestImportAuthCopiesIntoProfile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	addProfileForTest(t, app, "work")

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "auth.json")
	want := `{"tokens":{"access_token":"abc"}}`
	if err := os.WriteFile(sourcePath, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"import-auth", "work", "--from", sourcePath}); err != nil {
		t.Fatalf("import-auth: %v", err)
	}

	destPath := filepath.Join(root, "profiles", "work", "auth.json")
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("unexpected imported auth: %q", got)
	}
}

func TestListAndShowIncludeAuthMetadataWhenAvailable(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	addProfileForTest(t, app, "work", "--shortcut", "cwork", "--default")

	authPath := filepath.Join(root, "profiles", "work", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte(testutil.TestAuthJSON("acct_123", "person@example.com", "Person")), 0o600); err != nil {
		t.Fatal(err)
	}
	sessionPath := filepath.Join(root, "profiles", "work", "sessions", "2026", "03", "27", "rollout.jsonl")
	if err := os.MkdirAll(filepath.Dir(sessionPath), 0o755); err != nil {
		t.Fatal(err)
	}
	sessionData := "{\"timestamp\":\"2026-03-27T03:41:41.365Z\",\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\",\"rate_limits\":{\"primary\":{\"used_percent\":7.0,\"window_minutes\":300},\"secondary\":{\"used_percent\":2.0,\"window_minutes\":10080},\"plan_type\":\"plus\"}}}\n"
	if err := os.WriteFile(sessionPath, []byte(sessionData), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"list"}); err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "work [default, shortcut=cwork, auth=present, email=person@example.com]") {
		t.Fatalf("unexpected list output with auth metadata: %s", out)
	}
	if !strings.Contains(out, "USAGE_SNAPSHOT=5h=7% 7d=2% plan=plus") {
		t.Fatalf("unexpected usage snapshot in list output: %s", out)
	}

	stdout.Reset()
	if err := app.Run([]string{"show", "work"}); err != nil {
		t.Fatalf("show profile: %v", err)
	}
	out = stdout.String()
	for _, want := range []string{
		"auth_present=true",
		"auth_email=person@example.com",
		"auth_account_id=acct_123",
		"auth_name=Person",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in show output: %s", want, out)
		}
	}
}

func TestListMarksUnreadableAuthAsPresent(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	addProfileForTest(t, app, "work", "--default")

	authPath := filepath.Join(root, "profiles", "work", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte("{invalid"), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"list"}); err != nil {
		t.Fatalf("list profiles: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "work [default, auth=present, auth_info=unreadable]") {
		t.Fatalf("unexpected list output for unreadable auth: %s", out)
	}
}

func TestLoginRunsCodexLoginWithProfileEnv(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)

	addProfileForTest(t, app, "work", "--default")

	authPath := filepath.Join(root, "profiles", "work", "auth.json")
	cmdRunner.run = func(cmd *exec.Cmd) error {
		if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
			return err
		}
		return os.WriteFile(authPath, []byte(testutil.TestAuthJSON("acct_work", "work@example.com", "Work User")), 0o600)
	}

	if err := app.Run([]string{"login", "work"}); err != nil {
		t.Fatalf("login: %v", err)
	}
	if cmdRunner.last == nil {
		t.Fatal("expected runner to be invoked")
	}
	if got := cmdRunner.last.Args; len(got) != 2 || got[0] != "codex" || got[1] != "login" {
		t.Fatalf("unexpected command args: %#v", got)
	}
	if !containsEnv(cmdRunner.last.Env, "CODEX_HOME="+filepath.Join(root, "profiles", "work")) {
		t.Fatalf("CODEX_HOME not set correctly: %#v", cmdRunner.last.Env)
	}
}

func TestLoginDoesNotTreatStaleAuthAsFreshLogin(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)

	addProfileForTest(t, app, "work", "--default")

	authPath := filepath.Join(root, "profiles", "work", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatal(err)
	}
	stale := []byte(`{"stale":true}`)
	if err := os.WriteFile(authPath, stale, 0o600); err != nil {
		t.Fatal(err)
	}

	err := app.Run([]string{"login", "work"})
	if err == nil {
		t.Fatal("expected login to fail when no fresh auth.json is created")
	}
	if !strings.Contains(err.Error(), "was not updated") {
		t.Fatalf("unexpected login error: %v", err)
	}

	got, readErr := os.ReadFile(authPath)
	if readErr != nil {
		t.Fatalf("expected stale auth to be restored: %v", readErr)
	}
	if string(got) != string(stale) {
		t.Fatalf("expected stale auth to be restored, got %q", got)
	}
}

func TestLoginFailureKeepsExistingAuthFile(t *testing.T) {
	cmdRunner := &recordingRunner{err: errors.New("login failed")}
	app, root := newTestApp(t, cmdRunner)

	addProfileForTest(t, app, "work", "--default")

	authPath := filepath.Join(root, "profiles", "work", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"existing":true}`)
	if err := os.WriteFile(authPath, want, 0o600); err != nil {
		t.Fatal(err)
	}

	err := app.Run([]string{"login", "work"})
	if err == nil || !strings.Contains(err.Error(), "login failed") {
		t.Fatalf("expected login failure, got %v", err)
	}

	got, readErr := os.ReadFile(authPath)
	if readErr != nil {
		t.Fatalf("expected auth file to remain after failed login: %v", readErr)
	}
	if string(got) != string(want) {
		t.Fatalf("expected auth file to remain unchanged, got %q", got)
	}
}

func TestLoginCopyFromCurrentImportsAuthWithoutRunner(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	addProfileForTest(t, app, "personal")

	currentHome := t.TempDir()
	t.Setenv("CODEX_HOME", currentHome)
	want := `{"copied":true}`
	if err := os.WriteFile(filepath.Join(currentHome, "auth.json"), []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"login", "personal", "--copy-from-current"}); err != nil {
		t.Fatalf("login --copy-from-current: %v", err)
	}
	if cmdRunner.last != nil {
		t.Fatal("runner should not be invoked for --copy-from-current")
	}

	destPath := filepath.Join(root, "profiles", "personal", "auth.json")
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("unexpected copied auth: %q", got)
	}
}

func TestLoginCopyFromCurrentUsesExplicitProfileWhenProvided(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	addProfileForTest(t, app, "work", "--default")
	addProfileForTest(t, app, "person")

	currentHome := t.TempDir()
	t.Setenv("CODEX_HOME", currentHome)
	want := `{"copied":true}`
	if err := os.WriteFile(filepath.Join(currentHome, "auth.json"), []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"login", "person", "--copy-from-current"}); err != nil {
		t.Fatalf("login person --copy-from-current: %v", err)
	}
	if cmdRunner.last != nil {
		t.Fatal("runner should not be invoked for --copy-from-current")
	}

	destPath := filepath.Join(root, "profiles", "person", "auth.json")
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("unexpected copied auth for explicit profile: %q", got)
	}
}

func TestRunUsesDefaultProfileWhenOmitted(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)

	addProfileForTest(t, app, "work", "--default")

	if err := app.Run([]string{"run", "--", "codex", "."}); err != nil {
		t.Fatalf("run with default profile: %v", err)
	}
	if !containsEnv(cmdRunner.last.Env, "CODEX_HOME="+filepath.Join(root, "profiles", "work")) {
		t.Fatalf("CODEX_HOME not set correctly: %#v", cmdRunner.last.Env)
	}
}

func TestRunUsesExplicitProfileWhenProvided(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)

	addProfileForTest(t, app, "work", "--default")
	addProfileForTest(t, app, "person")

	if err := app.Run([]string{"run", "person", "--", "codex", "."}); err != nil {
		t.Fatalf("run with explicit profile: %v", err)
	}
	if !containsEnv(cmdRunner.last.Env, "CODEX_HOME="+filepath.Join(root, "profiles", "person")) {
		t.Fatalf("CODEX_HOME not set correctly for explicit profile: %#v", cmdRunner.last.Env)
	}
}

func TestStatusReportsDefaultAndBareCodexFallback(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	addProfileForTest(t, app, "work", "--default")

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CODEX_HOME", "")

	stdout.Reset()
	if err := app.Run([]string{"status"}); err != nil {
		t.Fatalf("status: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "default_profile=work") {
		t.Fatalf("expected default profile in status output: %s", out)
	}
	if !strings.Contains(out, "default_profile_home="+filepath.Join(root, "profiles", "work")) {
		t.Fatalf("expected default profile home in status output: %s", out)
	}
	if !strings.Contains(out, "current_env_codex_home=unset") {
		t.Fatalf("expected unset env in status output: %s", out)
	}
	if !strings.Contains(out, "bare_codex_home="+filepath.Join(home, ".codex")) {
		t.Fatalf("expected bare codex fallback home in status output: %s", out)
	}
	if !strings.Contains(out, "bare_codex_source=codex_default") {
		t.Fatalf("expected codex default source in status output: %s", out)
	}
}

func TestStatusReportsCurrentEnvProfileAndJSON(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	addProfileForTest(t, app, "work", "--default")
	workHome := filepath.Join(root, "profiles", "work")
	t.Setenv("CODEX_HOME", workHome)

	stdout.Reset()
	if err := app.Run([]string{"status", "--json"}); err != nil {
		t.Fatalf("status --json: %v", err)
	}

	var report map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("status json decode: %v\n%s", err, stdout.String())
	}
	currentEnv, ok := report["current_env"].(map[string]any)
	if !ok || currentEnv["status"] != "set" || currentEnv["profile_name"] != "work" {
		t.Fatalf("unexpected current_env: %#v", report["current_env"])
	}
	bareCodex, ok := report["bare_codex"].(map[string]any)
	if !ok || bareCodex["source"] != "env" || bareCodex["profile_name"] != "work" {
		t.Fatalf("unexpected bare_codex: %#v", report["bare_codex"])
	}
}

func TestAliasesOutput(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	addProfileForTest(t, app, "work", "--shortcut", "cwork")

	stdout.Reset()
	if err := app.Run([]string{"aliases", "--shell", "zsh"}); err != nil {
		t.Fatalf("aliases: %v", err)
	}
	if got := stdout.String(); got != "alias cwork=\"codex-switch run 'work' -- codex\"\n" {
		t.Fatalf("unexpected aliases output: %q", got)
	}
}

func TestInitShellWritesManagedSnippet(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	rcPath := initShellForTest(t, app, "zsh")
	stdout.Reset()
	text := readFileForTest(t, rcPath)
	if !strings.Contains(text, "# >>> codex-switch >>>") {
		t.Fatalf("missing managed block: %s", text)
	}
	if !strings.Contains(text, `eval "$('codex-switch' aliases --shell zsh)"`) {
		t.Fatalf("missing alias init command: %s", text)
	}
}

func TestFormatUsageSnapshotRendersDynamicWindowLabels(t *testing.T) {
	got := formatUsageSnapshot(profile.UsageSnapshot{
		Present:                true,
		PrimaryUsedPercent:     7,
		SecondaryUsedPercent:   2,
		PrimaryWindowMinutes:   360,
		SecondaryWindowMinutes: 4320,
		PlanType:               "plus",
	})
	if got != "6h=7% 3d=2% plan=plus" {
		t.Fatalf("unexpected formatted snapshot: %q", got)
	}
}

func TestInitShellIsIdempotent(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	rcPath := initShellForTest(t, app, "zsh")
	if err := app.Run([]string{"init-shell", "--shell", "zsh", "--rc-file", rcPath}); err != nil {
		t.Fatalf("second init-shell: %v", err)
	}

	text := readFileForTest(t, rcPath)
	if strings.Count(text, "# >>> codex-switch >>>") != 1 {
		t.Fatalf("expected one managed block, got: %s", text)
	}
}

func TestUninitShellRemovesManagedSnippet(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	rcPath := initShellForTest(t, app, "zsh")
	if err := app.Run([]string{"uninit-shell", "--shell", "zsh", "--rc-file", rcPath}); err != nil {
		t.Fatalf("uninit-shell: %v", err)
	}

	text := readFileForTest(t, rcPath)
	if strings.Contains(text, "# >>> codex-switch >>>") {
		t.Fatalf("managed block should be removed: %s", text)
	}
}

func TestUninitShellIsIdempotent(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	rcPath := filepath.Join(t.TempDir(), ".zshrc")
	if err := os.WriteFile(rcPath, []byte("export PATH=$PATH:/tmp/bin\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"uninit-shell", "--shell", "zsh", "--rc-file", rcPath}); err != nil {
		t.Fatalf("uninit-shell: %v", err)
	}
	if !strings.Contains(stdout.String(), "status=unchanged") {
		t.Fatalf("expected unchanged status, got: %s", stdout.String())
	}
}

func TestUninitShellRejectsUnsupportedShellEvenWithExplicitRCFile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	err := app.Run([]string{"uninit-shell", "--shell", "bogus", "--rc-file", filepath.Join(t.TempDir(), ".rc")})
	if err == nil {
		t.Fatal("expected unsupported shell to fail")
	}
	if !strings.Contains(err.Error(), `unsupported shell "bogus"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupRemovesManagedSnippetAndKeepsDataByDefault(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	rcPath := initShellForTest(t, app, "zsh")
	addProfileForTest(t, app, "work")

	stdout.Reset()
	if err := app.Run([]string{"cleanup", "--shell", "zsh", "--rc-file", rcPath}); err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	text := readFileForTest(t, rcPath)
	if strings.Contains(text, "# >>> codex-switch >>>") {
		t.Fatalf("managed block should be removed: %s", text)
	}
	if _, err := os.Stat(filepath.Join(root, "config.json")); err != nil {
		t.Fatalf("config should be preserved: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "shell_init=removed") {
		t.Fatalf("expected shell cleanup status: %s", out)
	}
	if !strings.Contains(out, "data=preserved") {
		t.Fatalf("expected preserved data status: %s", out)
	}
	if !strings.Contains(out, "binary=remove_manually") {
		t.Fatalf("expected manual binary removal note: %s", out)
	}
}

func TestCleanupPurgesDataDirectory(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	rcPath := initShellForTest(t, app, "zsh")
	addProfileForTest(t, app, "work")

	stdout.Reset()
	if err := app.Run([]string{"cleanup", "--shell", "zsh", "--rc-file", rcPath, "--purge-data"}); err != nil {
		t.Fatalf("cleanup --purge-data: %v", err)
	}

	if _, err := os.Stat(root); !os.IsNotExist(err) {
		t.Fatalf("expected data root to be removed, stat err=%v", err)
	}
	if !strings.Contains(stdout.String(), "data=purged") {
		t.Fatalf("expected purged data status: %s", stdout.String())
	}
}

func TestCleanupRejectsUnsafePurgeRoot(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	unsafeRoot := t.TempDir()
	rcPath := initShellForTest(t, app, "zsh")
	t.Setenv("CODEX_SWITCH_HOME", unsafeRoot)

	err := app.Run([]string{"cleanup", "--shell", "zsh", "--rc-file", rcPath, "--purge-data"})
	if err == nil {
		t.Fatal("expected cleanup --purge-data to reject unsafe root")
	}
	if !strings.Contains(err.Error(), "missing the .codex-switch-root marker") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(unsafeRoot); statErr != nil {
		t.Fatalf("unsafe root should not be removed: %v", statErr)
	}
	text := readFileForTest(t, rcPath)
	if !strings.Contains(text, "# >>> codex-switch >>>") {
		t.Fatalf("cleanup should not remove shell init when purge validation fails: %s", text)
	}
}

func TestCleanupRejectsUnsupportedShellEvenWithExplicitRCFile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	err := app.Run([]string{"cleanup", "--shell", "bogus", "--rc-file", filepath.Join(t.TempDir(), ".rc")})
	if err == nil {
		t.Fatal("expected unsupported shell to fail")
	}
	if !strings.Contains(err.Error(), `unsupported shell "bogus"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCleanupRejectsHomeDirectoryWithoutChangingShellInit(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	home := t.TempDir()
	rcPath := filepath.Join(t.TempDir(), ".zshrc")
	if err := app.Run([]string{"init-shell", "--shell", "zsh", "--rc-file", rcPath}); err != nil {
		t.Fatalf("init-shell: %v", err)
	}
	t.Setenv("HOME", home)
	t.Setenv("CODEX_SWITCH_HOME", home)

	err := app.Run([]string{"cleanup", "--shell", "zsh", "--rc-file", rcPath, "--purge-data"})
	if err == nil {
		t.Fatal("expected cleanup --purge-data to reject home directory")
	}
	if !strings.Contains(err.Error(), "refusing to purge home directory") {
		t.Fatalf("unexpected error: %v", err)
	}
	data, readErr := os.ReadFile(rcPath)
	if readErr != nil {
		t.Fatalf("expected rc file to remain readable: %v", readErr)
	}
	if !strings.Contains(string(data), "# >>> codex-switch >>>") {
		t.Fatalf("cleanup should not remove shell init when purge validation fails: %s", string(data))
	}
}

func TestCleanupRejectsFileRootWithoutChangingShellInit(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	fileRoot := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(fileRoot, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rcPath := initShellForTest(t, app, "zsh")
	t.Setenv("CODEX_SWITCH_HOME", fileRoot)

	err := app.Run([]string{"cleanup", "--shell", "zsh", "--rc-file", rcPath, "--purge-data"})
	if err == nil {
		t.Fatal("expected cleanup --purge-data to reject file root")
	}
	if !strings.Contains(err.Error(), "is not a directory") {
		t.Fatalf("unexpected error: %v", err)
	}
	text := readFileForTest(t, rcPath)
	if !strings.Contains(text, "# >>> codex-switch >>>") {
		t.Fatalf("cleanup should not remove shell init when purge validation fails: %s", text)
	}
}

func TestCleanupRejectsDirectoryThatOnlyLooksLikeDataRoot(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	unsafeRoot := t.TempDir()
	t.Setenv("CODEX_SWITCH_HOME", unsafeRoot)
	if err := os.WriteFile(filepath.Join(unsafeRoot, "config.json"), []byte(`{"unrelated":true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(unsafeRoot, "profiles"), 0o755); err != nil {
		t.Fatal(err)
	}

	err := app.Run([]string{"cleanup", "--shell", "zsh", "--rc-file", filepath.Join(t.TempDir(), ".zshrc"), "--purge-data"})
	if err == nil {
		t.Fatal("expected cleanup --purge-data to reject fake data root")
	}
	if !strings.Contains(err.Error(), "missing the .codex-switch-root marker") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(unsafeRoot); statErr != nil {
		t.Fatalf("fake data root should not be removed: %v", statErr)
	}
}

func TestCleanupPurgesLegacyDataDirectoryWithoutMarker(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	legacyRoot := t.TempDir()
	t.Setenv("CODEX_SWITCH_HOME", legacyRoot)
	if err := os.WriteFile(filepath.Join(legacyRoot, "config.json"), []byte("{\n  \"version\": 1,\n  \"default_profile\": \"work\",\n  \"profiles\": {\n    \"work\": {\n      \"codex_home\": \"/tmp/work\"\n    }\n  }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"cleanup", "--shell", "zsh", "--rc-file", filepath.Join(t.TempDir(), ".zshrc"), "--purge-data"}); err != nil {
		t.Fatalf("cleanup should purge legacy data root without marker: %v", err)
	}
	if _, err := os.Stat(legacyRoot); !os.IsNotExist(err) {
		t.Fatalf("expected legacy root to be removed, stat err=%v", err)
	}
	if !strings.Contains(stdout.String(), "data=purged") {
		t.Fatalf("expected purged data status: %s", stdout.String())
	}
}

func TestConfigFileIsValidJSON(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)

	if err := app.Run([]string{"add", "personal"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["version"].(float64) != 1 {
		t.Fatalf("unexpected version: %#v", decoded["version"])
	}
}

func TestDoctorReportsProfilesAndShellInit(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	if err := app.Run([]string{"add", "work", "--default"}); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(root, "profiles", "work", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte(testutil.TestAuthJSON("acct_work", "work@example.com", "Work User")), 0o600); err != nil {
		t.Fatal(err)
	}

	rcPath := filepath.Join(t.TempDir(), ".zshrc")
	if err := app.Run([]string{"init-shell", "--shell", "zsh", "--rc-file", rcPath}); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# >>> codex-switch >>>\n# <<< codex-switch <<<\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"doctor"}); err != nil {
		t.Fatalf("doctor: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "profiles=1") {
		t.Fatalf("expected profile count in doctor output: %s", out)
	}
	if !strings.Contains(out, "default_profile=ok (work)") {
		t.Fatalf("expected default profile in doctor output: %s", out)
	}
	if !strings.Contains(out, "profile[work]=") || !strings.Contains(out, "auth:present") {
		t.Fatalf("expected profile auth status in doctor output: %s", out)
	}
	for _, want := range []string{
		"email:work@example.com",
		"account_id:acct_work",
		"name:Work User",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in doctor output: %s", want, out)
		}
	}
	if !strings.Contains(out, "shell_init=ok") {
		t.Fatalf("expected shell init status in doctor output: %s", out)
	}
}

func TestDoctorDetectsCustomRCFileTrackedByInitShell(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	home := t.TempDir()
	customRC := filepath.Join(t.TempDir(), "custom-zshrc")
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	if err := app.Run([]string{"init-shell", "--shell", "zsh", "--rc-file", customRC}); err != nil {
		t.Fatalf("init-shell custom rc: %v", err)
	}

	stdout.Reset()
	if err := app.Run([]string{"doctor"}); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "shell_init=ok") {
		t.Fatalf("expected shell init to be detected from tracked custom rc: %s", out)
	}
	if !strings.Contains(out, "zsh: "+customRC) {
		t.Fatalf("expected doctor to report custom rc path: %s", out)
	}
}

func TestDoctorDetectsManagedBlockInAnotherShellDefaultRC(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	if err := app.Run([]string{"init-shell", "--shell", "bash"}); err != nil {
		t.Fatalf("init-shell bash: %v", err)
	}

	stdout.Reset()
	if err := app.Run([]string{"doctor"}); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "shell_init=ok") {
		t.Fatalf("expected shell init to be detected from bash rc: %s", out)
	}
	if !strings.Contains(out, "bash: "+filepath.Join(home, ".bashrc")) {
		t.Fatalf("expected doctor to report bash rc path: %s", out)
	}
}

func TestDoctorDiscoversLegacyCustomRCWithoutStateFile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	home := t.TempDir()
	legacyDir := filepath.Join(home, "dotfiles")
	legacyRC := filepath.Join(legacyDir, "legacy-zshrc")
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	block, err := shellinit.ManagedBlock("zsh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyRC, []byte(block), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(shellinit.StatePath(root)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"doctor"}); err != nil {
		t.Fatalf("doctor: %v", err)
	}

	out := stdout.String()
	if !strings.Contains(out, "shell_init=ok") {
		t.Fatalf("expected doctor to discover legacy custom rc: %s", out)
	}
	if !strings.Contains(out, "zsh: "+legacyRC) {
		t.Fatalf("expected doctor to report legacy custom rc path: %s", out)
	}
}

func TestDoctorJSONOutput(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	if err := app.Run([]string{"add", "work", "--default"}); err != nil {
		t.Fatal(err)
	}
	authPath := filepath.Join(root, "profiles", "work", "auth.json")
	if err := os.MkdirAll(filepath.Dir(authPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(authPath, []byte(testutil.TestAuthJSON("acct_work", "work@example.com", "Work User")), 0o600); err != nil {
		t.Fatal(err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# >>> codex-switch >>>\n# <<< codex-switch <<<\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"doctor", "--json"}); err != nil {
		t.Fatalf("doctor --json: %v", err)
	}

	var report map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("doctor json decode: %v\n%s", err, stdout.String())
	}
	if report["config_path"] == "" {
		t.Fatalf("expected config_path in doctor json: %#v", report)
	}
	defaultProfile, ok := report["default_profile"].(map[string]any)
	if !ok || defaultProfile["status"] != "ok" || defaultProfile["name"] != "work" {
		t.Fatalf("unexpected default_profile: %#v", report["default_profile"])
	}
	profiles, ok := report["profiles"].([]any)
	if !ok || len(profiles) != 1 {
		t.Fatalf("unexpected profiles: %#v", report["profiles"])
	}
	profileEntry, ok := profiles[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected profile entry: %#v", profiles[0])
	}
	if profileEntry["auth_status"] != "present" || profileEntry["auth_email"] != "work@example.com" || profileEntry["auth_account_id"] != "acct_work" || profileEntry["auth_name"] != "Work User" {
		t.Fatalf("unexpected profile metadata: %#v", profileEntry)
	}
	shellInit, ok := report["shell_init"].(map[string]any)
	if !ok || shellInit["status"] != "ok" {
		t.Fatalf("unexpected shell_init: %#v", report["shell_init"])
	}
}

func TestRemoveProfile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	if err := app.Run([]string{"add", "work", "--default"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"add", "personal"}); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"remove", "work"}); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if !strings.Contains(stdout.String(), "Removed profile work") {
		t.Fatalf("unexpected remove output: %s", stdout.String())
	}

	cfg := loadTestConfig(t, root)
	if _, ok := cfg.Profiles["work"]; ok {
		t.Fatal("profile 'work' should have been removed")
	}
	// Default should have been reassigned to the remaining profile.
	if cfg.DefaultProfile != "personal" {
		t.Fatalf("expected default to be reassigned, got %q", cfg.DefaultProfile)
	}
}

func TestRemoveNonexistentProfileFails(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	if err := app.Run([]string{"remove", "ghost"}); err == nil {
		t.Fatal("expected error removing nonexistent profile")
	}
}

func TestSetDefault(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	if err := app.Run([]string{"add", "work", "--default"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"add", "personal"}); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"set-default", "personal"}); err != nil {
		t.Fatalf("set-default: %v", err)
	}
	if !strings.Contains(stdout.String(), "Default profile set to personal") {
		t.Fatalf("unexpected set-default output: %s", stdout.String())
	}

	cfg := loadTestConfig(t, root)
	if cfg.DefaultProfile != "personal" {
		t.Fatalf("expected default to be 'personal', got %q", cfg.DefaultProfile)
	}
}

func TestSetDefaultNonexistentProfileFails(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	if err := app.Run([]string{"set-default", "ghost"}); err == nil {
		t.Fatal("expected error setting nonexistent profile as default")
	}
}

func TestVersionOutput(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	stdout.Reset()
	if err := app.Run([]string{"version"}); err != nil {
		t.Fatalf("version: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "test" {
		t.Fatalf("unexpected version output: %q", got)
	}
}

func TestEnvOutputForDefaultProfile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	if err := app.Run([]string{"add", "work", "--default"}); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"env", "--shell", "zsh"}); err != nil {
		t.Fatalf("env: %v", err)
	}
	expected := filepath.Join(root, "profiles", "work")
	out := stdout.String()
	if !strings.Contains(out, "export CODEX_HOME=") || !strings.Contains(out, expected) {
		t.Fatalf("unexpected env output: %s", out)
	}
}

func TestEnvOutputForExplicitProfile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)

	if err := app.Run([]string{"add", "work", "--default"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"add", "personal"}); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	if err := app.Run([]string{"env", "--shell", "fish", "personal"}); err != nil {
		t.Fatalf("env: %v", err)
	}
	expected := filepath.Join(root, "profiles", "personal")
	out := stdout.String()
	if !strings.Contains(out, "set -gx CODEX_HOME") || !strings.Contains(out, expected) {
		t.Fatalf("unexpected env output: %s", out)
	}
}

func TestAddDuplicateProfileFails(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	if err := app.Run([]string{"add", "work"}); err != nil {
		t.Fatal(err)
	}
	if err := app.Run([]string{"add", "work"}); err == nil {
		t.Fatal("expected error adding duplicate profile")
	}
}

func TestAddInvalidProfileNameFails(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	if err := app.Run([]string{"add", "-invalid"}); err == nil {
		t.Fatal("expected error for invalid profile name")
	}
}

func TestShowNonexistentProfileFails(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	if err := app.Run([]string{"show", "ghost"}); err == nil {
		t.Fatal("expected error showing nonexistent profile")
	}
}

func TestUnknownCommandFails(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	if err := app.Run([]string{"bogus"}); err == nil {
		t.Fatal("expected error for unknown command")
	}
}

func TestRunWithoutDefaultProfileFails(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, _ := newTestApp(t, cmdRunner)

	if err := app.Run([]string{"run", "--", "codex"}); err == nil {
		t.Fatal("expected error running without any profiles configured")
	}
}

func containsEnv(env []string, want string) bool {
	for _, item := range env {
		if item == want {
			return true
		}
	}
	return false
}
