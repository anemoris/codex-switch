# codex-switch

[English](./README.md) | [简体中文](./README.zh-CN.md) | [日本語](./README.ja.md) | [Español](./README.es.md) | 한국어

여러 Codex 계정을 각각 독립된 `CODEX_HOME` 디렉터리로 분리해서 관리합니다.

핵심은 단순합니다.

- 각 계정마다 전용 `CODEX_HOME` 을 둡니다
- `codex-switch` 가 선택한 프로필 디렉터리로 `codex` 를 실행합니다
- shortcut 은 공유 `~/.codex/auth.json` 을 건드리지 않고 shell alias 로 제공합니다

## 왜 이런 방식을 쓰는가

`CODEX_HOME` 을 분리하면 `sessions`, 사용량 메타데이터, 인증 스냅샷이 서로 섞이지 않습니다. 동작도 이해하기 쉽습니다. 어떤 계정이 활성화되는지는 `codex` 를 시작할 때 어떤 `CODEX_HOME` 을 썼는지로 결정됩니다.

## 프로젝트 구조

CLI 는 비교적 익숙한 Go 프로젝트 구조를 따릅니다.

```text
cmd/codex-switch/      # 바이너리 엔트리포인트
internal/cli/          # 명령 파싱 및 사용자 대상 동작
internal/config/       # 설정 모델과 영속화
internal/profile/      # CODEX_HOME 및 auth.json 작업
internal/runner/       # 외부 명령 실행 추상화
internal/shellinit/    # shell 초기화 스니펫 설치/제거
```

## 설치

Homebrew 로 설치:

```sh
brew install anemoris/tap/codex-switch
```

소스에서 직접 빌드하려면 저장소에 들어 있는 `Makefile` 을 쓰는 것이 가장 간단합니다.

```sh
make build
```

현재 commit 이 정확히 Git tag 위에 있으면 그 tag 가 빌드 버전으로 사용되고, 그렇지 않으면 `dev` 로 돌아갑니다. 특정 버전을 수동으로 지정해 빌드하려면:

```sh
make build VERSION=v1.0.0
```

원하면 `go build` 로 직접 빌드해도 됩니다.

```sh
go build -o codex-switch ./cmd/codex-switch
```

소스에서 빌드한 뒤 `codex-switch` 를 바로 실행하고 싶다면, 바이너리를 이미 `PATH` 에 잡혀 있는 디렉터리로 옮기거나 `./codex-switch` 로 실행하세요.

## 테스트/개발

테스트 실행:

```sh
make test
```

정적 검사 실행:

```sh
make vet
```

## Homebrew

`codex-switch` 는 `anemoris/tap` Homebrew tap 에 배포되어 있습니다.

```sh
brew install anemoris/tap/codex-switch
codex-switch doctor
```

자주 쓰는 라이프사이클 명령:

```sh
brew upgrade anemoris/tap/codex-switch
brew uninstall codex-switch
```

참고:

- tap 저장소 주소는 `anemoris/homebrew-tap` 입니다.
- `brew uninstall codex-switch` 는 Homebrew 가 관리하는 바이너리만 제거합니다.
- shell 초기화 스니펫이나 `~/.codex-switch`, `CODEX_SWITCH_HOME` 아래 데이터는 지우지 않습니다.
- shell 통합만 지우려면 `codex-switch cleanup` 을, 관리 데이터까지 지우려면 `codex-switch cleanup --purge-data` 를 사용하세요.

## 빠른 시작

처음 써볼 때는 보통 이 정도면 충분합니다.

```sh
go build -o codex-switch ./cmd/codex-switch

codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

현재 shell 에서 바로 `cwork .` 를 쓰고 싶다면:

```sh
eval "$(codex-switch aliases --shell zsh)"
cwork .
```

앞으로 여는 터미널에서 shortcut 이 자동으로 로드되게 하려면:

```sh
codex-switch init-shell --shell zsh
source ~/.zshrc
cwork .
```

여기만 기억하면 됩니다.

- `add --shortcut` 은 shortcut 이름을 설정에 저장만 합니다.
- 이미 열려 있는 터미널에 그 명령이 바로 생기지는 않습니다.
- `init-shell` 은 앞으로 열 shell 에만 적용됩니다. 현재 shell 은 여전히 `source ~/.zshrc` 나 새 터미널이 필요합니다.
- shortcut alias 는 자동으로 `codex` 를 앞에 붙이므로 `cwork .` 는 `codex-switch run work -- codex .` 와 같습니다.

## 명령

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

## 예시

격리된 두 개의 프로필 추가:

```sh
codex-switch add work --shortcut cwork --default
codex-switch add personal --shortcut cpersonal
```

프로필에 직접 로그인:

```sh
codex-switch login work
codex-switch login personal -- codex login
codex-switch login personal --copy-from-current
```

기존 인증 파일을 프로필로 가져오기:

```sh
codex-switch import-auth work --from ~/.codex/auth.json
codex-switch import-auth personal --from /path/to/old-codex-home
```

프로필 아래에서 Codex 실행:

```sh
codex-switch run work -- codex .
codex-switch run personal -- codex chat
codex-switch run -- codex .   # 기본 프로필 사용
```

설정된 프로필 확인:

```sh
codex-switch list
codex-switch show work
codex-switch status
```

인증 정보가 있으면 `list` 와 `show` 에 연결된 계정 요약도 함께 표시됩니다. `auth_email`, `auth_account_id`, `auth_name` 같은 필드를 바로 볼 수 있어서, 두 프로필이 실수로 같은 계정을 가리키고 있지는 않은지 확인하기 좋습니다.

현재 shell 에 프로필 내보내기:

```sh
eval "$(codex-switch env work)"
codex .
```

shortcut alias 생성:

```sh
eval "$(codex-switch aliases --shell zsh)"

cwork .
cpersonal chat
```

이 shortcut 들은 남은 인수를 그대로 `codex` 에 넘기므로 `cwork .` 이나 `cpersonal chat` 을 바로 사용할 수 있습니다.

shell rc 파일에 alias 로더 설치:

```sh
codex-switch init-shell --shell zsh
codex-switch init-shell --shell bash
codex-switch init-shell --shell fish
```

관리되는 shell 스니펫 제거:

```sh
codex-switch uninit-shell --shell zsh
codex-switch uninit-shell --shell bash
codex-switch uninit-shell --shell fish
```

shell 통합을 정리하고, 필요하면 저장된 데이터도 제거:

```sh
codex-switch cleanup --shell zsh
codex-switch cleanup --shell zsh --purge-data
```

빠르게 환경 점검:

```sh
codex-switch doctor
codex-switch doctor --json
```

`doctor` 는 각 프로필에 연결된 이메일, 계정 ID, 표시 이름도 보여줍니다. 계정이 뒤섞였는지 확인하거나 기본 프로필 설정 실수를 찾을 때 꽤 유용합니다.

## 대표적인 사용 흐름

새 격리 프로필을 만들고 처음부터 로그인:

```sh
codex-switch add work --shortcut cwork --default
codex-switch login work
codex-switch run work -- codex .
```

새 로그인 과정을 열지 않고 현재 계정 재사용:

```sh
codex-switch add personal --shortcut cpersonal
codex-switch login personal --copy-from-current
codex-switch run personal -- codex .
```
