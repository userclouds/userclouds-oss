#!/usr/bin/env bash

set -euo pipefail

# NB: we explicitly want space-based splitting for $@ in the for loop here
IFS=$' '

for recipe in "$@"; do
  if which "$recipe" >/dev/null 2>&1; then
    # Command already in PATH, no need to log
    :
  else
    # Check if brew has the recipe installed
    if ! brew list "$recipe" >/dev/null 2>&1; then
      echo "Missing required $recipe ... installing."
      brew install "$recipe"
    fi
  fi
done

target=$(cat .node-version)
if diff <(node -v) <(cat .node-version) >/dev/null; then
  echo "Correct NodeJS version."
else
  echo "Wrong NodeJS version. Installing $target"
  export N_PREFIX=$HOME/.n
  export PATH="$N_PREFIX/bin:$PATH"
  n "$target"
fi
