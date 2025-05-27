#! /usr/bin/env python3

# This script syncs a portion of our codebase into a public repo and opens a PR for the changes in the public repo.

import argparse
import json
import os
import re
import subprocess
import sys
from contextlib import contextmanager
from typing import Callable


@contextmanager
def log_step(message):
    if os.getenv("GITHUB_ACTIONS"):
        print(f"::group::{message}")
    else:
        print(f"\033[93m{message}\033[00m")
    try:
        yield
    finally:
        if os.getenv("GITHUB_ACTIONS"):
            print("::endgroup::")


def sync_paths(
    source_paths: [str],
    dest_dir: str,
    delete=True,
    relative=False,
    list_sources_only=True,
):
    """
    Syncs the given source_paths into dest_dir.

    Directories will be recursively copied by default. Following rsync convention, the source path
    "./mydir" will be copied into dest_dir/mydir, whereas given the source path "./mydir", the files
    inside of that directory will be copied into "dest_dir" directly (with no "mydir" parent).

    * If delete is True (recommended), then any files in dest_dir that don't exist in source_paths will be deleted.
    * If relative is True, then files will be copied into dest_dir under their paths relative to the current directory (e.g. ./some/file.txt will be copied to dest_dir/some/file.txt). If relative is False, then files will be copied directly into dest_dir (e.g. ./some/file.txt will be copied to dest_dir/file.txt).
    """

    if list_sources_only:
        for path in source_paths:
            if os.path.isdir(path):
                print(os.path.join(path, "**"))
            else:
                print(path)
        return

    flags = [
        "-avh",
        # Don't delete .git directory in target repo
        "--exclude=.git",
        # Not really an important flag, but it's nice to see which files are being copied that
        # actually changed. (The default timestamp diffing is not reliable because when cloning the
        # repo, the mod timestamps are all set to the current time.) Delete this if it proves to be
        # expensive
        "-c",
    ]
    if delete:
        flags.append("--delete")
    if relative:
        flags.append("--relative")
    cmd = ["rsync", *flags, *source_paths, dest_dir]
    print(" ".join(cmd))
    subprocess.check_call(cmd)


def sync_sdk(repo_name: str, public_repo_dir: str, list_sources_only: bool = False):
    """
    Sync files from public-repos/{repo_name} into the root of the public repo,
    as well as any files explicitly enumerated in
    public-repos/{repo_name}.rsync. Paths in the .rsync file will be passed to
    rsync as paths to sync (not used as a --files-from argument), so paths will
    be copied recursively, and rsync trailing slash semantics apply.
    """
    # Sync files from public-repos/{repo_name} into the root of the public
    # repo, without keeping the public-repos/{repo_name}/ directory structure
    # (relative=False)
    sync_paths(
        [f"public-repos/{repo_name}/"],
        public_repo_dir,
        relative=False,
        list_sources_only=list_sources_only,
    )

    # Try syncing additional paths
    try:
        with open(f"public-repos/{repo_name}.rsync") as fl:
            lines = (ln.strip() for ln in fl.readlines())
            sync_paths(
                [ln for ln in lines if ln and not ln.startswith("#")],
                public_repo_dir,
                # Preserve the relative file path structure from our monorepo
                # (i.e. don't sync all these files directly into the root of
                # the public repo without directory structure)
                relative=True,
                # Don't delete the files we just synced
                delete=False,
                list_sources_only=list_sources_only,
            )
    except FileNotFoundError:
        # We may not have a .rsync file for this repo. That's fine.
        pass

    # Copy over LICENSE file into root of repo
    sync_paths(
        ["legal/licenses/sdk/LICENSE"],
        public_repo_dir,
        list_sources_only=list_sources_only,
    )


def sync_ucconfig(
    repo_name: str, public_repo_dir: str, list_sources_only: bool = False
):
    """
    Sync ucconfig from public-repos/ucconfig and cmd/ucconfig. This is similar
    to sync_sdk, except this copies the files from cmd/ucconfig into the root
    of the public repo (rather than preserving the cmd/ucconfig prefix).
    """
    sync_paths(
        [f"public-repos/{repo_name}/", "cmd/ucconfig/", "legal/licenses/sdk/LICENSE"],
        public_repo_dir,
        relative=False,
        list_sources_only=list_sources_only,
    )


def sync_helm_charts(
    repo_name: str, public_repo_dir: str, list_sources_only: bool = False
):
    sync_paths(
        [f"public-repos/{repo_name}/", "legal/licenses/sdk/LICENSE"],
        public_repo_dir,
        relative=False,
        list_sources_only=list_sources_only,
    )


