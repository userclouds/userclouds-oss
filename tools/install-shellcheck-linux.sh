#!/usr/bin/env bash

set -euo pipefail
if [[ ! -f "bin/shellcheck" ]]; then
  mkdir -p bin
  # Updated .github/workflows/lint-shell.yml key for Cache shellcheck when
  curl -L https://github.com/koalaman/shellcheck/releases/download/v0.9.0/shellcheck-v0.9.0.linux.x86_64.tar.xz | tar --strip-components=1 -xJv -C bin/
else
  echo "shellcheck already installed"
fi
touch bin/shellcheck
