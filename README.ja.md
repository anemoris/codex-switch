# codex-switch

[English](./README.md) | [简体中文](./README.zh-CN.md) | 日本語 | [Español](./README.es.md) | [한국어](./README.ko.md)

複数の Codex アカウントを、独立した `CODEX_HOME` ディレクトリごとに分けて管理します。

やっていることはシンプルです。

- 各アカウントに専用の `CODEX_HOME` を持たせる
- `codex-switch` が選んだ profile のディレクトリで `codex` を起動する
- shortcut は共有の `~/.codex/auth.json` を書き換えず、shell alias として用意する

## なぜこの方式なのか

`CODEX_HOME` を分けておけば、`sessions`、usage メタデータ、認証スナップショットが混ざりません。挙動もわかりやすくて、どのアカウントが有効になるかは `codex` を起動したときの `CODEX_HOME` で決まります。

## プロジェクト構成

CLI は比較的素直な Go プロジェクト構成です。

```text
cmd/codex-switch/      # バイナリのエントリポイント
internal/cli/          # コマンド解析とユーザー向け挙動
internal/config/       # 設定モデルと永続化
internal/profile/      # CODEX_HOME と auth.json の操作
internal/runner/       # 外部コマンド実行の抽象化
internal/shellinit/    # shell 初期化スニペットの追加/削除
```

## インストール

Homebrew でインストールする場合:

```sh
brew install anemoris/tap/codex-switch
```

ソースからビルドするなら、いちばん手軽なのは付属の `Makefile` を使う方法です。

```sh
make build
```

これで `dev` バージョン文字列が自動で埋め込まれます。特定のバージョンを指定したい場合は次の通りです。

```sh
make build VERSION=v1.0.0
```

もちろん `go build` で直接ビルドしてもかまいません。

```sh
go build -o codex-switch ./cmd/codex-switch
```

ソースからビルドしてそのまま `codex-switch` を使いたいなら、バイナリを `PATH` に入っているディレクトリへ置くか、`./codex-switch` として実行してください。

## テストと開発

テストを実行する:

```sh
make test
```

静的チェックを実行する:

```sh
make vet
```

## Homebrew

`codex-switch` は `anemoris/tap` の Homebrew tap で公開されています。

```sh
brew install anemoris/tap/codex-switch
codex-switch doctor
```

よく使うライフサイクルコマンド:

```sh
brew upgrade anemoris/tap/codex-switch
brew uninstall codex-switch
```

補足:

- tap リポジトリは `anemoris/homebrew-tap` です。
- `brew uninstall codex-switch` で削除されるのは Homebrew 管理下のバイナリだけです。
- shell 初期化スニペットや `~/.codex-switch`、`CODEX_SWITCH_HOME` 配下のデータは残ります。
- shell 連携だけ消すなら `codex-switch cleanup`、データも消すなら `codex-switch cleanup --purge-data` を使ってください。

## クイックスタート

最初に試すなら、まずはこれで十分です。

```sh
go build -o codex-switch ./cmd/codex-switch

codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

今の shell ですぐ `cwork .` を使いたい場合:

```sh
eval "$(codex-switch aliases --shell zsh)"
cwork .
```

今後開くターミナルで shortcut を自動読み込みしたい場合:

```sh
codex-switch init-shell --shell zsh
source ~/.zshrc
cwork .
```

ここだけは押さえておいてください。

- `add --shortcut` は shortcut 名を設定に保存するだけです。
- すでに開いている shell に、その場でコマンドが追加されるわけではありません。
- `init-shell` が効くのは今後の shell です。現在の shell では `source ~/.zshrc` か新しいターミナルが必要です。
- shortcut alias には自動で `codex` が付くので、`cwork .` は `codex-switch run work -- codex .` と同じです。

## コマンド

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

## 例

2 つの独立した profile を追加する:

```sh
codex-switch add work --shortcut cwork --default
codex-switch add personal --shortcut cpersonal
```

profile に直接ログインする:

```sh
codex-switch login work
codex-switch login personal -- codex login
codex-switch login personal --copy-from-current
```

既存の `auth.json` を profile に取り込む:

```sh
codex-switch import-auth work --from ~/.codex/auth.json
codex-switch import-auth personal --from /path/to/old-codex-home
```

指定 profile で Codex を実行する:

```sh
codex-switch run work -- codex .
codex-switch run personal -- codex chat
codex-switch run -- codex .   # デフォルト profile を使用
```

設定済み profile を確認する:

```sh
codex-switch list
codex-switch show work
codex-switch status
```

認証情報があれば、`list` と `show` には紐づいているアカウントの要約も表示されます。`auth_email`、`auth_account_id`、`auth_name` のような項目が出るので、2 つの profile が本当に別アカウントを向いているか確認しやすくなります。

現在の shell に profile を反映する:

```sh
eval "$(codex-switch env work)"
codex .
```

shortcut alias を生成する:

```sh
eval "$(codex-switch aliases --shell zsh)"

cwork .
cpersonal chat
```

これらの shortcut は残りの引数をそのまま `codex` に渡すので、`cwork .` や `cpersonal chat` をそのまま使えます。

shell rc ファイルに alias ローダーを追加する:

```sh
codex-switch init-shell --shell zsh
codex-switch init-shell --shell bash
codex-switch init-shell --shell fish
```

管理対象の shell スニペットを削除する:

```sh
codex-switch uninit-shell --shell zsh
codex-switch uninit-shell --shell bash
codex-switch uninit-shell --shell fish
```

shell 連携を片付けて、必要ならデータディレクトリも削除する:

```sh
codex-switch cleanup --shell zsh
codex-switch cleanup --shell zsh --purge-data
```

環境をざっと診断する:

```sh
codex-switch doctor
codex-switch doctor --json
```

`doctor` では、各 profile に紐づくメールアドレス、アカウント ID、表示名も確認できます。アカウントの取り違えやデフォルト profile の設定ミスを見つけるときに便利です。

## 典型的な使い方

新しい独立 profile を作って最初からログインする:

```sh
codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

新しいログインを開かず、現在のアカウントを再利用する:

```sh
codex-switch add personal --shortcut cpersonal
codex-switch login personal --copy-from-current
codex-switch run personal -- codex .
```
