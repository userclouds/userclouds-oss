#!/usr/bin/env bash

set -euo pipefail
IFS=$'\n\t'

# if we need to use this outside of git (like old CI config), we can use tar + md5sum, but it's slow
# BEFORE=$(tar cf - . | md5sum)
# make codegen
# AFTER=$(tar cf - . | md5sum)
# ...

make codegen
RES=$(git status --porcelain)

if [ -n "$RES" ]; then
  echo "ERROR: running codegen changed something"
  echo "$RES"
  exit 1
fi

exit 0
