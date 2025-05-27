from __future__ import annotations

from . import ucjson
from .constants import _JSON_CONTENT_TYPE
from .uchttpclient import UCHttpResponse


class UserCloudsSDKError(Exception):
    """Base class for all exceptions raised by the UserClouds SDK."""

    def __init__(
        self,
        error: str | dict = "unspecified error",
        http_status_code: int = -1,
        request_id: str | None = None,
        headers: dict[str, str] | None = None,
    ) -> None:
        super().__init__(error)
        self._err = error
        self.error_json = error if isinstance(error, dict) else None
        self.code = http_status_code
        self.request_id = request_id
        self._headers = headers

    def __repr__(self):
        return f"Error({self._err}, {self.code}, {self.request_id})"

    @property
    def headers(self) -> dict[str, str] | None:
        return self._headers

    @classmethod
    def from_response(cls, resp: UCHttpResponse) -> UserCloudsSDKError:
        request_id = resp.headers.get("X-Request-Id")
        if _is_json(resp):
            resp_json = ucjson.loads(resp.text)
            return cls(
                error=resp_json["error"],
                request_id=resp_json.get("request_id", request_id),
                http_status_code=resp.status_code,
                headers=dict(resp.headers),
            )
        else:
            return cls(
                error=f"HTTP {resp.status_code} - {resp.text}",
                request_id=request_id,
                http_status_code=resp.status_code,
                headers=dict(resp.headers),
            )


def _is_json(resp: UCHttpResponse) -> bool:
    return resp.headers.get("Content-Type") == _JSON_CONTENT_TYPE
