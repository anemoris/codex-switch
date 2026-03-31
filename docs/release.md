# codex-switch 发布流程

这份文档面向项目维护者，覆盖三件事：

- 如何初始化和维护 Homebrew tap
- 如何通过 Git tag 驱动版本
- 如何完成一次完整 release

本文按 2026-03-21 查阅的 Homebrew 官方文档整理，适用于当前仓库结构：

```text
cmd/codex-switch/
internal/...
go.mod
LICENSE
README*.md
```

当前项目已经包含基于 Git tag 的自动化发布流水线。默认路径是推送 `vX.Y.Z` tag，由源码仓库里的 workflow 自动完成这些动作：

- 运行 `go vet ./...`
- 运行 `go test ./...`
- 创建同名 GitHub Release
- 计算源码 tarball 的 SHA256
- 更新 `homebrew-tap` 里的 `Formula/codex-switch.rb`

如果自动化失败，再走本文后半部分的手动兜底流程。

## 约定

建议使用下面这套仓库命名：

- 源码仓库：`YOUR_GITHUB_ORG/codex-switch`
- Homebrew tap 仓库：`YOUR_GITHUB_ORG/homebrew-tap`

如果 tap 仓库名是 `homebrew-tap`，用户侧常见安装方式会是：

```sh
brew tap YOUR_GITHUB_ORG/tap
brew install YOUR_GITHUB_ORG/tap/codex-switch
```

如果没有重名 formula，有时也可以直接：

```sh
brew install codex-switch
```

为了避免歧义，发布文档和 README 里建议始终写完整安装名：

```sh
brew install YOUR_GITHUB_ORG/tap/codex-switch
```

## 一次性准备

### 1. 创建 tap 仓库

先在 GitHub 创建空仓库：

```text
YOUR_GITHUB_ORG/homebrew-tap
```

然后本地初始化：

```sh
brew tap-new YOUR_GITHUB_ORG/homebrew-tap
```

这会生成一个标准 tap 结构。后续至少会用到：

```text
Formula/codex-switch.rb
```

### 2. 准备 formula

`codex-switch` 是 Go CLI，当前结构下推荐直接从源码 tarball 构建，而不是手工上传单独二进制。

初始 formula 可以参考：

```ruby
class CodexSwitch < Formula
  desc "Manage multiple Codex accounts with isolated CODEX_HOME directories"
  homepage "https://github.com/YOUR_GITHUB_ORG/codex-switch"
  url "https://github.com/YOUR_GITHUB_ORG/codex-switch/archive/refs/tags/v0.1.0.tar.gz"
  sha256 "REPLACE_WITH_SOURCE_TARBALL_SHA256"
  license "MIT"

  depends_on "go" => :build

  def install
    system "go", "build",
           "-ldflags", "-s -w -X main.version=#{version}",
           "-o", bin/"codex-switch",
           "./cmd/codex-switch"
  end

  test do
    ENV["CODEX_SWITCH_HOME"] = testpath/".codex-switch"
    system bin/"codex-switch", "add", "test", "--default"
    output = shell_output("#{bin}/codex-switch list")
    assert_match "test [default, auth=missing]", output
    assert_path_exists testpath/".codex-switch/config.json"
  end
end
```

这份测试故意不依赖 `codex` 本体，也不要求登录交互，更适合 Homebrew 的 `test do` 环境。

### 3. 配置自动化发布 secret

源码仓库里的 release workflow 需要能写入 tap 仓库，所以还要在源码仓库中配置一个 secret：

```text
HOMEBREW_TAP_TOKEN
```

这个 token 至少需要对 `YOUR_GITHUB_ORG/homebrew-tap` 具备 `contents: write` 权限。  
如果你使用 fine-grained personal access token，最小可行范围通常是：

- Repository access: `YOUR_GITHUB_ORG/homebrew-tap`
- Repository permissions:
  - Contents: Read and write

如果源码仓库和 tap 仓库不在同一个 owner 下，或者 tap 仓库名不是 `homebrew-tap`，需要同步修改源码仓库里的：

```text
.github/workflows/release.yml
```

### 4. 本地验证 tap

在 tap 仓库里保存 formula 后，先本地验证：

```sh
brew install --build-from-source YOUR_GITHUB_ORG/tap/codex-switch
brew test YOUR_GITHUB_ORG/tap/codex-switch
```

如果需要卸载重试：

```sh
brew uninstall codex-switch
```

## 版本策略

建议使用语义化版本，并统一使用带 `v` 前缀的 Git tag：

