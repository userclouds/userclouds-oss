import importlib.metadata
import os
import uuid
import warnings

from .errors import UserCloudsSDKError
from .models import APIErrorResponse


def _id_from_identical_conflict(err: UserCloudsSDKError) -> uuid.UUID:
    if err.code == 409:
        api_error = APIErrorResponse.from_json(err.error_json)
        if api_error.identical:
            return api_error.id
    raise err


def _read_env(name: str, desc: str) -> str:
    value = os.getenv(name)
    if not value:
        deprecated_name = name.removeprefix("USERCLOUDS_")
        value = os.getenv(deprecated_name)
        if value:
            warnings.warn(
                f"Warning: Environment variable '{deprecated_name}' is deprecated, please use '{name}' instead"
            )
    if not value:
        raise UserCloudsSDKError(
            f"Missing environment variable '{name}': UserClouds {desc}"
        )
    return value


_SDK_VERSION = importlib.metadata.version(__package__) or "unknown"
