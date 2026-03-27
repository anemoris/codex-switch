# codex-switch

[English](./README.md) | 简体中文 | [日本語](./README.ja.md) | [Español](./README.es.md) | [한국어](./README.ko.md)

用独立的 `CODEX_HOME` 目录，把多个 Codex 账号分开管理。

思路很直接：

- 每个账号都有自己单独的 `CODEX_HOME`
- `codex-switch` 会用你选中的 profile 目录启动 `codex`
- shortcut 通过 shell alias 提供，不需要去改共享的 `~/.codex/auth.json`

## 为什么这样做

把 `CODEX_HOME` 分开后，`sessions`、usage 元数据和认证快照就不会互相串在一起。行为也更好理解：当前用的是哪个账号，只取决于启动 `codex` 时用了哪个 `CODEX_HOME`。

## 项目结构

CLI 采用比较常见的 Go 项目结构：

```text
cmd/codex-switch/      # 二进制入口
internal/cli/          # 命令解析与用户侧行为
internal/config/       # 配置模型与持久化
internal/profile/      # CODEX_HOME 和 auth.json 操作
internal/runner/       # 外部命令执行抽象
internal/shellinit/    # shell 初始化片段安装/卸载
```

## 安装

最省事的方式是直接用仓库自带的 `Makefile`：

```sh
make build
```

这样会自动注入 `dev` 版本号。想指定版本号的话：

```sh
make build VERSION=v1.0.0
```

也可以直接用 `go build`：

```sh
go build -o codex-switch ./cmd/codex-switch
```

如果你是从源码构建，之后想直接运行 `codex-switch`，可以把二进制移到已经在 `PATH` 里的目录，或者直接用 `./codex-switch` 运行。

## 测试与开发

运行测试：

```sh
make test
```

运行静态检查：

```sh
make vet
```

## Homebrew

如果你准备用 Homebrew 分发这个 CLI，常见流程大概是这样：

```sh
brew install <tap>/codex-switch
codex-switch doctor
```

常用生命周期命令：

```sh
brew upgrade codex-switch
brew uninstall codex-switch
```

说明：

- 把 `<tap>` 换成你实际发布的 tap 名称，比如 `your-org/tap`。
- `brew uninstall codex-switch` 只会删掉 Homebrew 管理的二进制。
- 它不会删除 shell 初始化片段，也不会删除 `~/.codex-switch` 或 `CODEX_SWITCH_HOME` 下的数据。
- 想移除 shell 集成，用 `codex-switch cleanup`；连受管数据一起删，用 `codex-switch cleanup --purge-data`。

## 快速开始

第一次使用时，通常这样就够了：

```sh
go build -o codex-switch ./cmd/codex-switch

codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

如果你想让 `cwork .` 立刻在当前 shell 里可用：

```sh
eval "$(codex-switch aliases --shell zsh)"
cwork .
```

如果你想让 shortcut 在以后新开的终端里自动生效：

```sh
codex-switch init-shell --shell zsh
source ~/.zshrc
cwork .
```

这里有几件事要注意：

- `add --shortcut` 只是把 shortcut 名字写进配置。
- 它不会让当前已经打开的终端立刻多出这个命令。
- `init-shell` 影响的是后续新开的 shell，当前 shell 还是需要 `source ~/.zshrc` 或重新开一个终端。
- shortcut alias 会自动补上 `codex`，所以 `cwork .` 等价于 `codex-switch run work -- codex .`。

## 命令

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

## 示例

添加两个独立 profile：

```sh
codex-switch add work --shortcut cwork --default
codex-switch add personal --shortcut cpersonal
```

直接在某个 profile 里登录：

```sh
codex-switch login work
codex-switch login personal -- codex login
codex-switch login personal --copy-from-current
```

把已有 `auth.json` 导入某个 profile：

```sh
codex-switch import-auth work --from ~/.codex/auth.json
codex-switch import-auth personal --from /path/to/old-codex-home
```

在指定 profile 下运行 Codex：

```sh
codex-switch run work -- codex .
codex-switch run personal -- codex chat
codex-switch run -- codex .   # 使用默认 profile
```

查看当前配置：

```sh
codex-switch list
codex-switch show work
codex-switch status
```

如果已经有认证信息，`list` 和 `show` 会直接带出 profile 绑定的账号摘要，比如 `auth_email`、`auth_account_id`、`auth_name`。这样一眼就能看出两个 profile 到底是不是连到了不同账号。

把某个 profile 导出到当前 shell：

```sh
eval "$(codex-switch env work)"
codex .
```

生成 shortcut alias：

```sh
eval "$(codex-switch aliases --shell zsh)"

cwork .
cpersonal chat
```

这些 shortcut 会把后续参数原样传给 `codex`，所以 `cwork .`、`cpersonal chat` 这类写法都能直接用。

把 alias 自动加载片段写入 shell rc 文件：

```sh
codex-switch init-shell --shell zsh
codex-switch init-shell --shell bash
codex-switch init-shell --shell fish
```

移除受管的 shell 片段：

```sh
codex-switch uninit-shell --shell zsh
codex-switch uninit-shell --shell bash
codex-switch uninit-shell --shell fish
```

清理 shell 集成，并可选删除数据目录：

```sh
codex-switch cleanup --shell zsh
codex-switch cleanup --shell zsh --purge-data
```

执行环境检查：

```sh
codex-switch doctor
codex-switch doctor --json
```

`doctor` 也会列出每个 profile 当前绑定的邮箱、账号 ID 和显示名称，排查登录串号或者默认 profile 设错时会方便很多。

## 典型使用流程

创建新的独立 profile 并重新登录：

```sh
codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

复用当前账号，不重新打开登录流程：

```sh
codex-switch add personal --shortcut cpersonal
codex-switch login personal --copy-from-current
codex-switch run personal -- codex .
```