```text
v0.1.0
v0.1.1
v0.2.0
v1.0.0
```

推荐的 bump 原则：

- `patch`：bugfix、文档补充、小型内部重构，不改变用户命令语义
- `minor`：新增命令、新增 flag、新增非破坏性行为
- `major`：删除命令、改默认行为、改配置结构、引入不兼容变更

当前项目不维护单独的版本文件。版本来源只有一个：Git tag。

当前实现的约定是：

- 非 tag 构建默认显示 `dev`
- 如果当前 commit 正好落在 `vX.Y.Z` tag 上，`make build` 会自动把这个 tag 注入到二进制
- `codex-switch version` 会把语义化版本 tag 前缀里的 `v` 去掉，所以最终输出是 `0.1.0` 这种形式
- Homebrew formula 也通过构建参数注入版本，因此 Homebrew 安装后的 `codex-switch version` 会和 tag 对应

这套方式的好处是：

- 不需要维护第二份版本号
- 不会出现“代码里版本号没改，但 tag 已经变了”的双写问题
- 自动化工作流可以直接从 tag 派生 Release 和 Homebrew formula

## Release 前检查

正式发版前，至少做下面这些检查：

### 1. 确保工作区干净

```sh
git status --short
```

### 2. 跑测试

```sh
go test ./...
```

### 3. 检查 README 和发布文档

至少确认这些内容没有落后于实际命令：

- `README.md`
- `README.zh-CN.md`
- `README.ja.md`
- `README.es.md`
- `README.ko.md`
- `docs/release.md`

### 4. 确认许可证和仓库信息

当前仓库已经有：

- `LICENSE`
- `go.mod`

发版前只需要确认 formula 里的 `homepage`、`url`、`license` 与仓库一致。

## 标准 Release 流程

下面是当前项目推荐的自动化流程。

### 1. 完成代码与文档改动

在源码仓库中完成本次 release 的所有改动，并提交：

```sh
git add .
git commit -m "Prepare release vX.Y.Z"
```

### 2. 创建并推送 tag

```sh
git tag vX.Y.Z
git push origin main
git push origin vX.Y.Z
```

如果你的默认分支不是 `main`，替换成实际分支名即可。

### 3. 等待 release workflow 完成

源码仓库里的：

```text
.github/workflows/release.yml
```

会在 `vX.Y.Z` tag 推送后自动执行，完成这些动作：

- 跑 `go vet ./...`
- 跑 `go test ./...`
- 用 tag 版本构建一次二进制
- 如果 GitHub Release 不存在，则自动创建
- 下载源码 tarball 并计算 SHA256
- 更新 `YOUR_GITHUB_ORG/homebrew-tap` 里的 `Formula/codex-switch.rb`
- 提交并推送 tap 改动

如果 workflow 失败，先看失败点：

- 测试失败：先修源码仓库
- `HOMEBREW_TAP_TOKEN` 缺失：补 secret
- tap 推送失败：检查 token 权限或 tap 仓库名

### 4. 验证最终安装路径

workflow 成功后，至少再从用户视角验证一次：

```sh
brew tap YOUR_GITHUB_ORG/tap
brew install YOUR_GITHUB_ORG/tap/codex-switch
codex-switch version
codex-switch doctor
```

如果当前机器上之前装过源码版，最好再确认一下命中路径：

```sh
which codex-switch
```

## 手动兜底流程

如果自动化 workflow 因为 token、权限或仓库结构问题失败，可以按下面的手动流程兜底。

### 1. 创建 GitHub Release

在 GitHub 上基于 `vX.Y.Z` 创建 Release。  
即使你不上传额外二进制，GitHub 也会为 tag 提供源码 tarball，可以直接给 Homebrew formula 使用。

### 2. 计算源码 tarball 的 SHA256

下载 GitHub 自动生成的源码 tarball 并计算 SHA256：

```sh
curl -L -o codex-switch-vX.Y.Z.tar.gz \
  https://github.com/YOUR_GITHUB_ORG/codex-switch/archive/refs/tags/vX.Y.Z.tar.gz

shasum -a 256 codex-switch-vX.Y.Z.tar.gz
```

### 3. 更新 tap 中的 formula

编辑 `YOUR_GITHUB_ORG/homebrew-tap` 仓库里的：

```text
Formula/codex-switch.rb
```

至少更新这两个字段：

```ruby
url "https://github.com/YOUR_GITHUB_ORG/codex-switch/archive/refs/tags/vX.Y.Z.tar.gz"
sha256 "NEW_SHA256"
```

