package profile

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const SkillsDirName = "skills"
const SkillManifestName = "SKILL.md"

type SyncSkillsResult struct {
	SourceRoot string
	DestRoot   string
	Skills     []string
}

func DefaultSkillsRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex", SkillsDirName), nil
}

func SkillsRoot(codexHome string) string {
	return filepath.Join(codexHome, SkillsDirName)
}

func SyncSkills(codexHome string, skillNames []string, allSkills bool) (SyncSkillsResult, error) {
	sourceRoot, err := DefaultSkillsRoot()
	if err != nil {
		return SyncSkillsResult{}, err
	}
	destRoot := SkillsRoot(codexHome)

	sourceInfo, err := os.Stat(sourceRoot)
	if err != nil {
		return SyncSkillsResult{}, err
	}
	if !sourceInfo.IsDir() {
		return SyncSkillsResult{}, fmt.Errorf("%s is not a directory", sourceRoot)
	}

	var skills []string
	if allSkills {
		skills, err = ListSkills(sourceRoot)
		if err != nil {
			return SyncSkillsResult{}, err
		}
	} else {
		skills, err = normalizeSkillNames(skillNames)
		if err != nil {
			return SyncSkillsResult{}, err
		}
	}

	if err := EnsureHome(codexHome); err != nil {
		return SyncSkillsResult{}, err
	}
	if err := os.MkdirAll(destRoot, 0o755); err != nil {
		return SyncSkillsResult{}, err
	}
	if err := ensureDirectoryNoSymlink(destRoot); err != nil {
		return SyncSkillsResult{}, err
	}

	for _, skill := range skills {
		sourcePath := filepath.Join(sourceRoot, skill)
		info, err := os.Lstat(sourcePath)
		if err != nil {
			if os.IsNotExist(err) {
				return SyncSkillsResult{}, fmt.Errorf("skill %q not found in %s", skill, sourceRoot)
			}
			return SyncSkillsResult{}, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return SyncSkillsResult{}, fmt.Errorf("skill %q is a symlink; symlinks are not supported", skill)
		}
		if !info.IsDir() {
			return SyncSkillsResult{}, fmt.Errorf("skill %q is not a directory", skill)
		}
		destPath := filepath.Join(destRoot, skill)
		if err := removeExistingSkill(destRoot, destPath); err != nil {
			return SyncSkillsResult{}, err
		}
		if err := copyTree(sourcePath, destPath, destRoot); err != nil {
			return SyncSkillsResult{}, err
		}
	}

	return SyncSkillsResult{
		SourceRoot: sourceRoot,
		DestRoot:   destRoot,
		Skills:     skills,
	}, nil
}

func ListSkills(sourceRoot string) ([]string, error) {
	var skills []string
	err := filepath.WalkDir(sourceRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == sourceRoot {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink; symlinks are not supported", path)
		}
		if !d.IsDir() {
			return nil
		}
		if _, err := os.Stat(filepath.Join(path, SkillManifestName)); err == nil {
			rel, err := filepath.Rel(sourceRoot, path)
			if err != nil {
				return err
			}
			skills = append(skills, rel)
			return filepath.SkipDir
		} else if !os.IsNotExist(err) {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(skills)
	return skills, nil
}

func normalizeSkillNames(skillNames []string) ([]string, error) {
	if len(skillNames) == 0 {
		return nil, fmt.Errorf("at least one skill is required unless --all-skills is set")
	}
	out := make([]string, 0, len(skillNames))
	seen := map[string]bool{}
	for _, raw := range skillNames {
		name, err := normalizeSkillName(raw)
		if err != nil {
			return nil, err
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out, nil
}

func normalizeSkillName(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("skill name cannot be empty")
	}
	if filepath.IsAbs(raw) {
		return "", fmt.Errorf("skill %q must be relative to %s", raw, SkillsDirName)
	}
	cleaned := filepath.Clean(raw)
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("skill %q must stay within %s", raw, SkillsDirName)
	}
	return cleaned, nil
}

func copyTree(src, dest, destRoot string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink; symlinks are not supported", path)
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if info.IsDir() {
			return mkdirAllNoSymlink(destRoot, target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", path)
		}
		return copyRegularFile(path, target, info.Mode().Perm(), destRoot)
	})
}

func removeExistingSkill(destRoot, destPath string) error {
	destRoot = filepath.Clean(destRoot)
	destPath = filepath.Clean(destPath)
	rel, err := filepath.Rel(destRoot, destPath)
	if err != nil {
		return err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("%s must stay within %s", destPath, destRoot)
	}

	if err := ensureParentDirectoriesNoSymlink(destRoot, filepath.Dir(destPath)); err != nil {
		return err
	}
	info, err := os.Lstat(destPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink; symlinks are not supported", destPath)
	}
	return os.RemoveAll(destPath)
}

func copyRegularFile(src, dest string, perm os.FileMode, destRoot string) error {
	if err := mkdirAllNoSymlink(destRoot, filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if info, err := os.Lstat(dest); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink; symlinks are not supported", dest)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s is not a regular file", dest)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.CreateTemp(filepath.Dir(dest), "."+filepath.Base(dest)+".*.tmp")
	if err != nil {
		return err
	}
	tmp := out.Name()

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Chmod(tmp, perm); err != nil {
		os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dest)
}

func mkdirAllNoSymlink(root, path string, perm os.FileMode) error {
	if err := ensureDirectoryNoSymlink(root); err != nil {
		return err
	}

	root = filepath.Clean(root)
	cleaned := filepath.Clean(path)
	if cleaned == root {
		return nil
	}

	rel, err := filepath.Rel(root, cleaned)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("%s must stay within %s", cleaned, root)
	}

	current := root
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)

		info, err := os.Lstat(current)
		if err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("%s is a symlink; symlinks are not supported", current)
			}
			if !info.IsDir() {
				return fmt.Errorf("%s is not a directory", current)
			}
			continue
		}
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.Mkdir(current, perm); err != nil && !os.IsExist(err) {
			return err
		}
	}

	return nil
}

func ensureParentDirectoriesNoSymlink(root, path string) error {
	root = filepath.Clean(root)
	cleaned := filepath.Clean(path)
	if cleaned == root {
		return nil
	}

	rel, err := filepath.Rel(root, cleaned)
	if err != nil {
		return err
	}
	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("%s must stay within %s", cleaned, root)
	}

	current := root
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink; symlinks are not supported", current)
		}
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", current)
		}
	}
	return nil
}

func ensureDirectoryNoSymlink(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s is a symlink; symlinks are not supported", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}
