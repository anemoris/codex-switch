package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncSkillsCopiesSingleSkill(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	writeSkillFile(t, home, "humanizer", "SKILL.md", "skill docs")
	writeSkillFile(t, home, "humanizer", "references/guide.md", "guide")

	result, err := SyncSkills(destHome, []string{"humanizer"}, false)
	if err != nil {
		t.Fatal(err)
	}

	if result.SourceRoot != filepath.Join(home, ".codex", "skills") {
		t.Fatalf("unexpected source root: %s", result.SourceRoot)
	}
	if result.DestRoot != filepath.Join(destHome, "skills") {
		t.Fatalf("unexpected dest root: %s", result.DestRoot)
	}
	if got := strings.Join(result.Skills, ","); got != "humanizer" {
		t.Fatalf("unexpected synced skills: %s", got)
	}
	assertFileContents(t, filepath.Join(destHome, "skills", "humanizer", "SKILL.md"), "skill docs")
	assertFileContents(t, filepath.Join(destHome, "skills", "humanizer", "references", "guide.md"), "guide")
}

func TestSyncSkillsCopiesMultipleAndNestedSkills(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	writeSkillFile(t, home, "humanizer", "SKILL.md", "humanizer docs")
	writeSkillFile(t, home, ".system/imagegen", "SKILL.md", "imagegen docs")

	result, err := SyncSkills(destHome, []string{"humanizer", ".system/imagegen"}, false)
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.Join(result.Skills, ","); got != "humanizer,.system/imagegen" {
		t.Fatalf("unexpected synced skills: %s", got)
	}
	assertFileContents(t, filepath.Join(destHome, "skills", "humanizer", "SKILL.md"), "humanizer docs")
	assertFileContents(t, filepath.Join(destHome, "skills", ".system", "imagegen", "SKILL.md"), "imagegen docs")
}

