#!/usr/bin/env bash

set -eou pipefail
IFS=$'\n\t'

# this doesn't make sense in CodeBuild because we aren't operating on a single commit, so just bail
if [[ -v CODEBUILD_BUILD_ID ]]; then
  exit 0
fi

fail=0

set +e
golang=$(git diff --cached -- . ':(exclude)cmd/*' ':(exclude)public-repos/sdk-golang/samples/*' | grep -E "^\+[^\+]*fmt.Print")
set -e

if [[ -n $golang ]]; then
  fail=1
  echo "Did you mean to use fmt.Print*() instead of log.*? touch bypass_lint if you did"
fi

set +e
js=$(git diff --cached -- . | grep -E "^\+[^\+]*console.(log|info|error|trace)")
set -e

if [[ -n $js ]]; then
  fail=1
  echo "Did you mean to use console.log or equiv? touch bypass_lint if you did"
fi

if [ $fail -ne 0 ]; then
  exit 1
fi

exit 0
