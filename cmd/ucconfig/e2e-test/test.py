#! /usr/bin/env python3

# Note: you'll need to have ucconfig in $PATH to run this script.

import filecmp
import os
import re
import subprocess
import tempfile
import json
import yaml
import sys

from contextlib import contextmanager
from pathlib import Path
from urllib.parse import urlparse

TEST_DIR = os.path.dirname(__file__)

# https://stackoverflow.com/a/14693789
ANSI_ESCAPE_REGEXP = re.compile(r"\x1B(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])")
RESOURCE_DEFINITION_REGEXP = re.compile(r'TerraformTypeSuffix:\s+"([a-z0-9_]+)"')


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


class ChildProcessError(Exception):
    def __init__(self, output):
        self.output = output


def run_capturing_output(cmd):
    output = ""
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    for line in process.stdout:
        decoded = line.decode("utf-8")
        # Keep color codes in the output to the terminal, but strip them for the
        # captured output string so that we can search that string without color
        # codes messing up the match
        print(decoded.rstrip("\n"))
        output += ANSI_ESCAPE_REGEXP.sub("", decoded)
    if process.wait() != 0:
        raise ChildProcessError(output)
    return output


def validate_manifest(manifest_name):
    with open(os.path.join(TEST_DIR, manifest_name)) as f:
        if manifest_name.endswith(".json"):
            manifest = json.load(f)
        elif manifest_name.endswith(".yaml"):
            manifest = yaml.safe_load(f)
        else:
            raise Exception(f"Unknown manifest file extension: {manifest_name}")
    for resource in manifest["resources"]:
        if "<<TARGET_FQTN>>" not in resource["resource_uuids"]:
            raise Exception(
                f'<<TARGET_FQTN>> not found in resource_uuids for resource {resource["manifest_id"]}. '
                + "This is required so that the tests work against arbitrary tenants. "
                + "Please replace your tenant FQTN with <<TARGET_FQTN>>."
            )


@contextmanager
def manifest_with_real_fqtn(manifest_name):
    """
    Yields a file path to a manifest that has <<TARGET_FQTN>> replaced with the
    real target FQTN. Normally, we can apply ucconfig to multiple tenants no
    problem, but this end-to-end tests also runs gen-manifest and confirms that
    the generated manifest matches the original manifest. Since gen-manifest
    generates from scratch and only includes one tenant's FQTN, we just keep
    <<TARGET_FQTN>> in the e2e test manifests, and replace them with the real
    FQTN when we use them.
    """
    tenant_url = os.getenv("USERCLOUDS_TENANT_URL")
    if not tenant_url:
        raise Exception("USERCLOUDS_TENANT_URL is required")
    fqtn = urlparse(tenant_url).hostname.split(".")[0]

    # Note: we want to make a temporary file in the same directory as the
    # original manifest, so that relative references to e.g. transformer
    # definitions within that directory still work.
    target_path = os.path.join(
        TEST_DIR, manifest_name + ".substituted-tmp." + manifest_name.split(".")[-1]
    )
    with open(os.path.join(TEST_DIR, target_path), "w") as rewritten:
        with open(os.path.join(TEST_DIR, manifest_name)) as original:
            for line in original:
                rewritten.write(line.replace("<<TARGET_FQTN>>", fqtn))
    try:
        yield target_path
    finally:
        os.remove(target_path)


def apply_manifest(manifest_name, extra_args):
    with manifest_with_real_fqtn(manifest_name) as manifest_path:
        return run_capturing_output(
            ["ucconfig", "apply", manifest_path, "--auto-approve"] + extra_args
        )


def assert_no_changes(manifest_name, extra_args):
    with manifest_with_real_fqtn(manifest_name) as manifest_path:
        out = run_capturing_output(
            ["ucconfig", "apply", manifest_path, "--dry-run"] + extra_args
        )
        if "No changes. Your infrastructure matches the configuration." not in out:
            raise Exception("Got an unexpected diff!")
        return out


