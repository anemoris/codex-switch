package shellinit

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	StartMarker = "# >>> codex-switch >>>"
	EndMarker   = "# <<< codex-switch <<<"
)

func DetectShell(shellEnv string) string {
	base := filepath.Base(shellEnv)
	switch base {
	case "bash", "zsh", "fish":
		return base
	default:
		return "zsh"
	}
}

func DefaultRCPath(shell string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	switch shell {
	case "bash":
		return filepath.Join(home, ".bashrc"), nil
	case "zsh":
		return filepath.Join(home, ".zshrc"), nil
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func ManagedBlock(shell string) (string, error) {
	switch shell {
	case "bash", "zsh":
		return strings.Join([]string{
			StartMarker,
			fmt.Sprintf("eval \"$('codex-switch' aliases --shell %s)\"", shell),
			EndMarker,
			"",
		}, "\n"), nil
	case "fish":
		return strings.Join([]string{
			StartMarker,
			"'codex-switch' aliases --shell fish | source",
			EndMarker,
			"",
		}, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported shell %q", shell)
	}
}

func Install(rcPath, block string) (bool, error) {
	if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
		return false, err
	}

	existing, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return false, err
	}

	updated, changed := replaceManagedBlock(existing, block)
	if !changed {
		return false, nil
	}

	if err := atomicWriteFile(rcPath, updated, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func Uninstall(rcPath string) (bool, error) {
	existing, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	updated, changed := removeManagedBlock(existing)
	if !changed {
		return false, nil
	}

	if err := atomicWriteFile(rcPath, updated, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

func HasManagedBlock(rcPath string) (bool, error) {
	data, err := os.ReadFile(rcPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	text := string(data)
	start := strings.Index(text, StartMarker)
	end := strings.Index(text, EndMarker)
	return start >= 0 && end >= start, nil
}

func replaceManagedBlock(existing []byte, block string) ([]byte, bool) {
	text := string(existing)
	start := strings.Index(text, StartMarker)
	end := strings.Index(text, EndMarker)

	if start >= 0 && end >= start {
		end += len(EndMarker)
		if end < len(text) && text[end] == '\n' {
			end++
		}
		replaced := text[:start] + block + text[end:]
		if replaced == text {
			return existing, false
		}
		return []byte(replaced), true
	}

	var buf bytes.Buffer
	buf.Write(existing)
	if len(existing) > 0 && existing[len(existing)-1] != '\n' {
		buf.WriteByte('\n')
	}
	if len(existing) > 0 {
		buf.WriteByte('\n')
	}
	buf.WriteString(block)
	return buf.Bytes(), true
}

func removeManagedBlock(existing []byte) ([]byte, bool) {
	text := string(existing)
	start := strings.Index(text, StartMarker)
	end := strings.Index(text, EndMarker)
	if start < 0 || end < start {
		return existing, false
	}

	// Remove the extra blank line we add before the managed block, but keep the
	// newline that terminates the previous user line.
	if start > 1 && text[start-1] == '\n' && text[start-2] == '\n' {
		start--
	}

	end += len(EndMarker)
	if end < len(text) && text[end] == '\n' {
		end++
	}

	updated := text[:start] + text[end:]
	if strings.TrimSpace(updated) == "" {
		return []byte{}, true
	}
	return []byte(updated), true
}

// atomicWriteFile writes data to a temporary file and then renames it to path,
// preventing partial writes from corrupting the target file.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	if info, err := os.Stat(path); err == nil {
		perm = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
