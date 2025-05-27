#!/usr/bin/env bash

set -euo pipefail

PYTHON_SDK_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
python3 -m venv "${PYTHON_SDK_DIR}/.venv"
# shellcheck disable=SC1091
source "${PYTHON_SDK_DIR}/.venv/bin/activate"
pip install --upgrade pip
pip install -r "${PYTHON_SDK_DIR}/requirements-dev.txt"
pip install -e "${PYTHON_SDK_DIR}"
