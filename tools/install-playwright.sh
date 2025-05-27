#!/usr/bin/env bash

# this script exists (for now) to ensure we install this dependency in dev and CI, but
# not in the build system / staging / prod (that runs AL and doesn't work easily)

set -euo pipefail
IFS=$'\n\t'

set +e
EXISTS=$(grep "Amazon Linux" /etc/os-release)
set -e

# -z (roughly) means "is empty", so if we're not on Amazon Linux, we install playwright
if [ -z "$EXISTS" ]; then
  echo "Installing browser(s)"
  npx playwright install --with-deps chromium
fi
