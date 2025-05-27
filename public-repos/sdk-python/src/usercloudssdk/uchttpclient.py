from __future__ import annotations

import typing
from abc import ABC, abstractmethod
from dataclasses import dataclass

import httpx


@dataclass(frozen=True)
class UCHttpResponse:
    status_code: int
    headers: typing.MutableMapping[str, str]
    text: str


def UCHttpResponseFromHttpXResponse(resp: httpx.Response) -> UCHttpResponse:
    return UCHttpResponse(
        status_code=resp.status_code, headers=resp.headers, text=resp.text
    )


class UCHttpClient(ABC):
    @abstractmethod
    def get(
        self,
        url: str,
        *,
        params: dict[str, str] | None = None,
        headers: dict[str, str] | None = None,
    ) -> UCHttpResponse:
        pass

    @abstractmethod
    def post(
        self,
        url: str,
        *,
        params: dict[str, str] | None = None,
        content: str | bytes | None = None,
        headers: dict[str, str] | None = None,
    ) -> UCHttpResponse:
        pass

    @abstractmethod
    def put(
        self,
        url: str,
        *,
        content: str | bytes | None = None,
        headers: dict[str, str] | None = None,
    ) -> UCHttpResponse:
        pass

    @abstractmethod
    def delete(
        self,
        url: str,
        *,
        params: dict[str, str] | None = None,
        headers: dict[str, str] | None = None,
    ) -> UCHttpResponse:
        pass


class DefaultUCHttpClient(UCHttpClient):
    def __init__(self, *, base_url: str) -> None:
        self._client = httpx.Client(base_url=base_url)

    def get(self, url: str, *, params=None, headers=None) -> UCHttpResponse:
        return UCHttpResponseFromHttpXResponse(
            self._client.get(url, params=params, headers=headers)
        )

    def post(
        self, url: str, *, params=None, content=None, headers=None
    ) -> UCHttpResponse:
        return UCHttpResponseFromHttpXResponse(
            self._client.post(url, params=params, content=content, headers=headers)
        )

    def put(self, url: str, *, content=None, headers=None) -> UCHttpResponse:
        return UCHttpResponseFromHttpXResponse(
            self._client.put(url, content=content, headers=headers)
        )

    def delete(self, url: str, *, params=None, headers=None) -> UCHttpResponse:
        return UCHttpResponseFromHttpXResponse(
            self._client.delete(url, params=params, headers=headers)
        )


def create_default_uc_http_client(base_url: str) -> UCHttpClient:
    return DefaultUCHttpClient(base_url=base_url)


class UCHttpAsyncClient(ABC):
    @abstractmethod
    async def get_async(
        self,
        url: str,
        *,
        params: dict[str, str] | None = None,
        headers: dict[str, str] | None = None,
    ) -> UCHttpResponse:
        pass

    @abstractmethod
    async def post_async(
        self,
        url: str,
        *,
        params: dict[str, str] | None = None,
        content: str | bytes | None = None,
        headers: dict[str, str] | None = None,
    ) -> UCHttpResponse:
        pass

    @abstractmethod
    async def put_async(
        self,
        url: str,
        *,
        content: str | bytes | None = None,
        headers: dict[str, str] | None = None,
    ) -> UCHttpResponse:
        pass

    @abstractmethod
    async def delete_async(
        self,
        url: str,
        *,
        params: dict[str, str] | None = None,
        headers: dict[str, str] | None = None,
    ) -> UCHttpResponse:
        pass


class DefaultUCHttpAsyncClient(UCHttpAsyncClient):
    def __init__(self, *, base_url: str) -> None:
        self._client = httpx.AsyncClient(base_url=base_url)

    async def get_async(self, url: str, *, params=None, headers=None) -> UCHttpResponse:
        return UCHttpResponseFromHttpXResponse(
            await self._client.get(url, params=params, headers=headers)
        )

    async def post_async(
        self, url: str, *, params=None, content=None, headers=None
    ) -> UCHttpResponse:
        return UCHttpResponseFromHttpXResponse(
            await self._client.post(
                url, params=params, content=content, headers=headers
            )
        )

    async def put_async(
        self, url: str, *, content=None, headers=None
    ) -> UCHttpResponse:
        return UCHttpResponseFromHttpXResponse(
            await self._client.put(url, content=content, headers=headers)
        )

    async def delete_async(
        self, url: str, *, params=None, headers=None
    ) -> UCHttpResponse:
        return UCHttpResponseFromHttpXResponse(
            await self._client.delete(url, params=params, headers=headers)
        )


def create_default_uc_http_async_client(base_url: str) -> UCHttpClient:
    return DefaultUCHttpAsyncClient(base_url=base_url)


# Clients with timeout
class UCHttpClientWithTimeout(DefaultUCHttpClient):
    def __init__(self, *, base_url: str, timeout: int = 5) -> None:
        self._client = httpx.Client(base_url=base_url, timeout=timeout)


def create_uc_http_client_with_timeout(
    base_url: str, timeout=5
) -> UCHttpClientWithTimeout:
    return UCHttpClientWithTimeout(base_url=base_url, timeout=timeout)


class UCHttpAsyncClientWithTimeout(DefaultUCHttpClient):
    def __init__(self, *, base_url: str, timeout: int = 5) -> None:
        self._client = httpx.Client(base_url=base_url, timeout=timeout)


def create_uc_http_async_client_with_timeout(
    base_url: str, timeout=5
) -> UCHttpAsyncClientWithTimeout:
    return UCHttpAsyncClientWithTimeout(base_url=base_url, timeout=timeout)


# NoSSLVerification clients useful for debugging purposes, DO NOT USE for a production environment
class NoSSLVerificationHTTPClient(DefaultUCHttpClient):
    def __init__(self, *, base_url: str) -> None:
        self._client = httpx.Client(base_url=base_url, verify=False)


def create_no_ssl_http_client(base_url: str) -> NoSSLVerificationHTTPClient:
    return NoSSLVerificationHTTPClient(base_url=base_url)


class NoSSLVerificationHTTPAsyncClient(DefaultUCHttpAsyncClient):
    def __init__(self, *, base_url: str) -> None:
        self._client = httpx.AsyncClient(base_url=base_url, verify=False)


def create_no_ssl_http_async_client(base_url: str) -> NoSSLVerificationHTTPAsyncClient:
    return NoSSLVerificationHTTPAsyncClient(base_url=base_url)
