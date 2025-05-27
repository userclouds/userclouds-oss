# Add user data to tool

This tool reads the columns defined in a tenant creates fake data for users and adds it to the tenant.

## Prerequisites

- Python 3.11 or later
- direnv

## Setup

Run the following steps from the directory of this sample.

1. `make venv` - will create a python virtual environment and install the required packages.
2. Create a .envrc file (based on the template in envrc.template) and provide tenant URL, client ID & client secret.
3. `direnv allow` - to load the environment variables from the .envrc file.
4. `make check-env` - to check if the environment variables are set correctly.
5. `source .venv/bin/activate` - to activate the python virtual environment.

## Usage

After completing the setup, run the following command to add user data to the tenant.
Run the tool, it takes a few arguments, the first one is the number of users to add, and the second one is an optional list of purposes (names).
if a list of purposes is not provides, the tool will read the purposes from the UserClouds API and use them (all purposes).

example:

```shell
python3 uctool/add_data.py 3 operational marketing support
```