def assert_genmanifest_matches(manifest_name):
    with manifest_with_real_fqtn(manifest_name) as base_manifest_path:
        with tempfile.TemporaryDirectory() as tmpdirname:
            run_capturing_output(
                ["ucconfig", "gen-manifest", os.path.join(tmpdirname, manifest_name)]
            )
            diff_found = False

            # Compare manifest file
            if not filecmp.cmp(
                base_manifest_path, os.path.join(tmpdirname, manifest_name)
            ):
                print("Running gen-manifest modified the manifest:")
                subprocess.run(
                    [
                        "diff",
                        "-u",
                        base_manifest_path,
                        os.path.join(tmpdirname, manifest_name),
                    ]
                )
                diff_found = True

            # Compare values files
            values_dirname = Path(manifest_name).stem + "_values"
            existing_values_dir = os.path.join(TEST_DIR, values_dirname)
            generated_values_dir = os.path.join(tmpdirname, values_dirname)
            values_cmp = filecmp.dircmp(existing_values_dir, generated_values_dir)
            for f in values_cmp.left_only:
                print(f"Running genmanifest deleted {values_dirname}/{f}")
                diff_found = True
            for f in values_cmp.right_only:
                print(f"Running genmanifest created {values_dirname}/{f}")
                diff_found = True
            for f in values_cmp.diff_files:
                print(f"Running genmanifest modified {values_dirname}/{f}:")
                subprocess.run(
                    [
                        "diff",
                        "-u",
                        os.path.join(existing_values_dir, f),
                        os.path.join(generated_values_dir, f),
                    ]
                )
                print("")  # add blank line after diff output as separation
                diff_found = True

            if diff_found:
                raise Exception(
                    "genmanifest produced a differing manifest and/or external value files"
                )


def list_resource_types():
    with open(os.path.join(TEST_DIR, "../internal/resourcetypes/types.go")) as f:
        return RESOURCE_DEFINITION_REGEXP.findall(f.read())


def main():
    # Pass extra arguments onto ucconfig. (In the future maybe we'll want to
    # argparse here, but we have no other arguments at this time)
    ucconfig_apply_args = sys.argv[1:]

    # A previous failed run could have left unwanted resources. Start by
    # deleting all non-system resources.
    validate_manifest("empty.yaml")
    with log_step("Applying empty.yaml to get to baseline state..."):
        apply_manifest("empty.yaml", ucconfig_apply_args)
    with log_step("Applying empty.yaml again should not change anything..."):
        assert_no_changes("empty.yaml", ucconfig_apply_args)

    # Apply config with a bunch of resources
    with log_step("Applying lots-of-resources.yaml to test resource creation..."):
        validate_manifest("lots-of-resources.yaml")
        output = apply_manifest("lots-of-resources.yaml", ucconfig_apply_args)
        for tf_type_suffix in list_resource_types():
            if not re.search(
                f"userclouds_{tf_type_suffix}" + r"\.[a-zA-Z0-9_-]+ will be created",
                output,
            ):
                raise Exception(
                    f"Did not see resource type {tf_type_suffix} created in the output. Please add test coverage"
                )
    with log_step(
        "Applying lots-of-resources.yaml again should not change anything..."
    ):
        assert_no_changes("lots-of-resources.yaml", ucconfig_apply_args)

    with log_step(
        "Regenerating lots-of-resources.yaml should generate the same manifest..."
    ):
        assert_genmanifest_matches("lots-of-resources.yaml")

    with log_step(
        "Applying lots-of-resources-modified.yaml to test resource modification..."
    ):
        validate_manifest("lots-of-resources-modified.yaml")
        output = apply_manifest("lots-of-resources-modified.yaml", ucconfig_apply_args)
        for tf_type_suffix in list_resource_types():
            if not re.search(
                f"userclouds_{tf_type_suffix}"
                + r"\.[a-zA-Z0-9_-]+ (?:will be updated|must be replaced)",
                output,
            ):
                raise Exception(
                    f"Did not see resource type {tf_type_suffix} updated in the output. Please add test coverage"
                )
    with log_step(
        "Applying lots-of-resources-modified.yaml again should not change anything..."
    ):
        assert_no_changes("lots-of-resources-modified.yaml", ucconfig_apply_args)
    with log_step(
        "Regenerating lots-of-resources-modified.yaml should generate the same manifest..."
    ):
        assert_genmanifest_matches("lots-of-resources-modified.yaml")

    # Apply empty config
    with log_step("Applying empty.yaml to test resource deletion..."):
        apply_manifest("empty.yaml", ucconfig_apply_args)
    with log_step("Applying empty.yaml again should not change anything..."):
        assert_no_changes("empty.yaml", ucconfig_apply_args)


if __name__ == "__main__":
    main()
