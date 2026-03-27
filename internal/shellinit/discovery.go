package shellinit

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	maxDiscoveryDepth    = 4
	maxDiscoveryFileSize = 1 << 20
)

func DetectManagedBlock(path string) (Candidate, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Candidate{}, false, err
	}

	text := string(data)
	start := strings.Index(text, StartMarker)
	end := strings.Index(text, EndMarker)
	if start < 0 || end < start {
		return Candidate{}, false, nil
	}

	shell := detectShellFromManagedBlock(text)
	if shell == "" {
		shell = detectShellFromPath(path)
	}
	if shell == "" {
		shell = "zsh"
	}

	return Candidate{
		Shell:  shell,
		RCPath: filepath.Clean(path),
	}, true, nil
}

func DiscoverManagedBlock(root string) (Candidate, bool, error) {
	root = filepath.Clean(root)
	var found Candidate
	var foundOK bool

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDiscoveryDir(root, path, d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if tooDeep(root, path) {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Size() > maxDiscoveryFileSize {
			return nil
		}

		candidate, ok, err := DetectManagedBlock(path)
		if err != nil || !ok {
			return nil
		}
		found = candidate
		foundOK = true
		return filepath.SkipAll
	})
	if err != nil {
		return Candidate{}, false, err
	}
	return found, foundOK, nil
}

func shouldSkipDiscoveryDir(root, path, name string) bool {
	if tooDeep(root, path) {
		return true
	}
	switch name {
	case ".git", "node_modules", "Library", ".cache", ".Trash":
		return true
	default:
		return false
	}
}

func tooDeep(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return false
	}
	depth := strings.Count(rel, string(filepath.Separator)) + 1
	return depth > maxDiscoveryDepth
}

func detectShellFromManagedBlock(text string) string {
	switch {
	case strings.Contains(text, "aliases --shell bash"):
		return "bash"
	case strings.Contains(text, "aliases --shell zsh"):
		return "zsh"
	case strings.Contains(text, "aliases --shell fish"):
		return "fish"
	default:
		return ""
	}
}

func detectShellFromPath(path string) string {
	base := filepath.Base(path)
	switch {
	case strings.Contains(base, "bash"):
		return "bash"
	case strings.Contains(base, "zsh"):
		return "zsh"
	case strings.Contains(base, "fish"):
		return "fish"
	default:
		return ""
	}
}
