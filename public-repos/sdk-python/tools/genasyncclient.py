import re
import subprocess

infile = "src/usercloudssdk/client.py"
outfile = "src/usercloudssdk/asyncclient.py"

non_auto_generated_header = """from __future__ import annotations

import asyncio
import base64
import urllib.parse
import uuid
from dataclasses import asdict
from pathlib import Path

from . import ucjson
from .client_helpers import _SDK_VERSION, _id_from_identical_conflict, _read_env
from .constants import _JSON_CONTENT_TYPE, AuthnType, Region
from .errors import UserCloudsSDKError
from .models import (
    Accessor,
    AccessPolicy,
    AccessPolicyTemplate,
    Column,
    ColumnConsentedPurposes,
    ColumnDataType,
    ColumnRetentionDurationResponse,
    ColumnRetentionDurationsResponse,
    Edge,
    EdgeType,
    InspectTokenResponse,
    Mutator,
    Object,
    ObjectType,
    Organization,
    Purpose,
    ResourceID,
    Transformer,
    UpdateColumnRetentionDurationRequest,
    UpdateColumnRetentionDurationsRequest,
    UserResponse,
)
from .token import cache_token, get_cached_token, is_token_expiring
from .uchttpclient import create_default_uc_http_async_client


class AsyncClient:
    @classmethod
    def from_env(cls, client_factory=create_default_uc_http_async_client, **kwargs):
        return cls(
            url=_read_env("USERCLOUDS_TENANT_URL", "UserClouds Tenant URL"),
            client_id=_read_env("USERCLOUDS_CLIENT_ID", "UserClouds Client ID"),
            client_secret=_read_env("USERCLOUDS_CLIENT_SECRET", "UserClouds Client Secret"),
            client_factory=client_factory,
            **kwargs,
        )

    def __init__(
        self,
        url: str,
        client_id: str,
        client_secret: str,
        client_factory=create_default_uc_http_async_client,
        session_name: str | None = None,
        use_global_cache_for_token: bool = False,
        **kwargs,
    ):
        self._authorization = base64.b64encode(
            bytes(
                f"{ urllib.parse.quote(client_id)}:{ urllib.parse.quote(client_secret)}",
                "ISO-8859-1",
            )
        ).decode("ascii")
        self._client = client_factory(base_url=url, **kwargs)
        self._access_token: str | None = None  # lazy loaded
        self._use_global_cache_for_token = use_global_cache_for_token
        self._access_token_lock = asyncio.Lock()
        base_ua = f"UserClouds Python SDK v{_SDK_VERSION}"
        self._common_headers = {
            "User-Agent": f"{base_ua} [{session_name}]" if session_name else base_ua,
            "X-Usercloudssdk-Version": _SDK_VERSION,
        }
"""

