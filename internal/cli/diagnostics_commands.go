package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"codex-switch/internal/config"
	"codex-switch/internal/profile"
	"codex-switch/internal/shellinit"
)

func (a *App) runStatus(args []string) error {
	fs := a.newFlagSet("status")
	var outputJSON bool
	fs.BoolVar(&outputJSON, "json", false, "print status output as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usageError("usage: codex-switch status [--json]")
	}

	report, err := a.buildStatusReport()
	if err != nil {
		return err
	}
	if outputJSON {
		enc := json.NewEncoder(a.stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	if report.DefaultProfile.Name == "" {
		if _, err := fmt.Fprintln(a.stdout, "default_profile=missing"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(a.stdout, "default_profile=%s\n", report.DefaultProfile.Name); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(a.stdout, "default_profile_home=%s\n", report.DefaultProfile.CodexHome); err != nil {
			return err
		}
	}

	if report.CurrentEnv.Status == "set" {
		if _, err := fmt.Fprintf(a.stdout, "current_env_codex_home=%s\n", report.CurrentEnv.CodexHome); err != nil {
			return err
		}
		if report.CurrentEnv.ProfileName != "" {
			if _, err := fmt.Fprintf(a.stdout, "current_env_profile=%s\n", report.CurrentEnv.ProfileName); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintln(a.stdout, "current_env_profile=none"); err != nil {
				return err
			}
		}
	} else {
		if _, err := fmt.Fprintln(a.stdout, "current_env_codex_home=unset"); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintf(a.stdout, "bare_codex_home=%s\n", report.BareCodex.CodexHome); err != nil {
		return err
	}
	if report.BareCodex.ProfileName != "" {
		if _, err := fmt.Fprintf(a.stdout, "bare_codex_profile=%s\n", report.BareCodex.ProfileName); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintln(a.stdout, "bare_codex_profile=none"); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(a.stdout, "bare_codex_source=%s\n", report.BareCodex.Source); err != nil {
		return err
	}
	_, err = fmt.Fprintf(a.stdout, "hint=%s\n", report.Hint)
	return err
}

func (a *App) runDoctor(args []string) error {
	fs := a.newFlagSet("doctor")
	var outputJSON bool
	fs.BoolVar(&outputJSON, "json", false, "print doctor output as JSON")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return usageError("usage: codex-switch doctor [--json]")
	}

	report, err := a.buildDoctorReport()
	if err != nil {
		return err
	}
	if outputJSON {
		enc := json.NewEncoder(a.stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	if _, err := fmt.Fprintf(a.stdout, "codex_binary=%s", report.CodexBinary.Status); err != nil {
		return err
	}
	if report.CodexBinary.Path != "" {
		if _, err := fmt.Fprintf(a.stdout, " (%s)\n", report.CodexBinary.Path); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(a.stdout, " (%s)\n", report.CodexBinary.Error); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(a.stdout, "config_path=%s\n", report.ConfigPath); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(a.stdout, "profiles=%d\n", report.ProfilesCount); err != nil {
		return err
	}
	if report.DefaultProfile.Status == "missing" {
		if _, err := fmt.Fprintln(a.stdout, "default_profile=missing"); err != nil {
			return err
		}
	} else {
		if _, err := fmt.Fprintf(a.stdout, "default_profile=ok (%s)\n", report.DefaultProfile.Name); err != nil {
			return err
		}
	}

	for _, p := range report.Profiles {
		if _, err := fmt.Fprintf(a.stdout, "profile[%s]=codex_home:%s auth:%s", p.Name, p.CodexHome, p.AuthStatus); err != nil {
			return err
		}
		if p.AuthEmail != "" {
			if _, err := fmt.Fprintf(a.stdout, " email:%s", p.AuthEmail); err != nil {
				return err
			}
		}
		if p.AuthAccountID != "" {
			if _, err := fmt.Fprintf(a.stdout, " account_id:%s", p.AuthAccountID); err != nil {
				return err
			}
		}
		if p.AuthName != "" {
			if _, err := fmt.Fprintf(a.stdout, " name:%s", p.AuthName); err != nil {
				return err
			}
		}
		if p.AuthInfoError != "" {
			if _, err := fmt.Fprintf(a.stdout, " auth_info_error:%s", p.AuthInfoError); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(a.stdout); err != nil {
			return err
		}
	}

	_, err = fmt.Fprintf(a.stdout, "shell_init=%s (%s: %s)\n", report.ShellInit.Status, report.ShellInit.Shell, report.ShellInit.RCPath)
	return err
}

type doctorReport struct {
	CodexBinary    doctorCodexBinary    `json:"codex_binary"`
	ConfigPath     string               `json:"config_path"`
	ProfilesCount  int                  `json:"profiles_count"`
	DefaultProfile doctorDefaultProfile `json:"default_profile"`
	Profiles       []doctorProfile      `json:"profiles"`
	ShellInit      doctorShellInit      `json:"shell_init"`
}

type doctorCodexBinary struct {
	Status string `json:"status"`
	Path   string `json:"path,omitempty"`
	Error  string `json:"error,omitempty"`
}

type doctorDefaultProfile struct {
	Status string `json:"status"`
	Name   string `json:"name,omitempty"`
}

type doctorProfile struct {
	Name          string `json:"name"`
	CodexHome     string `json:"codex_home"`
	AuthStatus    string `json:"auth_status"`
	AuthEmail     string `json:"auth_email,omitempty"`
	AuthAccountID string `json:"auth_account_id,omitempty"`
	AuthName      string `json:"auth_name,omitempty"`
	AuthInfoError string `json:"auth_info_error,omitempty"`
}

type doctorShellInit struct {
	Status string `json:"status"`
	Shell  string `json:"shell"`
	RCPath string `json:"rc_path"`
}

type statusReport struct {
	DefaultProfile statusDefaultProfile `json:"default_profile"`
	CurrentEnv     statusCurrentEnv     `json:"current_env"`
	BareCodex      statusBareCodex      `json:"bare_codex"`
	Hint           string               `json:"hint"`
}

type statusDefaultProfile struct {
	Name      string `json:"name,omitempty"`
	CodexHome string `json:"codex_home,omitempty"`
}

type statusCurrentEnv struct {
	Status      string `json:"status"`
	CodexHome   string `json:"codex_home,omitempty"`
	ProfileName string `json:"profile_name,omitempty"`
}

type statusBareCodex struct {
	Source      string `json:"source"`
	CodexHome   string `json:"codex_home"`
	ProfileName string `json:"profile_name,omitempty"`
}

func (a *App) buildStatusReport() (statusReport, error) {
	cfg, err := a.store.Load()
	if err != nil {
		return statusReport{}, err
	}

	report := statusReport{
		CurrentEnv: statusCurrentEnv{Status: "unset"},
	}

	if cfg.DefaultProfile != "" {
		if p, ok := cfg.Profiles[cfg.DefaultProfile]; ok {
			report.DefaultProfile = statusDefaultProfile{
				Name:      cfg.DefaultProfile,
				CodexHome: p.CodexHome,
			}
		}
	}

	matches := func(path string) string {
		for name, p := range cfg.Profiles {
			if p.CodexHome == path {
				return name
			}
		}
		return ""
	}

	if current := os.Getenv("CODEX_HOME"); current != "" {
		expanded, err := config.ExpandPath(current)
		if err != nil {
			expanded = current
		}
		report.CurrentEnv = statusCurrentEnv{
			Status:      "set",
			CodexHome:   expanded,
			ProfileName: matches(expanded),
		}
		report.BareCodex = statusBareCodex{
			Source:      "env",
			CodexHome:   expanded,
			ProfileName: matches(expanded),
		}
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return statusReport{}, err
		}
		defaultCodexHome := filepath.Join(home, ".codex")
		report.BareCodex = statusBareCodex{
			Source:      "codex_default",
			CodexHome:   defaultCodexHome,
			ProfileName: matches(defaultCodexHome),
		}
	}

	if report.DefaultProfile.Name != "" {
		report.Hint = fmt.Sprintf("use `codex-switch run %s -- codex ...` or `eval \"$(codex-switch env %s)\"` to target the default profile", report.DefaultProfile.Name, report.DefaultProfile.Name)
	} else {
		report.Hint = "set a default profile or run `codex-switch run <profile> -- codex ...`"
	}

	return report, nil
}

func (a *App) buildDoctorReport() (doctorReport, error) {
	report := doctorReport{}

	if path, err := exec.LookPath("codex"); err == nil {
		report.CodexBinary = doctorCodexBinary{
			Status: "ok",
			Path:   path,
		}
	} else {
		report.CodexBinary = doctorCodexBinary{
			Status: "missing",
			Error:  err.Error(),
		}
	}

	cfg, err := a.store.Load()
	if err != nil {
		return doctorReport{}, err
	}
	report.ConfigPath = a.store.Path()
	report.ProfilesCount = len(cfg.Profiles)
	if cfg.DefaultProfile == "" {
		report.DefaultProfile = doctorDefaultProfile{Status: "missing"}
	} else {
		report.DefaultProfile = doctorDefaultProfile{
			Status: "ok",
			Name:   cfg.DefaultProfile,
		}
	}

	for _, name := range sortedProfileNames(cfg) {
		p := cfg.Profiles[name]
		authSummary, authErr := profile.ReadAuthSummary(p.CodexHome)
		authStatus := "missing"
		if authSummary.Present {
			authStatus = "present"
		}
		entry := doctorProfile{
			Name:          name,
			CodexHome:     p.CodexHome,
			AuthStatus:    authStatus,
			AuthEmail:     authSummary.Email,
			AuthAccountID: authSummary.AccountID,
			AuthName:      authSummary.Name,
		}
		if authErr != nil {
			entry.AuthInfoError = authErr.Error()
		}
		report.Profiles = append(report.Profiles, entry)
	}

	shell, rcPath, ok, err := a.detectShellInit()
	if err != nil {
		return doctorReport{}, err
	}
	report.ShellInit = doctorShellInit{
		Shell:  shell,
		RCPath: rcPath,
	}
	if ok {
		report.ShellInit.Status = "ok"
		return report, nil
	}
	report.ShellInit.Status = "missing"
	return report, nil
}

func (a *App) detectShellInit() (string, string, bool, error) {
	currentShell := shellinit.DetectShell(os.Getenv("SHELL"))
	candidates, fallbackShell, fallbackPath, err := a.shellInitCandidates(currentShell)
	if err != nil {
		return "", "", false, err
	}
	for _, candidate := range candidates {
		hasBlock, err := shellinit.HasManagedBlock(candidate.RCPath)
		if err != nil {
			continue
		}
		if hasBlock {
			return candidate.Shell, candidate.RCPath, true, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", false, err
	}
	if candidate, ok, err := shellinit.DiscoverManagedBlock(home); err == nil && ok {
		return candidate.Shell, candidate.RCPath, true, nil
	}
	return fallbackShell, fallbackPath, false, nil
}

func (a *App) shellInitCandidates(currentShell string) ([]shellinit.Candidate, string, string, error) {
	root := filepath.Dir(a.store.Path())
	state, err := shellinit.LoadState(root)
	if err != nil {
		return nil, "", "", err
	}

	shells := []string{currentShell}
	for _, shell := range []string{"bash", "zsh", "fish"} {
		if shell != currentShell {
			shells = append(shells, shell)
		}
	}

	seen := map[string]struct{}{}
	candidates := make([]shellinit.Candidate, 0, len(shells)*2)
	add := func(shell, rcPath string) {
		key := shell + "\x00" + filepath.Clean(rcPath)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		candidates = append(candidates, shellinit.Candidate{
			Shell:  shell,
			RCPath: filepath.Clean(rcPath),
		})
	}

	for _, shell := range shells {
		for _, rcPath := range state.Paths(shell) {
			add(shell, rcPath)
		}
	}
	for _, shell := range shells {
		rcPath, err := shellinit.DefaultRCPath(shell)
		if err != nil {
			return nil, "", "", err
		}
		add(shell, rcPath)
	}

	fallbackPath, err := shellinit.DefaultRCPath(currentShell)
	if err != nil {
		return nil, "", "", err
	}
	return candidates, currentShell, fallbackPath, nil
}
