# codex-switch

English | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md) | [Español](./README.es.md) | [한국어](./README.ko.md)

Use separate `CODEX_HOME` directories to keep multiple Codex accounts isolated.

The idea is simple:

- each account gets its own `CODEX_HOME`
- `codex-switch` starts `codex` with the profile you choose
- shortcuts are shell aliases, so there is no need to rewrite a shared `~/.codex/auth.json`

## Why This Setup

Separate `CODEX_HOME` directories keep `sessions`, usage metadata, and auth snapshots from bleeding into each other. It also keeps the behavior easy to reason about: the active account is whichever one belongs to the `CODEX_HOME` that launched `codex`.

## Project Layout

The CLI follows a fairly standard Go project layout:

```text
cmd/codex-switch/      # binary entrypoint
internal/cli/          # command parsing and user-facing behavior
internal/config/       # config model and persistence
internal/profile/      # CODEX_HOME and auth.json operations
internal/runner/       # external command execution abstraction
internal/shellinit/    # shell initialization snippet install/uninstall
```

## Install

Install with Homebrew:

```sh
brew install anemoris/tap/codex-switch
```

To build from source, the included `Makefile` is the easiest way:

```sh
make build
```

That injects the `dev` version string automatically. To build a specific version:

```sh
make build VERSION=v1.0.0
```

You can also build it directly with `go build`:

```sh
go build -o codex-switch ./cmd/codex-switch
```

If you build from source and want to run `codex-switch` directly, move the binary into a directory that is already in your `PATH`, or run it as `./codex-switch`.

## Testing And Development

Run the test suite:

```sh
make test
```

Run static checks:

```sh
make vet
```

## Homebrew

`codex-switch` is published through the `anemoris/tap` Homebrew tap:

```sh
brew install anemoris/tap/codex-switch
codex-switch doctor
```

Common lifecycle commands:

```sh
brew upgrade anemoris/tap/codex-switch
brew uninstall codex-switch
```

Notes:

- The tap repository is `anemoris/homebrew-tap`.
- `brew uninstall codex-switch` removes the Homebrew-managed binary.
- It does not remove shell initialization snippets or user data under `~/.codex-switch` or `CODEX_SWITCH_HOME`.
- Use `codex-switch cleanup` to remove shell integration, and `codex-switch cleanup --purge-data` if you also want to delete managed data.

## Quick Start

A good first run looks like this:

```sh
go build -o codex-switch ./cmd/codex-switch

codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

If you want `cwork .` to work in the current shell right away:

```sh
eval "$(codex-switch aliases --shell zsh)"
cwork .
```

If you want shortcuts to load automatically in future terminals:

```sh
codex-switch init-shell --shell zsh
source ~/.zshrc
cwork .
```

Keep these details in mind:

- `add --shortcut` only stores the shortcut name in config.
- It does not create a command in the terminal that is already open.
- `init-shell` affects future shells. Your current shell still needs `source ~/.zshrc` or a new terminal.
- A shortcut alias automatically prefixes `codex`, so `cwork .` is equivalent to `codex-switch run work -- codex .`.

## Commands

```sh
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
```

## Examples

Add two isolated profiles:

```sh
codex-switch add work --shortcut cwork --default
codex-switch add personal --shortcut cpersonal
```

Log into a profile directly:

```sh
codex-switch login work
codex-switch login personal -- codex login
codex-switch login personal --copy-from-current
```

Import an existing auth file into a profile:

```sh
codex-switch import-auth work --from ~/.codex/auth.json
codex-switch import-auth personal --from /path/to/old-codex-home
```

Run Codex under a profile:

```sh
codex-switch run work -- codex .
codex-switch run personal -- codex chat
codex-switch run -- codex .   # uses default profile
```

Inspect configured profiles:

```sh
codex-switch list
codex-switch show work
codex-switch status
```

When auth data is present, `list` and `show` include the bound account summary directly, including fields such as `auth_email`, `auth_account_id`, and `auth_name`. That makes it much easier to confirm that two profiles really point to different accounts.

Export a profile into the current shell:

```sh
eval "$(codex-switch env work)"
codex .
```

Generate shortcut aliases:

```sh
eval "$(codex-switch aliases --shell zsh)"

cwork .
cpersonal chat
```

These shortcuts pass the remaining arguments straight to `codex`, so forms like `cwork .` and `cpersonal chat` work as expected.

Install the alias loader into your shell rc file:

```sh
codex-switch init-shell --shell zsh
codex-switch init-shell --shell bash
codex-switch init-shell --shell fish
```

Remove the managed shell snippet:

```sh
codex-switch uninit-shell --shell zsh
codex-switch uninit-shell --shell bash
codex-switch uninit-shell --shell fish
```

Clean shell integration and optionally remove stored data:

```sh
codex-switch cleanup --shell zsh
codex-switch cleanup --shell zsh --purge-data
```

Run a quick environment check:

```sh
codex-switch doctor
codex-switch doctor --json
```

`doctor` also prints the current auth email, account ID, and display name for each profile. That is handy when you are trying to spot account mix-ups or a bad default profile quickly.

## Typical Workflows

Create a new isolated profile and log in from scratch:

```sh
codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

Reuse the current account without opening a new login flow:

```sh
codex-switch add personal --shortcut cpersonal
codex-switch login personal --copy-from-current
codex-switch run personal -- codex .
```
