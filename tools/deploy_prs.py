#!/usr/bin/python

# sgarrity 6/23 - most of our deploy tooling has sort of evolved in shell scripts
# for no great reason other than laziness. asherf suggested moving towards python
# as an equally simple but safer / easier to read / maintain / etc alternative.
# This makes sense to me, so I started with this one as a test case. Leaving this
# note here on the "off chance" that someone in the future is reading this, we
# haven't migrated all of our tooling, and you're wondering what's "new" vs "old".

import json
import os
import subprocess
import sys


# Check if the Github CLI tool can auth with Github
def check_auth():
    exit_code, output = subprocess.getstatusoutput("gh auth status")
    if exit_code != 0:
        print("You need to authenticate with Github using the 'gh' CLI tool")
        print(
            "Run 'gh auth login --hostname github.com --git-protocol ssh' to authenticate"
        )
        exit(1)
    elif "protocol: https" in output.lower():
        print("You need to authenticate with Github using the 'gh' CLI tool and ssh")
        print("Run 'gh auth logout' to logout")
        print(
            "And then run 'gh auth login --hostname github.com --git-protocol ssh' to authenticate"
        )
        exit(1)


def get_author(pr_dict: dict) -> tuple[str, str]:
    author_dict = pr_dict["author"]
    is_bot = author_dict["is_bot"]
    if is_bot:
        return author_dict["login"], None
    name = author_dict["name"]
    login = author_dict["login"]
    formatted_name = f"{name} ({login})" if name else login
    return formatted_name, name or login


def get_prs(from_hash: str, to_hash: str) -> tuple[list[str], list[str]]:
    commits = subprocess.getoutput(
        f'git log --pretty=format:"%H" {from_hash}..{to_hash}'
    ).split("\n")
    # grab a list of recent PRs to try to match to commits
    pr_list_output = subprocess.getoutput(
        "gh pr list --state merged --base master --limit 100 --json number,title,author,mergeCommit"
    )
    prs = json.loads(pr_list_output)
    prs_maps = {}
    for pr_dict in prs:
        author, human = get_author(pr_dict)
        msg = f"#{pr_dict['number']} {pr_dict['title']} - {author}"
        mc = pr_dict["mergeCommit"]
        if mc:  # there are some cases where the mergeCommit is null
            prs_maps[mc["oid"]] = (msg, human)
    human_authors = {
        prs_maps[commitHash][1]
        for commitHash in commits
        if commitHash in prs_maps and prs_maps[commitHash][1]
    }
    return [
        prs_maps[commitHash][0] for commitHash in commits if commitHash in prs_maps
    ], sorted(human_authors)


if __name__ == "__main__":
    if len(sys.argv) < 3:
        print("Usage: deploy_prs.py <from hash> <to hash>")
        exit(1)

    from_hash = sys.argv[1]
    to_hash = sys.argv[2]
    check_auth()
    found_prs, human_authors = get_prs(from_hash, to_hash)
    if len(found_prs) > 0:
        print("Deploying PRs:")
        print("\n".join(found_prs))
        if human_authors:
            print("Following people have PRs in this deploy:")
            print("\n".join(human_authors))
    else:
        print("No PRs found, deploying commits:")
        os.system(f"git --no-pager log --oneline {from_hash}..{to_hash}")
