from __future__ import annotations

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
            client_secret=_read_env(
                "USERCLOUDS_CLIENT_SECRET", "UserClouds Client Secret"
            ),
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

    # User Operations

    # AuthN user methods (shouldn't be used by UserStore customers)

    async def CreateUserAsync(
        self,
        id: uuid.UUID | None = None,
        organization_id: uuid.UUID | None = None,
        region: str | Region | None = None,
    ) -> uuid.UUID:
        if isinstance(region, Region):
            region = region.value
        body = {
            "id": id,
            "organization_id": organization_id,
            "region": region,
        }
        resp_json = await self._post_async("/authn/users", json_data=body)
        return resp_json.get("id")

    async def CreateUserWithPasswordAsync(
        self,
        username: str,
        password: str,
        id: uuid.UUID | None = None,
        organization_id: uuid.UUID | None = None,
        region: str | Region | None = None,
    ) -> uuid.UUID:
        if isinstance(region, Region):
            region = region.value
        body = {
            "username": username,
            "password": password,
            "authn_type": AuthnType.PASSWORD.value,
            "id": id,
            "organization_id": organization_id,
            "region": region,
        }

        resp_json = await self._post_async("/authn/users", json_data=body)
        return resp_json.get("id")

    async def ListUsersAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
        organization_id: str | None = None,
    ) -> list[UserResponse]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        if organization_id is not None:
            params["organization_id"] = organization_id
        params["version"] = "3"
        resp_json = await self._get_async("/authn/users", params=params)

        users = [UserResponse.from_json(ur) for ur in resp_json["data"]]
        return users

    async def GetUserAsync(self, id: uuid.UUID) -> UserResponse:
        resp_json = await self._get_async(f"/authn/users/{id}")
        return UserResponse.from_json(resp_json)

    async def UpdateUserAsync(self, id: uuid.UUID, profile: dict) -> UserResponse:
        resp_json = await self._put_async(
            f"/authn/users/{id}", json_data={"profile": profile}
        )
        return UserResponse.from_json(resp_json)

    async def DeleteUserAsync(self, id: uuid.UUID) -> bool:
        return await self._delete_async(f"/authn/users/{id}")

    async def GetConsentedPurposesForUserAsync(
        self, id: uuid.UUID, columns: list[ResourceID] | None = None
    ) -> list[ColumnConsentedPurposes]:
        body = {"user_id": id}
        if columns:
            body["columns"] = [col.to_dict() for col in columns]
        resp_json = await self._post_async(
            "userstore/api/consentedpurposes", json_data=body
        )
        return [ColumnConsentedPurposes.from_json(p) for p in resp_json["data"]]

    # Userstore user methods (should be used along with Accessor and Mutator methods)

    async def CreateUserWithMutatorAsync(
        self,
        mutator_id: uuid.UUID,
        context: dict,
        row_data: dict,
        id: uuid.UUID | None = None,
        organization_id: uuid.UUID | None = None,
        region: str | Region | None = None,
    ) -> uuid.UUID:
        if isinstance(region, Region):
            region = region.value
        body = {
            "mutator_id": mutator_id,
            "context": context,
            "row_data": row_data,
            "id": id,
            "organization_id": organization_id,
            "region": region,
        }
        return await self._post_async("/userstore/api/users", json_data=body)

    # ColumnDataType Operations

    async def CreateColumnDataTypeAsync(
        self, dataType: ColumnDataType, if_not_exists: bool = False
    ) -> ColumnDataType:
        try:
            resp_json = await self._post_async(
                "/userstore/config/datatypes",
                json_data={"data_type": dataType.__dict__},
            )
            return ColumnDataType.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                dataType.id = _id_from_identical_conflict(err)
                return dataType
            raise err

    async def DeleteColumnDataTypeAsync(self, id: uuid.UUID) -> bool:
        return await self._delete_async(f"/userstore/config/datatypes/{id}")

    async def GetColumnDataTypeAsync(self, id: uuid.UUID) -> ColumnDataType:
        resp_json = await self._get_async(f"/userstore/config/datatypes/{id}")
        return ColumnDataType.from_json(resp_json)

    async def ListColumnDataTypesAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[ColumnDataType]:
        params: dict[str, int | str] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        resp_json = await self._get_async("/userstore/config/datatypes", params=params)
        dataTypes = [
            ColumnDataType.from_json(dataType) for dataType in resp_json["data"]
        ]
        return dataTypes

    async def UpdateColumnDataTypeAsync(
        self, dataType: ColumnDataType
    ) -> ColumnDataType:
        resp_json = await self._put_async(
            f"/userstore/config/datatypes/{dataType.id}",
            json_data={"data_type": dataType.__dict__},
        )
        return ColumnDataType.from_json(resp_json)

    # Column Operations

    async def CreateColumnAsync(
        self, column: Column, if_not_exists: bool = False
    ) -> Column:
        try:
            resp_json = await self._post_async(
                "/userstore/config/columns", json_data={"column": column.__dict__}
            )
            return Column.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                column.id = _id_from_identical_conflict(err)
                return column
            raise err

    async def DeleteColumnAsync(self, id: uuid.UUID) -> bool:
        return await self._delete_async(f"/userstore/config/columns/{id}")

    async def GetColumnAsync(self, id: uuid.UUID) -> Column:
        resp_json = await self._get_async(f"/userstore/config/columns/{id}")
        return Column.from_json(resp_json)

    async def ListColumnsAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[Column]:
        params: dict[str, int | str] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        resp_json = await self._get_async("/userstore/config/columns", params=params)
        columns = [Column.from_json(col) for col in resp_json["data"]]
        return columns

    async def UpdateColumnAsync(self, column: Column) -> Column:
        resp_json = await self._put_async(
            f"/userstore/config/columns/{column.id}",
            json_data={"column": column.__dict__},
        )
        return Column.from_json(resp_json)

    # Purpose Operations

    async def CreatePurposeAsync(
        self, purpose: Purpose, if_not_exists: bool = False
    ) -> Purpose:
        try:
            resp_json = await self._post_async(
                "/userstore/config/purposes", json_data={"purpose": purpose.__dict__}
            )
            return Purpose.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                purpose.id = _id_from_identical_conflict(err)
                return purpose
            raise err

    async def DeletePurposeAsync(self, id: uuid.UUID) -> bool:
        return await self._delete_async(f"/userstore/config/purposes/{id}")

    async def GetPurposeAsync(self, id: uuid.UUID) -> Purpose:
        json_resp = await self._get_async(f"/userstore/config/purposes/{id}")
        return Purpose.from_json(json_resp)

    async def ListPurposesAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[Purpose]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        resp_json = await self._get_async("/userstore/config/purposes", params=params)
        purposes = [Purpose.from_json(p) for p in resp_json["data"]]
        return purposes

    async def UpdatePurposeAsync(self, purpose: Purpose) -> Purpose:
        resp_json = await self._put_async(
            f"/userstore/config/purposes/{purpose.id}",
            json_data={"purpose": purpose.__dict__},
        )
        return Purpose.from_json(resp_json)

    # Retention Duration Operations

    # Tenant Retention Duration

    # A configured tenant retention duration will apply for
    # all column purposes that do not have a configured purpose
    # retention duration default or a configured column purpose
    # retention duration. If a tenant retention duration is
    # not configured, soft-deleted values will not be retained
    # by default.

    # create a tenant retention duration default
    async def CreateSoftDeletedRetentionDurationOnTenantAsync(
        self, req: UpdateColumnRetentionDurationRequest
    ) -> ColumnRetentionDurationResponse:
        resp = await self._post_async(
            "/userstore/config/softdeletedretentiondurations", json_data=req.to_json()
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # delete a tenant retention duration default
    async def DeleteSoftDeletedRetentionDurationOnTenantAsync(
        self, durationID: uuid.UUID
    ) -> bool:
        return await self._delete_async(
            f"/userstore/config/softdeletedretentiondurations/{durationID}"
        )

    # get a specific tenant retention duration default
    async def GetSoftDeletedRetentionDurationOnTenantAsync(
        self, durationID: uuid.UUID
    ) -> ColumnRetentionDurationResponse:
        resp = await self._get_async(
            f"/userstore/config/softdeletedretentiondurations/{durationID}"
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # get tenant retention duration, or default value if not specified
    async def GetDefaultSoftDeletedRetentionDurationOnTenantAsync(
        self,
    ) -> ColumnRetentionDurationResponse:
        resp = await self._get_async("/userstore/config/softdeletedretentiondurations")
        return ColumnRetentionDurationResponse.from_json(resp)

    # update a specific tenant retention duration default
    async def UpdateSoftDeletedRetentionDurationOnTenantAsync(
        self, durationID: uuid.UUID, req: UpdateColumnRetentionDurationRequest
    ) -> ColumnRetentionDurationResponse:
        resp = await self._put_async(
            f"/userstore/config/softdeletedretentiondurations/{durationID}",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # Purpose Retention Durations

    # A configured purpose retention duration will apply for all
    # column purposes that include the specified purpose, unless
    # a retention duration has been configured for a specific
    # column purpose.

    # create a purpose retention duration default
    async def CreateSoftDeletedRetentionDurationOnPurposeAsync(
        self, purposeID: uuid.UUID, req: UpdateColumnRetentionDurationRequest
    ) -> ColumnRetentionDurationResponse:
        resp = await self._post_async(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # delete a purpose retention duration default
    async def DeleteSoftDeletedRetentionDurationOnPurposeAsync(
        self, purposeID: uuid.UUID, durationID: uuid.UUID
    ) -> bool:
        return await self._delete_async(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations/{durationID}"
        )

    # get a specific purpose retention duration default
    async def GetSoftDeletedRetentionDurationOnPurposeAsync(
        self, purposeID: uuid.UUID, durationID: uuid.UUID
    ) -> ColumnRetentionDurationResponse:
        resp = await self._get_async(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations/{durationID}"
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # get purpose retention duration, or default value if not specified
    async def GetDefaultSoftDeletedRetentionDurationOnPurposeAsync(
        self, purposeID: uuid.UUID
    ) -> ColumnRetentionDurationResponse:
        resp = await self._get_async(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations"
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # update a specific purpose retention duration default
    async def UpdateSoftDeletedRetentionDurationOnPurposeAsync(
        self,
        purposeID: uuid.UUID,
        durationID: uuid.UUID,
        req: UpdateColumnRetentionDurationRequest,
    ) -> ColumnRetentionDurationResponse:
        resp = await self._put_async(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations/{durationID}",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # Column Retention Durations

    # A configured column purpose retention duration will override
    # any configured purpose level, tenant level, or system-level
    # default retention durations.

    # get a specific column purpose retention duration
    async def GetSoftDeletedRetentionDurationOnColumnAsync(
        self, columnID: uuid.UUID, durationID: uuid.UUID
    ) -> ColumnRetentionDurationResponse:
        resp = await self._get_async(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations/{durationID}"
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # get the retention duration for each purpose for a given column
    async def GetSoftDeletedRetentionDurationsOnColumnAsync(
        self, columnID: uuid.UUID
    ) -> ColumnRetentionDurationsResponse:
        resp = await self._get_async(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations"
        )
        return ColumnRetentionDurationsResponse.from_json(resp)

    # delete a specific column purpose retention duration
    async def DeleteSoftDeletedRetentionDurationOnColumnAsync(
        self, columnID: uuid.UUID, durationID: uuid.UUID
    ) -> bool:
        return await self._delete_async(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations/{durationID}"
        )

    # update a specific column purpose retention duration
    async def UpdateSoftDeletedRetentionDurationOnColumnAsync(
        self,
        columnID: uuid.UUID,
        durationID: uuid.UUID,
        req: UpdateColumnRetentionDurationRequest,
    ) -> ColumnRetentionDurationResponse:
        resp = await self._put_async(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations/{durationID}",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # update the specified purpose retention durations for the column
    # - durations can be added, deleted, or updated for each purpose
    async def UpdateSoftDeletedRetentionDurationsOnColumnAsync(
        self, columnID: uuid.UUID, req: UpdateColumnRetentionDurationsRequest
    ) -> ColumnRetentionDurationsResponse:
        resp = await self._post_async(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationsResponse.from_json(resp)

    # Access Policy Templates

    async def CreateAccessPolicyTemplateAsync(
        self, access_policy_template: AccessPolicyTemplate, if_not_exists: bool = False
    ) -> AccessPolicyTemplate | UserCloudsSDKError:
        try:
            resp_json = await self._post_async(
                "/tokenizer/policies/accesstemplate",
                json_data={"access_policy_template": access_policy_template.__dict__},
            )
            return AccessPolicyTemplate.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                access_policy_template.id = _id_from_identical_conflict(err)
                return access_policy_template
            raise err

    async def ListAccessPolicyTemplatesAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ):
        params: dict[str, int | str] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        resp_json = await self._get_async(
            "/tokenizer/policies/accesstemplate", params=params
        )

        templates = [AccessPolicyTemplate.from_json(apt) for apt in resp_json["data"]]
        return templates

    async def GetAccessPolicyTemplateAsync(self, rid: ResourceID):
        if hasattr(rid, "id"):
            resp_json = await self._get_async(
                f"/tokenizer/policies/accesstemplate/{rid.id}"
            )
            return AccessPolicyTemplate.from_json(resp_json)
        elif hasattr(rid, "name"):
            resp_json = await self._get_async(
                f"/tokenizer/policies/accesstemplate?template_name={rid.name}"
            )
            if len(resp_json["data"]) == 1:
                return AccessPolicyTemplate.from_json(resp_json["data"][0])
            raise UserCloudsSDKError(
                f"Access Policy Template with name {rid.name} not found", 404
            )
        raise UserCloudsSDKError("Invalid ResourceID", 400)

    async def UpdateAccessPolicyTemplateAsync(
        self, access_policy_template: AccessPolicyTemplate
    ):
        resp_json = await self._put_async(
            f"/tokenizer/policies/accesstemplate/{access_policy_template.id}",
            json_data={"access_policy_template": access_policy_template.__dict__},
        )
        return AccessPolicyTemplate.from_json(resp_json)

    async def DeleteAccessPolicyTemplateAsync(self, id: uuid.UUID, version: int):
        return await self._delete_async(
            f"/tokenizer/policies/accesstemplate/{id}",
            params={"template_version": str(version)},
        )

    # Access Policies

    async def CreateAccessPolicyAsync(
        self, access_policy: AccessPolicy, if_not_exists: bool = False
    ) -> AccessPolicy | UserCloudsSDKError:
        try:
            resp_json = await self._post_async(
                "/tokenizer/policies/access",
                json_data={"access_policy": access_policy.__dict__},
            )
            return AccessPolicy.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                access_policy.id = _id_from_identical_conflict(err)
                return access_policy
            raise err

    async def ListAccessPoliciesAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ):
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        resp_json = await self._get_async("/tokenizer/policies/access", params=params)

        policies = [AccessPolicy.from_json(ap) for ap in resp_json["data"]]
        return policies

    async def GetAccessPolicyAsync(self, rid: ResourceID):
        if hasattr(rid, "id"):
            resp_json = await self._get_async(f"/tokenizer/policies/access/{rid.id}")
            return AccessPolicy.from_json(resp_json)
        elif hasattr(rid, "name"):
            resp_json = await self._get_async(
                f"/tokenizer/policies/access?policy_name={rid.name}"
            )
            if len(resp_json["data"]) == 1:
                return AccessPolicy.from_json(resp_json["data"][0])
            raise UserCloudsSDKError(
                f"Access Policy with name {rid.name} not found", 404
            )
        raise UserCloudsSDKError("Invalid ResourceID", 400)

    async def UpdateAccessPolicyAsync(self, access_policy: AccessPolicy):
        resp_json = await self._put_async(
            f"/tokenizer/policies/access/{access_policy.id}",
            json_data={"access_policy": access_policy.__dict__},
        )
        return AccessPolicy.from_json(resp_json)

    async def DeleteAccessPolicyAsync(self, id: uuid.UUID, version: int):
        return await self._delete_async(
            f"/tokenizer/policies/access/{id}",
            params={"policy_version": str(version)},
        )

    # Transformers

    async def CreateTransformerAsync(
        self, transformer: Transformer, if_not_exists: bool = False
    ):
        try:
            resp_json = await self._post_async(
                "/tokenizer/policies/transformation",
                json_data={"transformer": transformer.__dict__},
            )
            return Transformer.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                transformer.id = _id_from_identical_conflict(err)
                return transformer
            raise err

    async def GetTransformerAsync(self, rid: ResourceID):
        if hasattr(rid, "id"):
            resp_json = await self._get_async(
                f"/tokenizer/policies/transformation/{rid.id}"
            )
            return Transformer.from_json(resp_json)
        elif hasattr(rid, "name"):
            resp_json = await self._get_async(
                f"/tokenizer/policies/transformation?transformer_name={rid.name}"
            )
            if len(resp_json["data"]) == 1:
                return Transformer.from_json(resp_json["data"][0])
            raise UserCloudsSDKError(f"Transformer with name {rid.name} not found", 404)
        raise UserCloudsSDKError("Invalid ResourceID", 400)

    async def ListTransformersAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ):
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        resp_json = await self._get_async(
            "/tokenizer/policies/transformation", params=params
        )
        transformers = [Transformer.from_json(tf) for tf in resp_json["data"]]
        return transformers

    async def UpdateTransformerAsync(self, transformer: Transformer):
        resp_json = await self._put_async(
            f"/tokenizer/policies/transformation/{transformer.id}",
            json_data={"transformer": transformer.__dict__},
        )
        return Transformer.from_json(resp_json)

    async def DeleteTransformerAsync(self, id: uuid.UUID):
        return await self._delete_async(f"/tokenizer/policies/transformation/{id}")

    # Accessor Operations

    async def CreateAccessorAsync(
        self, accessor: Accessor, if_not_exists: bool = False
    ) -> Accessor:
        try:
            resp_json = await self._post_async(
                "/userstore/config/accessors", json_data={"accessor": accessor.__dict__}
            )
            return Accessor.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                accessor.id = _id_from_identical_conflict(err)
                return accessor
            raise err

    async def DeleteAccessorAsync(self, id: uuid.UUID) -> bool:
        return await self._delete_async(f"/userstore/config/accessors/{id}")

    async def GetAccessorAsync(self, id: uuid.UUID) -> Accessor:
        j = await self._get_async(f"/userstore/config/accessors/{id}")
        return Accessor.from_json(j)

    async def ListAccessorsAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[Accessor]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        resp_json = await self._get_async("/userstore/config/accessors", params=params)

        accessors = [Accessor.from_json(acs) for acs in resp_json["data"]]
        return accessors

    async def UpdateAccessorAsync(self, accessor: Accessor) -> Accessor:
        resp_json = await self._put_async(
            f"/userstore/config/accessors/{accessor.id}",
            json_data={"accessor": accessor.__dict__},
        )
        return Accessor.from_json(resp_json)

    async def ExecuteAccessorAsync(
        self,
        accessor_id: uuid.UUID,
        context: dict,
        selector_values: list,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
        region: str | None = None,
        access_primary_db_only: bool | None = None,
    ) -> list:
        body = {
            "accessor_id": accessor_id,
            "context": context,
            "selector_values": selector_values,
        }
        if region is not None:
            body["region"] = region
        if access_primary_db_only is not None:
            body["access_primary_db_only"] = access_primary_db_only
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after

        return await self._post_async(
            "/userstore/api/accessors", json_data=body, params=params
        )

    # Mutator Operations

    async def CreateMutatorAsync(
        self, mutator: Mutator, if_not_exists: bool = False
    ) -> Mutator:
        try:
            resp_json = await self._post_async(
                "/userstore/config/mutators", json_data={"mutator": mutator.__dict__}
            )
            return Mutator.from_json(resp_json)
        except UserCloudsSDKError as e:
            if if_not_exists:
                mutator.id = _id_from_identical_conflict(e)
                return mutator
            raise e

    async def DeleteMutatorAsync(self, id: uuid.UUID) -> bool:
        return await self._delete_async(f"/userstore/config/mutators/{id}")

    async def GetMutatorAsync(self, id: uuid.UUID) -> Mutator:
        resp_json = await self._get_async(f"/userstore/config/mutators/{id}")
        return Mutator.from_json(resp_json)

    async def ListMutatorsAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[Mutator]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        j = await self._get_async("/userstore/config/mutators", params=params)

        mutators = [Mutator.from_json(m) for m in j["data"]]
        return mutators

    async def UpdateMutatorAsync(self, mutator: Mutator) -> Mutator:
        json_resp = await self._put_async(
            f"/userstore/config/mutators/{mutator.id}",
            json_data={"mutator": mutator.__dict__},
        )
        return Mutator.from_json(json_resp)

    async def ExecuteMutatorAsync(
        self,
        mutator_id: uuid.UUID,
        context: dict,
        selector_values: list,
        row_data: dict,
        region: str | None = None,
    ) -> str:
        body = {
            "mutator_id": mutator_id,
            "context": context,
            "selector_values": selector_values,
            "row_data": row_data,
        }
        if region is not None:
            body["region"] = region

        j = await self._post_async("/userstore/api/mutators", json_data=body)
        return j

    # Token Operations

    async def CreateTokenAsync(
        self,
        data: str,
        transformer_rid: ResourceID,
        access_policy_rid: ResourceID,
    ) -> str:
        body = {
            "data": data,
            "transformer_rid": transformer_rid.__dict__,
            "access_policy_rid": access_policy_rid.__dict__,
        }

        json_resp = await self._post_async("/tokenizer/tokens", json_data=body)
        return json_resp["data"]

    async def LookupOrCreateTokensAsync(
        self,
        data: list[str],
        transformers: list[ResourceID],
        access_policies: list[ResourceID],
    ) -> list[str]:
        body = {
            "data": data,
            "transformer_rids": [asdict(t) for t in transformers],
            "access_policy_rids": [asdict(a) for a in access_policies],
        }

        j = await self._post_async(
            "/tokenizer/tokens/actions/lookuporcreate", json_data=body
        )
        return j["tokens"]

    async def ResolveTokensAsync(
        self, tokens: list[str], context: dict, purposes: list[ResourceID]
    ) -> list[str]:
        body = {"tokens": tokens, "context": context, "purposes": purposes}

        j = await self._post_async("/tokenizer/tokens/actions/resolve", json_data=body)
        return j

    async def DeleteTokenAsync(self, token: str) -> bool:
        return await self._delete_async("/tokenizer/tokens", params={"token": token})

    async def InspectTokenAsync(self, token: str) -> InspectTokenResponse:
        body = {"token": token}

        j = await self._post_async("/tokenizer/tokens/actions/inspect", json_data=body)
        return InspectTokenResponse.from_json(j)

    async def LookupTokenAsync(
        self,
        data: str,
        transformer_rid: Transformer,
        access_policy_rid: AccessPolicy,
    ) -> str:
        body = {
            "data": data,
            "transformer_rid": transformer_rid.__dict__,
            "access_policy_rid": access_policy_rid.__dict__,
        }

        j = await self._post_async("/tokenizer/tokens/actions/lookup", json_data=body)
        return j["tokens"]

    # AuthZ Operations

    async def ListObjectsAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[Object]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        j = await self._get_async("/authz/objects", params=params)

        objects = [Object.from_json(o) for o in j["data"]]
        return objects

    async def CreateObjectAsync(
        self, object: Object, if_not_exists: bool = False
    ) -> Object:
        try:
            j = await self._post_async(
                "/authz/objects", json_data={"object": object.__dict__}
            )
            return Object.from_json(j)
        except UserCloudsSDKError as e:
            if if_not_exists:
                object.id = _id_from_identical_conflict(e)
                return object
            raise e

    async def GetObjectAsync(self, id: uuid.UUID) -> Object:
        j = await self._get_async(f"/authz/objects/{id}")
        return Object.from_json(j)

    async def DeleteObjectAsync(self, id: uuid.UUID):
        return await self._delete_async(f"/authz/objects/{id}")

    async def ListEdgesAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[Edge]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        j = await self._get_async("/authz/edges", params=params)

        edges = [Edge.from_json(e) for e in j["data"]]
        return edges

    async def CreateEdgeAsync(self, edge: Edge, if_not_exists: bool = False) -> Edge:
        try:
            j = await self._post_async(
                "/authz/edges", json_data={"edge": edge.__dict__}
            )
            return Edge.from_json(j)
        except UserCloudsSDKError as e:
            if if_not_exists:
                edge.id = _id_from_identical_conflict(e)
                return edge
            raise e

    async def GetEdgeAsync(self, id: uuid.UUID) -> Edge:
        j = await self._get_async(f"/authz/edges/{id}")
        return Edge.from_json(j)

    async def DeleteEdgeAsync(self, id: uuid.UUID):
        return await self._delete_async(f"/authz/edges/{id}")

    async def ListObjectTypesAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[ObjectType]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        j = await self._get_async("/authz/objecttypes", params=params)

        object_types = [ObjectType.from_json(ot) for ot in j["data"]]
        return object_types

    async def CreateObjectTypeAsync(
        self, object_type: ObjectType, if_not_exists: bool = False
    ) -> ObjectType:
        try:
            j = await self._post_async(
                "/authz/objecttypes", json_data={"object_type": object_type.__dict__}
            )
            return ObjectType.from_json(j)
        except UserCloudsSDKError as e:
            if if_not_exists:
                object_type.id = _id_from_identical_conflict(e)
                return object_type
            raise e

    async def GetObjectTypeAsync(self, id: uuid.UUID) -> ObjectType:
        j = await self._get_async(f"/authz/objecttypes/{id}")
        return ObjectType.from_json(j)

    async def DeleteObjectTypeAsync(self, id: uuid.UUID):
        return await self._delete_async(f"/authz/objecttypes/{id}")

    async def ListEdgeTypesAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[EdgeType]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        j = await self._get_async("/authz/edgetypes", params=params)

        edge_types = [EdgeType.from_json(et) for et in j["data"]]
        return edge_types

    async def CreateEdgeTypeAsync(
        self, edge_type: EdgeType, if_not_exists: bool = False
    ) -> EdgeType:
        try:
            j = await self._post_async(
                "/authz/edgetypes", json_data={"edge_type": edge_type.__dict__}
            )
            return EdgeType.from_json(j)
        except UserCloudsSDKError as e:
            if if_not_exists:
                edge_type.id = _id_from_identical_conflict(e)
                return edge_type
            raise e

    async def GetEdgeTypeAsync(self, id: uuid.UUID) -> EdgeType:
        j = await self._get_async(f"/authz/edgetypes/{id}")
        return EdgeType.from_json(j)

    async def DeleteEdgeTypeAsync(self, id: uuid.UUID) -> bool:
        return await self._delete_async(f"/authz/edgetypes/{id}")

    async def ListOrganizationsAsync(
        self,
        limit: int = 0,
        starting_after: str | None = None,
        ending_before: str | None = None,
        filter_clause: str | None = None,
        sort_key: str | None = None,
        sort_order: str | None = None,
    ) -> list[Organization]:
        params: dict[str, str | int] = {}
        if ending_before is not None:
            params["ending_before"] = ending_before
        if filter_clause is not None:
            params["filter"] = filter_clause
        if limit > 0:
            params["limit"] = limit
        if sort_key is not None:
            params["sort_key"] = sort_key
        if sort_order is not None:
            params["sort_order"] = sort_order
        if starting_after is not None:
            params["starting_after"] = starting_after
        params["version"] = "3"
        j = await self._get_async("/authz/organizations", params=params)

        organizations = [Organization.from_json(o) for o in j["data"]]
        return organizations

    async def CreateOrganizationAsync(
        self, organization: Organization, if_not_exists: bool = False
    ) -> Organization:
        try:
            json_data = await self._post_async(
                "/authz/organizations",
                json_data={"organization": organization.__dict__},
            )
            return Organization.from_json(json_data)
        except UserCloudsSDKError as e:
            if if_not_exists:
                organization.id = _id_from_identical_conflict(e)
                return organization
            raise e

    async def GetOrganizationAsync(self, id: uuid.UUID) -> Organization:
        json_data = await self._get_async(f"/authz/organizations/{id}")
        return Organization.from_json(json_data)

    async def DeleteOrganizationAsync(self, id: uuid.UUID) -> bool:
        return await self._delete_async(f"/authz/organizations/{id}")

    async def CheckAttributeAsync(
        self,
        source_object_id: uuid.UUID,
        target_object_id: uuid.UUID,
        attribute_name: str,
    ) -> bool:
        j = await self._get_async(
            f"/authz/checkattribute?source_object_id={source_object_id}&target_object_id={target_object_id}&attribute={attribute_name}"
        )
        return j.get("has_attribute")

    async def DownloadUserstoreSDKAsync(self, include_example=True) -> str:
        return await self._download_async(
            f"/userstore/download/codegensdk.py?include_example={include_example and 'true' or 'false'}"
        )

    async def SaveUserstoreSDKAsync(
        self, path: Path, include_example: bool = False
    ) -> None:
        sdk = await self.DownloadUserstoreSDKAsync(include_example=include_example)
        path.write_text(sdk)

    async def GetExternalOIDCIssuersAsync(self) -> list[str]:
        return await self._get_async("/userstore/oidcissuers")

    async def UpdateExternalOIDCIssuersAsync(self, issuers: list[str]) -> list[str]:
        return await self._put_async("/userstore/oidcissuers", json_data=issuers)

    async def UploadDataImportFileAsync(
        self, file_path: Path, import_type: str = "executemutator"
    ) -> uuid.UUID:
        json_data = await self._get_async(
            f"/userstore/upload/dataimport?import_type={import_type}"
        )
        import_id = uuid.UUID(json_data["import_id"])
        with open(file_path, "rb") as f:
            resp = await self._client.put_async(json_data["presigned_url"], content=f)
            if resp.status_code >= 400:
                raise UserCloudsSDKError.from_response(resp)
        return import_id

    async def CheckDataImportStatusAsync(self, import_id: uuid.UUID) -> dict:
        json_data = await self._get_async(f"/userstore/upload/dataimport/{import_id}")
        return json_data

    # Access Token Helpers

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
        resp = await self._client.post_async(
            url, params=params, headers=headers, content=content
        )
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
