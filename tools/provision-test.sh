#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

echo "Running provision --simulate against all JSON provisioning files..."

# thank you shellcheck SC2044 for this enlightening syntax :)
# specifically if you hadn't already reset IFS (like we do), you would get
# unexpected behavior on filenames with spaces. -print0 on find causes null byte
# separators which read handles with IFS= (implied null).
for UCENV in "dev" "debug" "staging" "prod"; do
  while IFS= read -r -d '' filename; do
    object_type="$(echo "$filename" | sed -E 's/^.*\/([^\/\._]+).*.json/\1/')"
    echo "UC_UNIVERSE=$UCENV bin/provision --simulate provision $object_type $filename"
    UC_CONFIG_DIR=./config,./helm/userclouds/base_configs UC_UNIVERSE=$UCENV bin/provision --simulate provision "$object_type" "$filename"
  done < <(find config/provisioning/$UCENV -type f -name "*.json" -print0)
done

while IFS= read -r -d '' filename; do
  object_type="$(echo "$filename" | sed -E 's/^.*\/([^\/\._]+).*.json/\1/')"
  echo "UC_UNIVERSE=dev bin/provision --simulate provision $object_type $filename"
  UC_UNIVERSE=dev bin/provision --simulate provision "$object_type" "$filename"
done < <(find config/provisioning/samples -type f -name "*.json" -print0)
