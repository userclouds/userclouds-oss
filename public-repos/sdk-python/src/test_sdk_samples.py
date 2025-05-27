from __future__ import annotations

import os

import pytest

from authz_sample import run_authz_sample
from tokenizer_sample import run_tokenizer_sample
from usercloudssdk.client import Client
from usercloudssdk.constants import Region
from usercloudssdk.uchttpclient import create_uc_http_client_with_timeout
from userstore_sample import run_userstore_sample


class TestSDKSamples:
    @pytest.fixture(
        params=[
            {},
            {"client_factory": create_uc_http_client_with_timeout, "timeout": 5},
        ]
    )
    def ucclient(self, request) -> Client:
        kwargs = {"session_name": os.environ.get("UC_SESSION_NAME")}
        kwargs.update(request.param)
        return Client.from_env(**kwargs)

    def test_authz(self, ucclient: Client) -> None:
        run_authz_sample(ucclient)

    def test_tokenizer(self, ucclient: Client) -> None:
        run_tokenizer_sample(ucclient)

    def test_userstore(self, ucclient: Client) -> None:
        user_region = os.environ.get("UC_REGION", Region.AWS_US_EAST_1)
        run_userstore_sample(client=ucclient, user_region=user_region)
