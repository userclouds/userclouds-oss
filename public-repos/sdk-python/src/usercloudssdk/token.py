from __future__ import annotations

import time

import jwt

_token_cache: dict[str, str] = {}


def get_cached_token(authorization: str) -> str | None:
    global _token_cache
    return _token_cache.get(authorization, None)


def cache_token(authorization: str, token: str) -> None:
    if not token:
        return
    global _token_cache
    _token_cache[authorization] = token


def is_token_expiring(token: str | None) -> bool:
    if not token:
        return True
    # TODO: this takes advantage of an implementation detail that we use JWTs for
    # access tokens, but we should probably either expose an endpoint to verify
    # expiration time, or expect to retry requests with a well-formed error, or
    # change our bearer token format in time.
    decoded_token = jwt.decode(token, options={"verify_signature": False})
    expiration_time = decoded_token.get("exp")
    if not expiration_time:
        return True
    return expiration_time < time.time()
