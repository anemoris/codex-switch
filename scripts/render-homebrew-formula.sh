#!/usr/bin/env bash

set -euo pipefail

if [ "$#" -ne 4 ]; then
  echo "usage: $0 <owner> <repo> <tag> <sha256>" >&2
  exit 1
fi

owner="$1"
repo="$2"
tag="$3"
sha256="$4"

cat <<EOF
class CodexSwitch < Formula
  desc "Manage multiple Codex accounts with isolated CODEX_HOME directories"
  homepage "https://github.com/${owner}/${repo}"
  url "https://github.com/${owner}/${repo}/archive/refs/tags/${tag}.tar.gz"
  sha256 "${sha256}"
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
EOF
