package cli

import "fmt"

func (a *App) printHelp() error {
	_, err := fmt.Fprint(a.stdout, `codex-switch manages multiple Codex profiles with isolated CODEX_HOME directories.

Usage:
  codex-switch add <name> [--home path] [--label text] [--shortcut command] [--default]
  codex-switch list
  codex-switch show <name>
  codex-switch status [--json]
  codex-switch doctor [--json]
  codex-switch remove <name>
  codex-switch set-default <name>
  codex-switch run [<profile>] [-- command args...]
  codex-switch login [<profile>] [--copy-from-current] [-- command args...]
  codex-switch import-auth <profile> [--from path]
  codex-switch env [<profile>] [--shell sh|bash|zsh|fish]
  codex-switch aliases [--shell bash|zsh|fish]
  codex-switch init-shell [--shell bash|zsh|fish] [--rc-file path]
  codex-switch uninit-shell [--shell bash|zsh|fish] [--rc-file path]
  codex-switch cleanup [--shell bash|zsh|fish] [--rc-file path] [--purge-data]
  codex-switch version

Examples:
  codex-switch add work --shortcut cwork --default
  codex-switch login work
  codex-switch login personal --copy-from-current
  codex-switch import-auth personal --from ~/.codex/auth.json
  codex-switch status
  codex-switch doctor
  codex-switch init-shell --shell zsh
  codex-switch uninit-shell --shell zsh
  codex-switch cleanup --shell zsh --purge-data
  codex-switch run work -- codex .
  eval "$(codex-switch env work)"
  eval "$(codex-switch aliases --shell zsh)"
`)
	return err
}
