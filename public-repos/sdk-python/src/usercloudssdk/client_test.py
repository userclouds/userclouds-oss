from __future__ import annotations

import pytest

from usercloudssdk.client import Client
from usercloudssdk.errors import UserCloudsSDKError


class TestClient:
    _FAKE_URL = "http://fake.seinfeld.org"

    @pytest.fixture
    def ucclient(self) -> Client:
        return Client(client_id="jerry", client_secret="cosmo", url=self._FAKE_URL)

    def test_get_token_success(self, ucclient: Client, httpx_mock) -> None:
        httpx_mock.add_response(
            method="POST",
            url=f"{self._FAKE_URL}/oidc/token",
            json={"access_token": "newman"},
        )
        token = ucclient._get_access_token()
        assert token == "newman"

    def test_get_token_success_failure_http_401(
        self, ucclient: Client, httpx_mock
    ) -> None:
        httpx_mock.add_response(
            method="POST",
            url=f"{self._FAKE_URL}/oidc/token",
            status_code=401,
            headers={"X-Request-ID": "kramer"},
            json={
                "error": "invalid_client",
                "error_description": "no plex app with Plex client ID ''",
            },
        )
        with pytest.raises(UserCloudsSDKError, match="invalid_client") as excinfo:
            ucclient._get_access_token()
        assert excinfo.value.code == 401
        assert excinfo.value.request_id == "kramer"
