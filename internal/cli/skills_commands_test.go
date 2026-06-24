package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncSkillsCopiesSpecificSkillToProfile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)
	home := setupDefaultSkillsForCLITest(t)
	writeCLISkillFile(t, home, "humanizer", "SKILL.md", "humanizer docs")
	addProfileForTest(t, app, "work")

	stdout.Reset()
	if err := app.Run([]string{"sync-skills", "work", "humanizer"}); err != nil {
		t.Fatalf("sync-skills: %v", err)
	}

	assertCLIFileContents(t, filepath.Join(root, "profiles", "work", "skills", "humanizer", "SKILL.md"), "humanizer docs")
	out := stdout.String()
	for _, want := range []string{
		"Synced skills for profile work",
		"skills=1",
		"source=" + filepath.Join(home, ".codex", "skills"),
		"dest=" + filepath.Join(root, "profiles", "work", "skills"),
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in output: %s", want, out)
		}
	}
}

func TestSyncSkillsCopiesMultipleAndNestedSkillsToProfile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	home := setupDefaultSkillsForCLITest(t)
	writeCLISkillFile(t, home, "humanizer", "SKILL.md", "humanizer docs")
	writeCLISkillFile(t, home, ".system/imagegen", "SKILL.md", "imagegen docs")
	addProfileForTest(t, app, "work")

	if err := app.Run([]string{"sync-skills", "work", "humanizer", ".system/imagegen"}); err != nil {
		t.Fatalf("sync-skills: %v", err)
	}

	assertCLIFileContents(t, filepath.Join(root, "profiles", "work", "skills", "humanizer", "SKILL.md"), "humanizer docs")
	assertCLIFileContents(t, filepath.Join(root, "profiles", "work", "skills", ".system", "imagegen", "SKILL.md"), "imagegen docs")
}

func TestSyncSkillsAllSkillsToProfile(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	home := setupDefaultSkillsForCLITest(t)
	writeCLISkillFile(t, home, "humanizer", "SKILL.md", "humanizer docs")
	writeCLISkillFile(t, home, ".system/imagegen", "SKILL.md", "imagegen docs")
	addProfileForTest(t, app, "work")

	if err := app.Run([]string{"sync-skills", "work", "--all-skills"}); err != nil {
		t.Fatalf("sync-skills --all-skills: %v", err)
	}

	assertCLIFileContents(t, filepath.Join(root, "profiles", "work", "skills", "humanizer", "SKILL.md"), "humanizer docs")
	assertCLIFileContents(t, filepath.Join(root, "profiles", "work", "skills", ".system", "imagegen", "SKILL.md"), "imagegen docs")
}

func TestSyncSkillsAllProfilesSpecificSkill(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	stdout := app.stdout.(*bytes.Buffer)
	home := setupDefaultSkillsForCLITest(t)
	writeCLISkillFile(t, home, "humanizer", "SKILL.md", "humanizer docs")
	addProfileForTest(t, app, "work")
	addProfileForTest(t, app, "personal")

	stdout.Reset()
	if err := app.Run([]string{"sync-skills", "--all", "humanizer"}); err != nil {
		t.Fatalf("sync-skills --all: %v", err)
	}

	assertCLIFileContents(t, filepath.Join(root, "profiles", "work", "skills", "humanizer", "SKILL.md"), "humanizer docs")
	assertCLIFileContents(t, filepath.Join(root, "profiles", "personal", "skills", "humanizer", "SKILL.md"), "humanizer docs")
	out := stdout.String()
	if !strings.Contains(out, "Synced skills for profile personal") || !strings.Contains(out, "Synced skills for profile work") || !strings.Contains(out, "profiles_synced=2") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSyncSkillsAllProfilesAllSkills(t *testing.T) {
	cmdRunner := &recordingRunner{}
	app, root := newTestApp(t, cmdRunner)
	home := setupDefaultSkillsForCLITest(t)
	writeCLISkillFile(t, home, ".system/imagegen", "SKILL.md", "imagegen docs")
	addProfileForTest(t, app, "work")
	addProfileForTest(t, app, "personal")

	if err := app.Run([]string{"sync-skills", "--all", "--all-skills"}); err != nil {
		t.Fatalf("sync-skills --all --all-skills: %v", err)
	}

	assertCLIFileContents(t, filepath.Join(root, "profiles", "work", "skills", ".system", "imagegen", "SKILL.md"), "imagegen docs")
	assertCLIFileContents(t, filepath.Join(root, "profiles", "personal", "skills", ".system", "imagegen", "SKILL.md"), "imagegen docs")
}

func TestSyncSkillsRejectsInvalidArgumentCombinations(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing target", args: []string{"sync-skills", "humanizer"}},
		{name: "target and all profiles", args: []string{"sync-skills", "work", "--all", "humanizer"}},
		{name: "missing skills", args: []string{"sync-skills", "work"}},
		{name: "skills and all skills", args: []string{"sync-skills", "work", "--all-skills", "humanizer"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdRunner := &recordingRunner{}
			app, _ := newTestApp(t, cmdRunner)
			if tt.name != "missing target" {
				addProfileForTest(t, app, "work")
			}
			err := app.Run(tt.args)
			if err == nil || !strings.Contains(err.Error(), "usage: codex-switch sync-skills") {
				t.Fatalf("expected usage error, got %v", err)
			}
		})
	}
}

func setupDefaultSkillsForCLITest(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeCLISkillFile(t *testing.T, home, skill, relPath, contents string) {
	t.Helper()
	path := filepath.Join(home, ".codex", "skills", filepath.FromSlash(skill), filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertCLIFileContents(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("unexpected contents for %s: want %q got %q", path, want, got)
	}
}
