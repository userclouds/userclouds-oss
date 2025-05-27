#! /usr/bin/env python3

# This script downloads the latest ucconfig version that has been released and
# runs its end-to-end tests to verify that it still works against today's
# UC backend.
#
# Run this script with USERCLOUDS_TENANT_URL, USERCLOUDS_CLIENT_ID, and
# USERCLOUDS_CLIENT_SECRET set.

import importlib.util
import json
import os
import platform
import shutil
import subprocess
import sys
import tempfile

import urllib3

if __name__ == "__main__":
    if not importlib.util.find_spec("yaml"):
        raise Exception("Please install PyYAML to run this script.")

    tmpdirname = tempfile.mkdtemp()
    print(f"NOTE: test files will be downloaded into {tmpdirname}.")
    try:
        print("Getting ucconfig latest release info...")
        # using urllib3 since boto3 already has a dependency on it
        # https://docs.github.com/en/rest/releases/releases#get-the-latest-release
        resp = urllib3.request(
            "GET", "https://api.github.com/repos/userclouds/ucconfig/releases/latest"
        )
        if resp.status != 200:
            raise Exception(f"Failed to get latest release info: {resp.status}")

        latest_release = json.loads(resp.data)
        latest_release_tag = latest_release["tag_name"]
        print(f"Testing ucconfig {latest_release_tag}")

        kernel = platform.system().lower()
        arch = platform.machine()
        bin_asset = next(
            (
                a
                for a in latest_release["assets"]
                if kernel in a["name"] and arch in a["name"]
            ),
            None,
        )
        if not bin_asset:
            raise Exception(f"No binary asset found for {kernel} {arch}")
        bin_url = bin_asset["browser_download_url"]
        bin_name = bin_asset["name"]
        print("Downloading ucconfig binary from ", bin_url)
        subprocess.check_output(
            ["curl", "--silent", "--location", "-o", bin_name, bin_url], cwd=tmpdirname
        )
        print(f"Unzipping ucconfig binary to {tmpdirname}/bin...")
        os.mkdir(f"{tmpdirname}/bin")
        subprocess.check_output(["tar", "-xzf", bin_name, "-C", "bin"], cwd=tmpdirname)

        print("Downloading ucconfig repo...")
        subprocess.check_output(
            ["git", "clone", "https://github.com/userclouds/ucconfig.git"],
            cwd=tmpdirname,
            stderr=subprocess.STDOUT,
        )
        repo_dir = tmpdirname + "/ucconfig"
        print(f"Checking out {latest_release_tag}...")
        subprocess.check_output(
            ["git", "checkout", latest_release_tag],
            cwd=repo_dir,
            stderr=subprocess.STDOUT,
        )

        e2e_script = repo_dir + "/e2e-test/test.py"
        print(f"Running end-to-end test using {e2e_script}...")
        child_path = os.environ.copy()
        child_path["PATH"] = f'{tmpdirname}/bin:{os.environ["PATH"]}'
        # Quick self-test: verify that running "ucconfig" will run the version
        # that we downloaded from GitHub
        ucconfig_path = (
            subprocess.check_output(["which", "ucconfig"], env=child_path)
            .decode()
            .strip()
        )
        if not ucconfig_path.startswith(tmpdirname):
            raise Exception(
                f'Running "ucconfig" is resolving to {ucconfig_path}, which is not the version we downloaded from GitHub. $PATH: {child_path["PATH"]}'
            )
        subprocess.check_call([e2e_script], cwd=repo_dir, env=child_path)
    except Exception:
        print(
            f"{sys.argv[0]}: An error occurred. Downloaded files are in {tmpdirname} for debugging"
        )
        raise
    shutil.rmtree(tmpdirname)
