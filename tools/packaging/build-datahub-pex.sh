#!/usr/bin/env bash

set -euo pipefail

python3 --version
rm -rf .venv-pex
python3 -m venv .venv-pex
# shellcheck disable=SC1091
source .venv-pex/bin/activate
pip install --upgrade pip
time pip install 'acryl-datahub[postgres,redshift]==0.13.1.2'
REQS=$(pip freeze)
pip install 'pex==2.3.0'
pex --version
# shellcheck disable=SC2086
time pex $REQS -o bin/datahub-ingester.pex -P userclouds@src/python -m userclouds.datahub_ingester
ls -lah bin/
