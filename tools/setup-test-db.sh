#!/usr/bin/env bash

# Not user-facing! Run `make test` instead.
# takes one arg, which is a file to write the path to the test db to
# this is a bit roundabout but it keeps the code paths the same betweek Makefile and VSCode,
# since VSCode doesn't have a way to pass args between a pre-run task and the launch command
# (that I found in 10min of looking)

set -euo pipefail
IFS=$'\n\t'

docker run --rm -d -e POSTGRES_PASSWORD=mysecretpassword \
  --name testdb \
  --tmpfs /var/lib/postgresql/data:size=8G \
  -p 54321:5432 \
  postgres \
  -c max_connections=850

_TESTDB=$(mktemp -d)

# NB: keep these in sync with Makefile file paths in test target
_TESTDB_CONN=$_TESTDB/connfile

echo "postgres://postgres:mysecretpassword@127.0.0.1:54321/postgres?sslmode=disable" >"$_TESTDB_CONN"

echo "$_TESTDB" >"$1"
