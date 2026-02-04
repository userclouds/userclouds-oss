#!/usr/bin/env bash

# this script exists (for now) to ensure we install this dependency in dev and CI, but
# not in the build system / staging / prod (that runs AL and doesn't work easily)

set -euo pipefail
IFS=$'\n\t'

set +e
EXISTS=$(grep "Amazon Linux" /etc/os-release 2>/dev/null)
set -e

# -z (roughly) means "is empty", so if we're not on Amazon Linux, we install playwright
if [ -z "$EXISTS" ]; then
  # Check if Playwright browsers are already installed
  # We look for the chromium_headless_shell directory that matches our Playwright version (1.48.1 = build 1169)
  PLAYWRIGHT_CACHE="${HOME}/Library/Caches/ms-playwright"
  if [ "$(uname)" = "Linux" ]; then
    PLAYWRIGHT_CACHE="${HOME}/.cache/ms-playwright"
  fi

  if [ -d "${PLAYWRIGHT_CACHE}/chromium_headless_shell-1169" ]; then
    echo "Playwright browsers already installed (found chromium_headless_shell-1169)"
  else
    echo "Installing Playwright browsers (chromium)..."
    yarn playwright install --with-deps chromium
    echo "Playwright browsers installed successfully"
  fi
fi
