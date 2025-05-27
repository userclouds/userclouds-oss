# UserClouds Python SDK

[![PyPI version](https://badge.fury.io/py/usercloudssdk.svg)](http://badge.fury.io/py/usercloudssdk)
[![PyPi page link -- Python versions](https://img.shields.io/pypi/pyversions/usercloudssdk.svg)](https://pypi.python.org/pypi/usercloudssdk)
[![Code style: black](https://img.shields.io/badge/code%20style-black-000000.svg)](https://github.com/psf/black)

Prerequisites:

- Python 3.9+

Install the SDK by running

```shell
pip3 install usercloudssdk
```

You need to have a tenant on UserClouds to run the sample code against. Once you have the tenant information, you can try running the various samples in this repo. Clone this repo locally, and then update the following lines in `userstore_sample.py`:

```shell
client_id = "<REPLACE ME>"
client_secret = "<REPLACE ME>"
url = "<REPLACE ME>"
```

with the details from your tenant.

then run:

```shell
python3 userstore_sample.py
```

You can do the same with `authz_sample.py` and `tokenizer_sample.py`
