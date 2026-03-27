package shellinit

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
)

const StateFileName = "shell-init-state.json"

type State struct {
	Entries map[string][]string `json:"entries,omitempty"`
}

type Candidate struct {
	Shell  string
	RCPath string
}

func LoadState(root string) (State, error) {
	data, err := os.ReadFile(StatePath(root))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{Entries: map[string][]string{}}, nil
		}
		return State{}, err
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	if state.Entries == nil {
		state.Entries = map[string][]string{}
	}
	for shell, paths := range state.Entries {
		state.Entries[shell] = normalizePaths(paths)
	}
	return state, nil
}

func SaveState(root string, state State) error {
	for shell, paths := range state.Entries {
		normalized := normalizePaths(paths)
		if len(normalized) == 0 {
			delete(state.Entries, shell)
			continue
		}
		state.Entries[shell] = normalized
	}

	path := StatePath(root)
	if len(state.Entries) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func StatePath(root string) string {
	return filepath.Join(root, StateFileName)
}

func (s *State) Add(shell, rcPath string) {
	if s.Entries == nil {
		s.Entries = map[string][]string{}
	}
	s.Entries[shell] = append(s.Entries[shell], filepath.Clean(rcPath))
	s.Entries[shell] = normalizePaths(s.Entries[shell])
}

func (s *State) Remove(shell, rcPath string) {
	if s.Entries == nil {
		return
	}
	want := filepath.Clean(rcPath)
	paths := s.Entries[shell]
	out := paths[:0]
	for _, path := range paths {
		if filepath.Clean(path) != want {
			out = append(out, path)
		}
	}
	if len(out) == 0 {
		delete(s.Entries, shell)
		return
	}
	s.Entries[shell] = out
}

func (s State) Paths(shell string) []string {
	if s.Entries == nil {
		return nil
	}
	paths := append([]string(nil), s.Entries[shell]...)
	sort.Strings(paths)
	return paths
}

func normalizePaths(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		clean := filepath.Clean(path)
		if _, ok := seen[clean]; ok {
			continue
		}
		seen[clean] = struct{}{}
		out = append(out, clean)
	}
	sort.Strings(out)
	return out
}