func TestSyncSkillsAllCopiesManifestDirectories(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	writeSkillFile(t, home, "humanizer", "SKILL.md", "humanizer docs")
	writeSkillFile(t, home, ".system/imagegen", "SKILL.md", "imagegen docs")
	writeSkillFile(t, home, ".system", "README.md", "not a skill")

	result, err := SyncSkills(destHome, nil, true)
	if err != nil {
		t.Fatal(err)
	}

	if got := strings.Join(result.Skills, ","); got != ".system/imagegen,humanizer" {
		t.Fatalf("unexpected all skills: %s", got)
	}
	assertFileContents(t, filepath.Join(destHome, "skills", "humanizer", "SKILL.md"), "humanizer docs")
	assertFileContents(t, filepath.Join(destHome, "skills", ".system", "imagegen", "SKILL.md"), "imagegen docs")
	if _, err := os.Stat(filepath.Join(destHome, "skills", ".system", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("expected non-skill grouping file to be skipped, got %v", err)
	}
}

func TestSyncSkillsReplacesMatchingSkillAndPreservesOtherTargetSkills(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	writeSkillFile(t, home, "humanizer", "SKILL.md", "fresh docs")

	destSkill := filepath.Join(destHome, "skills", "humanizer")
	if err := os.MkdirAll(destSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(destSkill, "SKILL.md"), []byte("stale docs"), 0o644); err != nil {
		t.Fatal(err)
	}
	staleExtraPath := filepath.Join(destSkill, "stale.md")
	if err := os.WriteFile(staleExtraPath, []byte("delete me"), 0o644); err != nil {
		t.Fatal(err)
	}
	extraPath := filepath.Join(destHome, "skills", "local-only", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(extraPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(extraPath, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := SyncSkills(destHome, []string{"humanizer"}, false); err != nil {
		t.Fatal(err)
	}

	assertFileContents(t, filepath.Join(destSkill, "SKILL.md"), "fresh docs")
	if _, err := os.Stat(staleExtraPath); !os.IsNotExist(err) {
		t.Fatalf("expected stale file inside replaced skill to be removed, got %v", err)
	}
	assertFileContents(t, extraPath, "keep me")
}

func TestSyncSkillsErrorsForMissingSkill(t *testing.T) {
	setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")

	_, err := SyncSkills(destHome, []string{"missing"}, false)
	if err == nil || !strings.Contains(err.Error(), `skill "missing" not found`) {
		t.Fatalf("expected missing skill error, got %v", err)
	}
}

func TestSyncSkillsErrorsForPathEscape(t *testing.T) {
	setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")

	_, err := SyncSkills(destHome, []string{"../auth.json"}, false)
	if err == nil || !strings.Contains(err.Error(), "must stay within") {
		t.Fatalf("expected path escape error, got %v", err)
	}
}

func TestSyncSkillsErrorsForMissingSourceRoot(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := SyncSkills(filepath.Join(t.TempDir(), "profile"), []string{"humanizer"}, false)
	if err == nil {
		t.Fatal("expected missing source root error")
	}
}

func TestSyncSkillsErrorsForSymlink(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	targetDir := filepath.Join(t.TempDir(), "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, filepath.Join(home, ".codex", "skills", "linked")); err != nil {
		t.Fatal(err)
	}

	_, err := SyncSkills(destHome, []string{"linked"}, false)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected symlink error, got %v", err)
	}
}

func TestSyncSkillsReplacesDestinationDirectorySymlinkInsideMatchingSkill(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	writeSkillFile(t, home, "humanizer", "SKILL.md", "skill docs")
	writeSkillFile(t, home, "humanizer", "references/guide.md", "guide")

	targetDir := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(destHome, "skills", "humanizer", "references")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatal(err)
	}

	if _, err := SyncSkills(destHome, []string{"humanizer"}, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "guide.md")); !os.IsNotExist(err) {
		t.Fatalf("expected sync not to write through destination symlink, got %v", err)
	}
	assertFileContents(t, filepath.Join(destHome, "skills", "humanizer", "references", "guide.md"), "guide")
}

func TestSyncSkillsReplacesDestinationFileSymlinkInsideMatchingSkill(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	writeSkillFile(t, home, "humanizer", "SKILL.md", "skill docs")

	outsideFile := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(destHome, "skills", "humanizer", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, linkPath); err != nil {
		t.Fatal(err)
	}

	if _, err := SyncSkills(destHome, []string{"humanizer"}, false); err != nil {
		t.Fatal(err)
	}
	assertFileContents(t, outsideFile, "outside")
	assertFileContents(t, filepath.Join(destHome, "skills", "humanizer", "SKILL.md"), "skill docs")
}

func TestSyncSkillsErrorsForDestinationSkillSymlink(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	writeSkillFile(t, home, "humanizer", "SKILL.md", "skill docs")

	targetDir := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(destHome, "skills", "humanizer")
	if err := os.MkdirAll(filepath.Dir(linkPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetDir, linkPath); err != nil {
		t.Fatal(err)
	}

	_, err := SyncSkills(destHome, []string{"humanizer"}, false)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("expected destination skill symlink error, got %v", err)
	}
}

func TestSyncSkillsDoesNotFollowDestinationTempSymlink(t *testing.T) {
	home := setupDefaultSkillsHome(t)
	destHome := filepath.Join(t.TempDir(), "profile")
	writeSkillFile(t, home, "humanizer", "SKILL.md", "skill docs")

	outsideFile := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outsideFile, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	tmpLink := filepath.Join(destHome, "skills", "humanizer", "SKILL.md.tmp")
	if err := os.MkdirAll(filepath.Dir(tmpLink), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideFile, tmpLink); err != nil {
		t.Fatal(err)
	}

	if _, err := SyncSkills(destHome, []string{"humanizer"}, false); err != nil {
		t.Fatal(err)
	}
	assertFileContents(t, filepath.Join(destHome, "skills", "humanizer", "SKILL.md"), "skill docs")
	assertFileContents(t, outsideFile, "outside")
}

func setupDefaultSkillsHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".codex", "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	return home
}

func writeSkillFile(t *testing.T, home, skill, relPath, contents string) {
	t.Helper()
	path := filepath.Join(home, ".codex", "skills", filepath.FromSlash(skill), filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertFileContents(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != want {
		t.Fatalf("unexpected contents for %s: want %q got %q", path, want, got)
	}
}
