#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

./tools/ensure-python-venv.sh
VENV_BIN_DIR=".venv/bin"
VENV_FULL_PATH=$(realpath ${VENV_BIN_DIR})

for DIR in "./tools" "./src/python" "./public-repos/sdk-python/src" "./public-repos/samples/contacts-manager-webapp" "./public-repos/samples/add-user-data"; do
  echo "Running Black ${DIR}"
  "${VENV_FULL_PATH}/black" --check "${DIR}"
  pushd "${DIR}"
  echo "Running iSort ${DIR}"
  "${VENV_FULL_PATH}/isort" --check-only --profile black .
  popd
  echo "Running ruff linter ${DIR}"
  "${VENV_FULL_PATH}/ruff" check --ignore E501 $DIR
  echo "Running pyupgrade ${DIR}"
  find "${DIR}" -type f -name "*.py" | grep -v ".venv" | xargs "${VENV_FULL_PATH}/pyupgrade" --py39-plus
done