如果 release 包含许可证、描述、测试方式变化，也同步调整 formula。

### 4. 本地验证 formula

在更新过的 tap 仓库里执行：

```sh
brew uninstall codex-switch
brew install --build-from-source YOUR_GITHUB_ORG/tap/codex-switch
brew test YOUR_GITHUB_ORG/tap/codex-switch
```

建议再做一次最小烟雾测试：

```sh
codex-switch doctor
codex-switch add smoke --default
codex-switch list
brew uninstall codex-switch
```

如果你不想污染本机默认目录，可以临时指定：

```sh
CODEX_SWITCH_HOME="$(mktemp -d)" codex-switch doctor
```

### 5. 提交 tap 改动

在 tap 仓库中提交并推送：

```sh
git add Formula/codex-switch.rb
git commit -m "codex-switch vX.Y.Z"
git push origin main
```

### 6. 验证最终安装路径

最好在另一台机器、另一个 shell 环境，或至少一个干净环境里做最终检查：

```sh
brew tap YOUR_GITHUB_ORG/tap
brew install YOUR_GITHUB_ORG/tap/codex-switch
codex-switch doctor
```

如果 README 里写的是：

```sh
brew install YOUR_GITHUB_ORG/tap/codex-switch
```

那你就必须按这个路径亲自验证一次，避免发布后才发现 tap 名或 formula 名写错。

## 快速版本升级清单

如果自动化已经配置好，后续版本升级通常只需要这几步：

1. 在源码仓库完成改动并跑 `go test ./...`
2. 创建并推送 tag：`vX.Y.Z`
3. 等待 `release.yml` 完成
4. 执行一次 `brew install YOUR_GITHUB_ORG/tap/codex-switch`
5. 执行一次 `codex-switch version` / `codex-switch doctor`

## 常见问题

### 1. 为什么 formula 不直接装预编译二进制

当前项目最简单、最稳的路线是“从源码构建”：

- tap 更容易维护
- 不需要先搭建多平台二进制产物流程
- Go CLI 的构建成本通常可接受

如果以后你想缩短用户安装时间，可以再增加 bottles 或 GitHub Release 二进制资产流程，但那是下一阶段优化，不是现在的必需项。

### 2. 为什么 `test do` 不用 `--help` 或 `--version`

Homebrew 官方更鼓励测试“实际可运行的基本功能”，而不是只检查帮助信息。  
对 `codex-switch` 来说，`add` + `list` 是比 `--help` 更有信号的最小测试。

### 3. `remove` 会不会删除用户数据

不会。当前实现里，`remove` 只删除配置项，不会删除 profile 的 `CODEX_HOME` 目录。  
因此 formula 测试也不应该假设 `remove` 会清理磁盘目录。

### 4. `cleanup --purge-data` 会不会删除所有 profile

不会。它只删除 `codex-switch` 的根目录，也就是默认的 `~/.codex-switch` 或 `CODEX_SWITCH_HOME` 指向的目录。  
如果某个 profile 是用 `--home /absolute/path` 建在根目录外面，仍然需要手工处理。

## 推荐的目录与职责

为了降低维护成本，建议把职责分成两块：

### 源码仓库 `codex-switch`

负责：

- 代码
- README
- 发布说明
- Git tag
- GitHub Release

### tap 仓库 `homebrew-tap`

负责：

- `Formula/codex-switch.rb`
- 版本升级后的 `url` / `sha256`
- Homebrew 安装入口

这样做的好处是：

- 源码和分发职责清晰分离
- 不会把 tap 的提交历史和源码历史混在一起
- 后续如果你还要分发别的 CLI，也能继续复用同一个 tap

## 后续可选优化

等手动流程稳定后，可以再考虑自动化：

- GitHub Actions 自动跑 `go test ./...`
- GitHub Actions 在打 tag 后自动创建 Release
- 自动生成并提交 tap formula 更新 PR
- 后续再增加 bottles 或预编译二进制资产

当前项目已经有 tag 驱动的 release 自动化。后续如果你还想继续减少手工步骤，可以再考虑：

- 自动在 tap 仓库里创建 PR，而不是直接推送 `main`
- 自动构建并发布 bottle
- 自动附带跨平台二进制 release asset

## 官方参考

- Homebrew Taps:
  https://docs.brew.sh/Taps
- How to Create and Maintain a Tap:
  https://docs.brew.sh/How-to-Create-and-Maintain-a-Tap
- Formula Cookbook:
  https://docs.brew.sh/Formula-Cookbook
