#!/usr/bin/env bash

set -euo pipefail

# Creates a virtual env under the python SDK dir and installs the SDK in editable mode into that virtual env.
VENV_BASE_DIR=public-repos/sdk-python/ ./tools/ensure-python-venv.sh
public-repos/sdk-python/.venv/bin/pip3 install -e public-repos/sdk-python
