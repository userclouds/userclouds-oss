#!/usr/bin/env bash

set -euo pipefail

PYTHON_SDK_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
# shellcheck disable=SC1091
source "${PYTHON_SDK_DIR}/.venv/bin/activate"
find "${PYTHON_SDK_DIR}/src" -print0 -type f -name "*.py" | xargs pyupgrade --py39-plus
black "${PYTHON_SDK_DIR}/src"
isort --profile black "${PYTHON_SDK_DIR}/src"