non_auto_generated_footer = """
    async def _get_access_token_async(self) -> str:
        if self._use_global_cache_for_token:
            token = get_cached_token(self._authorization)
            if token:
                return token
        # Encode the client ID and client secret
        headers = {
            "Authorization": f"Basic {self._authorization}",
            "Content-Type": "application/x-www-form-urlencoded",
        }
        headers.update(self._common_headers)
        body = "grant_type=client_credentials"

        # Note that we use requests directly here (instead of _post) because we don't
        # want to refresh the access token as we are trying to get it. :)
        resp = await self._client.post_async(
            "/oidc/token", headers=headers, content=body
        )
        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        json_data = ucjson.loads(resp.text)
        token = json_data["access_token"]
        if self._use_global_cache_for_token:
            cache_token(self._authorization, token)
        return token

    async def _refresh_access_token_if_needed_async(self) -> None:
        if self._access_token is None:
            async with self._access_token_lock:
                if self._access_token is None:
                    self._access_token = await self._get_access_token_async()
                    return

        if is_token_expiring(self._access_token):
            async with self._access_token_lock:
                if is_token_expiring(self._access_token):
                    self._access_token = await self._get_access_token_async()

    # Request Helpers

    def _get_headers(self) -> dict:
        headers = {"Authorization": f"Bearer {self._access_token}"}
        headers.update(self._common_headers)
        return headers

    async def _prep_json_data_async(
        self, json_data: dict | str | None
    ) -> tuple[dict, str | None]:
        await self._refresh_access_token_if_needed_async()
        headers = self._get_headers()
        content = None
        if json_data is not None:
            headers["Content-Type"] = _JSON_CONTENT_TYPE
            content = (
                json_data if isinstance(json_data, str) else ucjson.dumps(json_data)
            )
        return headers, content

    async def _get_async(self, url, params: dict[str, str | int] | None = None) -> dict:
        await self._refresh_access_token_if_needed_async()
        resp = await self._client.get_async(
            url, params=params, headers=self._get_headers()
        )
        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        return ucjson.loads(resp.text)

    async def _post_async(
        self,
        url,
        json_data: dict | str | None = None,
        params: dict[str, str | int] | None = None,
    ) -> dict | list:
        headers, content = await self._prep_json_data_async(json_data)
        resp = await self._client.post_async(url, params=params, headers=headers, content=content)
        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        return ucjson.loads(resp.text)

    async def _put_async(self, url, json_data: dict | str | None = None) -> dict | list:
        headers, content = await self._prep_json_data_async(json_data)
        resp = await self._client.put_async(url, headers=headers, content=content)
        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        return ucjson.loads(resp.text)

    async def _delete_async(
        self, url, params: dict[str, str | int] | None = None
    ) -> bool:
        await self._refresh_access_token_if_needed_async()
        resp = await self._client.delete_async(
            url, params=params, headers=self._get_headers()
        )

        if resp.status_code == 404:
            return False

        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        return resp.status_code == 204

    async def _download_async(self, url) -> str:
        await self._refresh_access_token_if_needed_async()
        resp = await self._client.get_async(url, headers=self._get_headers())
        return resp.text
"""

with open(outfile, "w") as out, open(infile) as f:
    out.write(non_auto_generated_header)

    doing_substitution = False
    for index, line in enumerate(f):
        if "# User Operations" in line:
            doing_substitution = True

        if not doing_substitution:
            continue

        # replace "def" with "async def" and end function name with "Async"
        line = re.sub(
            r"^(\s*)def\s+([a-zA-Z_][a-zA-Z_0-9]*)\s*\(",
            r"\1async def \2Async(",
            line,
        )
        # replace self._get with await self._get_async
        line = re.sub(r"\sself\._get\(", r" await self._get_async(", line)
        # replace self._client.get with await self._client.get_async
        line = re.sub(r"\sself\._client.get\(", r" await self._client.get_async(", line)
        # replace self._post with await self._post_async
        line = re.sub(r"\sself\._post\(", r" await self._post_async(", line)
        # replace self._client.post with await self._client.post_async
        line = re.sub(
            r"\sself\._client.post\(", r" await self._client.post_async(", line
        )
        # replace self._put with await self._put_async
        line = re.sub(r"\sself\._put\(", r" await self._put_async(", line)
        # replace self._client.put with await self._client.put_async
        line = re.sub(r"\sself\._client.put\(", r" await self._client.put_async(", line)
        # replace self._delete with await self._delete_async
        line = re.sub(r"\sself\._delete\(", r" await self._delete_async(", line)
        # replace self._client.delete with await self._client.delete_async
        line = re.sub(
            r"\sself\._client.delete\(", r" await self._client.delete_async(", line
        )
        # replace self._download with await self._download_async
        line = re.sub(r"\sself\._download\(", r" await self._download_async(", line)
        # replace self._download with await self._download_async
        line = re.sub(
            r"\sself\.DownloadUserstoreSDK\(",
            r" await self.DownloadUserstoreSDKAsync(",
            line,
        )

        out.write(line)

        if "# Access Token Helpers" in line:
            break

    out.write(non_auto_generated_footer)

subprocess.run(["black", outfile])
