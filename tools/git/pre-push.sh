#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# save this for later because somehow `git lfs pre-push "$@"` breaks read otherwise
STDIN=$(cat -)

# check lfs
command -v git-lfs >/dev/null 2>&1 || {
  printf >&2 "\nThis repository is configured for Git LFS but 'git-lfs' was not found on your path. If you no longer wish to use Git LFS, remove this hook by deleting .git/hooks/pre-push.\n"
  exit 2
}
git lfs pre-push "$@"

# use a fail var to keep track of this so we can do cleanup (eg unstash) rather than immediate exits
fail=0

stashed=$(git stash --keep-index -u)
# stashed="No local changes to save." # for debugging

z40=0000000000000000000000000000000000000000

# remember where we started but avoid "HEAD" :)
changed_head=0
current_head=$(git rev-parse --symbolic-full-name HEAD)
if [ "$current_head" = "HEAD" ]; then
  current_head=$(git show --oneline -s | head -c 8)
fi
if [[ $current_head =~ "refs/heads/" ]]; then
  current_head=${current_head:11}
fi

IFS=' '
while read -r local_ref local_sha remote_ref remote_sha; do
  if [ "$local_ref" == "refs/heads/master" ] || [ "$remote_ref" == "refs/heads/master" ]; then
    echo "don't push directly to master"
    exit 1
  fi

  # silence shellcheck, but need to read these or local_sha is set incorrectly
  echo "$remote_ref $remote_sha" >/dev/null

  # don't do anything when deleting remotes
  if [ "$local_sha" = "$z40" ]; then
    continue
  fi

  # did we specify something other than HEAD to push?
  # note to future self: it seems that if you have a push target configured
  # (show at the bottom of git remote show origin), local_ref is empty if not explicitly
  # specified, but if there isn't a tracking branch, you get refs/head/[branch] even
  # if it's implicit
  if [ "$local_ref" != "" ]; then
    # figure out what the local reference points to, and if it's a tag, get the commit it references
    actual_local_sha=$(git rev-parse "$local_ref")
    if [[ $(git rev-parse --symbolic-full-name "$local_ref") =~ "refs/tags/" ]]; then
      actual_local_sha=$(git rev-list -n 1 "$local_ref") # deref the tag
    fi

    if [[ $actual_local_sha != $(git rev-parse HEAD) ]]; then
      # check out whatever we're actually pushing
      echo "Switching your working directory to your push refspec $local_ref"
      changed_head=1
      git checkout -q "$local_ref"
    fi
  fi

  # TODO all of these checks should probably be factored elsewhere to eg use in CI

  # any FIXMEs sneak in?
  if test -f bypass_fixme; then
    echo "bypass_fixme exists, skipping pre-push fixme check"
    rm bypass_fixme
  else
    echo "Checking for fixmes before pushing..."
    set +e
    make test-fixme
    RES=$?
    set -e
    [ $RES -ne 0 ] && fail=1 && echo "make test-fixme, please fix before pushing"
  fi

  if [ $fail -eq 0 ]; then
    if test -f bypass_gen; then
      echo "bypass_gen exists, skipping pre-push generate"
      rm bypass_gen
    else
      echo "Running make codegen before pushing..."
      # check to make sure we've run codegen
      set +e
      make test-codegen
      RES=$?
      set -e

      [ $RES -ne 0 ] && fail=1 && echo "running code-gen changed something, please fix before pushing"
    fi
  fi

  if [ $fail -eq 0 ]; then
    if test -f bypass_config; then
      echo "bypass_config exists, skipping pre-push config checks..."
      rm bypass_config
    else
      set +e
      make test-provisioning
      RES=$?
      set -e
      [ $RES -ne 0 ] && fail=1 && echo "make test-provisioning failed"
    fi
  fi

  # Note: we intentionally skip lint & tests here to save time, and let them fail in CI

  if [ $fail -eq 0 ]; then
    if test -f bypass_helm; then
      echo "bypass_helm exists; skipping helm tests"
      rm bypass_helm
    else
      echo "Running make test-helm before pushing..."
      set +e
      make test-helm
      RES=$?
      set -e
      [ $RES -ne 0 ] && fail=1 && echo "make test-helm failed"
    fi
  fi

done < <(echo "$STDIN")

if [ $changed_head -ne 0 ]; then
  # reset to where we were before
  echo "Resetting your working state to $current_head"
  git checkout "$current_head"
fi

# unstash if needed
if [ "$stashed" != "No local changes to save" ]; then
  git stash pop -q
fi

# stop the push if we failed anything
[ $fail -ne 0 ] && exit 1

exit 0
