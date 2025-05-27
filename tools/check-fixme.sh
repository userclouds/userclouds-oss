#!/usr/bin/env bash

set -euo pipefail

RES=$(git grep -c "^+\s.*[F]IXME" ':!tools/check-fixme.sh') || true

if ((RES > 0)); then
  echo "Please resolve FIXMEs, or convert them to TODOs"
  git grep "^+\s.*[F]IXME" ':!tools/check-fixme.sh'
  exit 1
fi