def sync_and_push(
    repo_name: str,
    public_repo_dir: str,
    sync_func: Callable[[str, str], None],
    reviewer_usernames: [str],
    no_push: bool = False,
):
    with log_step("Prepping public repo..."):
        # Ensure there are no uncommitted changes in the public repo
        if (
            subprocess.check_output(
                ["git", "status", "--porcelain"], cwd=public_repo_dir
            )
            .decode()
            .strip()
            != ""
        ):
            print(
                f"The public repo {public_repo_dir} has uncommitted changes. Please commit or stash them before running this script"
            )
            sys.exit(1)

        # Checkout the latest master for the public repo
        origin_info = subprocess.check_output(
            ["git", "remote", "show", "origin"], cwd=public_repo_dir
        ).decode()
        main_branch_name = (
            re.compile("HEAD branch: ([a-zA-Z0-9]+)").search(origin_info).group(1)
        )
        subprocess.check_output(
            ["git", "checkout", main_branch_name], cwd=public_repo_dir
        )
        subprocess.check_output(["git", "pull"], cwd=public_repo_dir)

    with log_step("Syncing to new branch..."):
        # Create new branch for these updates
        monorepo_sha = (
            subprocess.check_output(["git", "rev-parse", "HEAD"]).decode().strip()
        )
        new_branch_name = f"sync-{monorepo_sha}"
        subprocess.check_output(
            ["git", "checkout", "-B", new_branch_name], cwd=public_repo_dir
        )

        # Sync and push
        sync_func(repo_name, public_repo_dir)
        changed = (
            subprocess.check_output(
                ["git", "status", "--porcelain"], cwd=public_repo_dir
            )
            .decode()
            .strip()
        )
        if changed == "":
            print("No changes to commit or push")
            return
        subprocess.check_output(["git", "add", "."], cwd=public_repo_dir)
        subprocess.check_output(
            ["git", "commit", "-m", f"Sync monorepo state at {monorepo_sha}"],
            cwd=public_repo_dir,
        )

    if no_push:
        return

    with log_step("Pushing branch..."):
        subprocess.check_output(
            ["git", "push", "-f", "-u", "origin", new_branch_name], cwd=public_repo_dir
        )
        print(f"Pushed new branch {new_branch_name}")

    with log_step("Creating PR..."):
        # Get the name of the PR associated with the current HEAD commit, if
        # one exists. (Prefer using PR titles instead of commit titles here,
        # because when someone lands a PR with several commits without
        # squashing, CI will only run this script once, and it makes more sense
        # to use the PR title rather than the last-commit title)
        try:
            source_pr_info = subprocess.check_output(
                [
                    "gh",
                    "api",
                    f"/repos/userclouds/userclouds/commits/{monorepo_sha}/pulls",
                ]
            )
            prs = json.loads(source_pr_info)
            source_pr_name = prs[0]["title"] if prs else None
        except subprocess.CalledProcessError:
            source_pr_name = None

        # Open PR
        source_description = (
            f'"{source_pr_name}"' if source_pr_name else monorepo_sha[:10]
        )
        title = f"Sync monorepo state at {source_description}"
        body = f"Syncing from userclouds/userclouds@{monorepo_sha}"
        cmd = [
            "gh",
            "pr",
            "create",
            "-B",
            main_branch_name,
            "-H",
            new_branch_name,
            "-t",
            title,
            "-b",
            body,
            "-r",
            ",".join(reviewer_usernames),
        ]
        # Note: permission errors may manifest as 404s. Needs the full "repo" scope for classic tokens
        # and r/w "pull requests" scope for personal access tokens
        try:
            subprocess.check_output(
                cmd,
                cwd=public_repo_dir,
                env=dict(os.environ, **{"HUB_VERBOSE": "1"}),
                stderr=subprocess.STDOUT,
            )
        except subprocess.CalledProcessError as err:
            print(f"Error creating PR:\n{err.output.decode()}")
            sys.exit(1)
        print("Done!")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "target",
        choices=[
            "sdk-golang",
            "sdk-python",
            "sdk-typescript",
            "terraform-provider-userclouds",
            "ucconfig",
            "helm-charts",
        ],
    )
    parser.add_argument(
        "--reviewer",
        action="append",
        default=[],
        help="Github username of a reviewer for the PR. Can be specified multiple times",
    )
    parser.add_argument(
        "--list-sources-only",
        action="store_true",
        help="List the source file paths for the sync, and don't do anything besides that",
    )
    parser.add_argument(
        "--no-push",
        action="store_true",
        help="Sync to local dir only. Don't push the changes to the public repo",
    )
    args = parser.parse_args()

    # Check if we are running from userclouds
    if os.path.basename(os.getcwd()) != "userclouds":
        print("The script expects to be running from userclouds root")
        sys.exit(1)

    # Check if the target public repo directory is where we expect
    target_dir = os.path.join(os.path.dirname(os.getcwd()), args.target)
    if not os.path.exists(target_dir):
        print(
            f"Could not find {target_dir}. The script expects to be running with {args.target} being a peer directory"
        )
        sys.exit(1)

    if args.target in [
        "sdk-golang",
        "sdk-python",
        "sdk-typescript",
        "terraform-provider-userclouds",
    ]:
        sync_func = sync_sdk
    elif args.target == "ucconfig":
        sync_func = sync_ucconfig
    elif args.target == "helm-charts":
        sync_func = sync_helm_charts
    else:
        raise Exception(f"Unknown target {args.target}")

    if args.list_sources_only:
        sync_func(args.target, target_dir, list_sources_only=True)
    else:
        sync_and_push(
            args.target, target_dir, sync_func, args.reviewer, no_push=args.no_push
        )


if __name__ == "__main__":
    main()
