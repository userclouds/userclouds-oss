#!/usr/bin/env bash

# This exists to clean up the test database after the tests have run.
# It's currently used only by VSCode .vscode/tasks.json
# Not user-facing! Run `make test` instead.

docker rm -f testdb
rm -rf "$(cat /tmp/vscode_testdb)"
