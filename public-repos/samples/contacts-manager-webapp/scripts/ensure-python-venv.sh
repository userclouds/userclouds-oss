#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

PYTHON3="$(command -v python3)"

PYTHON_VIRTUALENV="${VENV_BASE_DIR:-}.venv"
REQUIREMENTS_FINGERPRINT="$(git hash-object -t blob requirements-dev.txt requirements.txt)"
VENV_FINGERPRINT_FILE="${PYTHON_VIRTUALENV}/.fingerprint-${REQUIREMENTS_FINGERPRINT}"

if [ -e "${VENV_FINGERPRINT_FILE}" ]; then
  echo "Virtualenv fingerprint matches, skipping virtualenv creation"
else
  PYTHON_VERSION="$("${PYTHON3}" --version)"
  echo "Installing ${PYTHON_VERSION} virtualenv"
  rm -rf "${PYTHON_VIRTUALENV}"
  "${PYTHON3}" -m venv "${PYTHON_VIRTUALENV}"
  "${PYTHON_VIRTUALENV}/bin/pip" install --upgrade pip
  "${PYTHON_VIRTUALENV}/bin/pip" install -r requirements-dev.txt
  "${PYTHON_VIRTUALENV}/bin/pip" install -r requirements.txt
  touch "${VENV_FINGERPRINT_FILE}"
fi


