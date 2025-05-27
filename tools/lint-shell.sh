#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

shellscripts=$(echo "$1" | tr " " "\n")
echo "Checking shell formatting with shfmt..."
# this next comment specifically prevents shellcheck asking us to quote $shellscripts, which wouldn't work
# shellcheck disable=SC2086
SHSCS=$(bin/shfmt -i 2 -s -l -d $shellscripts)
test -z "$SHSCS" || (echo "shfmt needs running on:" && echo "$SHSCS" && exit 1)

# we specifically curl shellcheck and copy it to [repo root]/bin
echo "Linting shell scripts with shellcheck..."
# this next comment specifically prevents shellcheck asking us to quote $shellscripts, which wouldn't work
# shellcheck disable=SC2086
bin/shellcheck $shellscripts
