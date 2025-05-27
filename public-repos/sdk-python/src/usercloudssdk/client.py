from __future__ import annotations

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
from .uchttpclient import create_default_uc_http_client


class Client:
    @classmethod
    def from_env(cls, client_factory=create_default_uc_http_client, **kwargs):
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
        client_factory=create_default_uc_http_client,
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
        base_ua = f"UserClouds Python SDK v{_SDK_VERSION}"
        self._common_headers = {
            "User-Agent": f"{base_ua} [{session_name}]" if session_name else base_ua,
            "X-Usercloudssdk-Version": _SDK_VERSION,
        }

    # User Operations

    # AuthN user methods (shouldn't be used by UserStore customers)

    def CreateUser(
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
        resp_json = self._post("/authn/users", json_data=body)
        return resp_json.get("id")

    def CreateUserWithPassword(
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

        resp_json = self._post("/authn/users", json_data=body)
        return resp_json.get("id")

    def ListUsers(
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
        resp_json = self._get("/authn/users", params=params)

        users = [UserResponse.from_json(ur) for ur in resp_json["data"]]
        return users

    def GetUser(self, id: uuid.UUID) -> UserResponse:
        resp_json = self._get(f"/authn/users/{id}")
        return UserResponse.from_json(resp_json)

    def UpdateUser(self, id: uuid.UUID, profile: dict) -> UserResponse:
        resp_json = self._put(f"/authn/users/{id}", json_data={"profile": profile})
        return UserResponse.from_json(resp_json)

    def DeleteUser(self, id: uuid.UUID) -> bool:
        return self._delete(f"/authn/users/{id}")

    def GetConsentedPurposesForUser(
        self, id: uuid.UUID, columns: list[ResourceID] | None = None
    ) -> list[ColumnConsentedPurposes]:
        body = {"user_id": id}
        if columns:
            body["columns"] = [col.to_dict() for col in columns]
        resp_json = self._post("userstore/api/consentedpurposes", json_data=body)
        return [ColumnConsentedPurposes.from_json(p) for p in resp_json["data"]]

    # Userstore user methods (should be used along with Accessor and Mutator methods)

    def CreateUserWithMutator(
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
        return self._post("/userstore/api/users", json_data=body)

    # ColumnDataType Operations

    def CreateColumnDataType(
        self, dataType: ColumnDataType, if_not_exists: bool = False
    ) -> ColumnDataType:
        try:
            resp_json = self._post(
                "/userstore/config/datatypes",
                json_data={"data_type": dataType.__dict__},
            )
            return ColumnDataType.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                dataType.id = _id_from_identical_conflict(err)
                return dataType
            raise err

    def DeleteColumnDataType(self, id: uuid.UUID) -> bool:
        return self._delete(f"/userstore/config/datatypes/{id}")

    def GetColumnDataType(self, id: uuid.UUID) -> ColumnDataType:
        resp_json = self._get(f"/userstore/config/datatypes/{id}")
        return ColumnDataType.from_json(resp_json)

    def ListColumnDataTypes(
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
        resp_json = self._get("/userstore/config/datatypes", params=params)
        dataTypes = [
            ColumnDataType.from_json(dataType) for dataType in resp_json["data"]
        ]
        return dataTypes

    def UpdateColumnDataType(self, dataType: ColumnDataType) -> ColumnDataType:
        resp_json = self._put(
            f"/userstore/config/datatypes/{dataType.id}",
            json_data={"data_type": dataType.__dict__},
        )
        return ColumnDataType.from_json(resp_json)

    # Column Operations

    def CreateColumn(self, column: Column, if_not_exists: bool = False) -> Column:
        try:
            resp_json = self._post(
                "/userstore/config/columns", json_data={"column": column.__dict__}
            )
            return Column.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                column.id = _id_from_identical_conflict(err)
                return column
            raise err

    def DeleteColumn(self, id: uuid.UUID) -> bool:
        return self._delete(f"/userstore/config/columns/{id}")

    def GetColumn(self, id: uuid.UUID) -> Column:
        resp_json = self._get(f"/userstore/config/columns/{id}")
        return Column.from_json(resp_json)

    def ListColumns(
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
        resp_json = self._get("/userstore/config/columns", params=params)
        columns = [Column.from_json(col) for col in resp_json["data"]]
        return columns

    def UpdateColumn(self, column: Column) -> Column:
        resp_json = self._put(
            f"/userstore/config/columns/{column.id}",
            json_data={"column": column.__dict__},
        )
        return Column.from_json(resp_json)

    # Purpose Operations

    def CreatePurpose(self, purpose: Purpose, if_not_exists: bool = False) -> Purpose:
        try:
            resp_json = self._post(
                "/userstore/config/purposes", json_data={"purpose": purpose.__dict__}
            )
            return Purpose.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                purpose.id = _id_from_identical_conflict(err)
                return purpose
            raise err

    def DeletePurpose(self, id: uuid.UUID) -> bool:
        return self._delete(f"/userstore/config/purposes/{id}")

    def GetPurpose(self, id: uuid.UUID) -> Purpose:
        json_resp = self._get(f"/userstore/config/purposes/{id}")
        return Purpose.from_json(json_resp)

    def ListPurposes(
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
        resp_json = self._get("/userstore/config/purposes", params=params)
        purposes = [Purpose.from_json(p) for p in resp_json["data"]]
        return purposes

    def UpdatePurpose(self, purpose: Purpose) -> Purpose:
        resp_json = self._put(
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
    def CreateSoftDeletedRetentionDurationOnTenant(
        self, req: UpdateColumnRetentionDurationRequest
    ) -> ColumnRetentionDurationResponse:
        resp = self._post(
            "/userstore/config/softdeletedretentiondurations", json_data=req.to_json()
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # delete a tenant retention duration default
    def DeleteSoftDeletedRetentionDurationOnTenant(self, durationID: uuid.UUID) -> bool:
        return self._delete(
            f"/userstore/config/softdeletedretentiondurations/{durationID}"
        )

    # get a specific tenant retention duration default
    def GetSoftDeletedRetentionDurationOnTenant(
        self, durationID: uuid.UUID
    ) -> ColumnRetentionDurationResponse:
        resp = self._get(
            f"/userstore/config/softdeletedretentiondurations/{durationID}"
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # get tenant retention duration, or default value if not specified
    def GetDefaultSoftDeletedRetentionDurationOnTenant(
        self,
    ) -> ColumnRetentionDurationResponse:
        resp = self._get("/userstore/config/softdeletedretentiondurations")
        return ColumnRetentionDurationResponse.from_json(resp)

    # update a specific tenant retention duration default
    def UpdateSoftDeletedRetentionDurationOnTenant(
        self, durationID: uuid.UUID, req: UpdateColumnRetentionDurationRequest
    ) -> ColumnRetentionDurationResponse:
        resp = self._put(
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
    def CreateSoftDeletedRetentionDurationOnPurpose(
        self, purposeID: uuid.UUID, req: UpdateColumnRetentionDurationRequest
    ) -> ColumnRetentionDurationResponse:
        resp = self._post(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # delete a purpose retention duration default
    def DeleteSoftDeletedRetentionDurationOnPurpose(
        self, purposeID: uuid.UUID, durationID: uuid.UUID
    ) -> bool:
        return self._delete(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations/{durationID}"
        )

    # get a specific purpose retention duration default
    def GetSoftDeletedRetentionDurationOnPurpose(
        self, purposeID: uuid.UUID, durationID: uuid.UUID
    ) -> ColumnRetentionDurationResponse:
        resp = self._get(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations/{durationID}"
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # get purpose retention duration, or default value if not specified
    def GetDefaultSoftDeletedRetentionDurationOnPurpose(
        self, purposeID: uuid.UUID
    ) -> ColumnRetentionDurationResponse:
        resp = self._get(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations"
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # update a specific purpose retention duration default
    def UpdateSoftDeletedRetentionDurationOnPurpose(
        self,
        purposeID: uuid.UUID,
        durationID: uuid.UUID,
        req: UpdateColumnRetentionDurationRequest,
    ) -> ColumnRetentionDurationResponse:
        resp = self._put(
            f"/userstore/config/purposes/{purposeID}/softdeletedretentiondurations/{durationID}",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # Column Retention Durations

    # A configured column purpose retention duration will override
    # any configured purpose level, tenant level, or system-level
    # default retention durations.

    # get a specific column purpose retention duration
    def GetSoftDeletedRetentionDurationOnColumn(
        self, columnID: uuid.UUID, durationID: uuid.UUID
    ) -> ColumnRetentionDurationResponse:
        resp = self._get(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations/{durationID}"
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # get the retention duration for each purpose for a given column
    def GetSoftDeletedRetentionDurationsOnColumn(
        self, columnID: uuid.UUID
    ) -> ColumnRetentionDurationsResponse:
        resp = self._get(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations"
        )
        return ColumnRetentionDurationsResponse.from_json(resp)

    # delete a specific column purpose retention duration
    def DeleteSoftDeletedRetentionDurationOnColumn(
        self, columnID: uuid.UUID, durationID: uuid.UUID
    ) -> bool:
        return self._delete(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations/{durationID}"
        )

    # update a specific column purpose retention duration
    def UpdateSoftDeletedRetentionDurationOnColumn(
        self,
        columnID: uuid.UUID,
        durationID: uuid.UUID,
        req: UpdateColumnRetentionDurationRequest,
    ) -> ColumnRetentionDurationResponse:
        resp = self._put(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations/{durationID}",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationResponse.from_json(resp)

    # update the specified purpose retention durations for the column
    # - durations can be added, deleted, or updated for each purpose
    def UpdateSoftDeletedRetentionDurationsOnColumn(
        self, columnID: uuid.UUID, req: UpdateColumnRetentionDurationsRequest
    ) -> ColumnRetentionDurationsResponse:
        resp = self._post(
            f"/userstore/config/columns/{columnID}/softdeletedretentiondurations",
            json_data=req.to_json(),
        )
        return ColumnRetentionDurationsResponse.from_json(resp)

    # Access Policy Templates

    def CreateAccessPolicyTemplate(
        self, access_policy_template: AccessPolicyTemplate, if_not_exists: bool = False
    ) -> AccessPolicyTemplate | UserCloudsSDKError:
        try:
            resp_json = self._post(
                "/tokenizer/policies/accesstemplate",
                json_data={"access_policy_template": access_policy_template.__dict__},
            )
            return AccessPolicyTemplate.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                access_policy_template.id = _id_from_identical_conflict(err)
                return access_policy_template
            raise err

    def ListAccessPolicyTemplates(
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
        resp_json = self._get("/tokenizer/policies/accesstemplate", params=params)

        templates = [AccessPolicyTemplate.from_json(apt) for apt in resp_json["data"]]
        return templates

    def GetAccessPolicyTemplate(self, rid: ResourceID):
        if hasattr(rid, "id"):
            resp_json = self._get(f"/tokenizer/policies/accesstemplate/{rid.id}")
            return AccessPolicyTemplate.from_json(resp_json)
        elif hasattr(rid, "name"):
            resp_json = self._get(
                f"/tokenizer/policies/accesstemplate?template_name={rid.name}"
            )
            if len(resp_json["data"]) == 1:
                return AccessPolicyTemplate.from_json(resp_json["data"][0])
            raise UserCloudsSDKError(
                f"Access Policy Template with name {rid.name} not found", 404
            )
        raise UserCloudsSDKError("Invalid ResourceID", 400)

    def UpdateAccessPolicyTemplate(self, access_policy_template: AccessPolicyTemplate):
        resp_json = self._put(
            f"/tokenizer/policies/accesstemplate/{access_policy_template.id}",
            json_data={"access_policy_template": access_policy_template.__dict__},
        )
        return AccessPolicyTemplate.from_json(resp_json)

    def DeleteAccessPolicyTemplate(self, id: uuid.UUID, version: int):
        return self._delete(
            f"/tokenizer/policies/accesstemplate/{id}",
            params={"template_version": str(version)},
        )

    # Access Policies

    def CreateAccessPolicy(
        self, access_policy: AccessPolicy, if_not_exists: bool = False
    ) -> AccessPolicy | UserCloudsSDKError:
        try:
            resp_json = self._post(
                "/tokenizer/policies/access",
                json_data={"access_policy": access_policy.__dict__},
            )
            return AccessPolicy.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                access_policy.id = _id_from_identical_conflict(err)
                return access_policy
            raise err

    def ListAccessPolicies(
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
        resp_json = self._get("/tokenizer/policies/access", params=params)

        policies = [AccessPolicy.from_json(ap) for ap in resp_json["data"]]
        return policies

    def GetAccessPolicy(self, rid: ResourceID):
        if hasattr(rid, "id"):
            resp_json = self._get(f"/tokenizer/policies/access/{rid.id}")
            return AccessPolicy.from_json(resp_json)
        elif hasattr(rid, "name"):
            resp_json = self._get(f"/tokenizer/policies/access?policy_name={rid.name}")
            if len(resp_json["data"]) == 1:
                return AccessPolicy.from_json(resp_json["data"][0])
            raise UserCloudsSDKError(
                f"Access Policy with name {rid.name} not found", 404
            )
        raise UserCloudsSDKError("Invalid ResourceID", 400)

    def UpdateAccessPolicy(self, access_policy: AccessPolicy):
        resp_json = self._put(
            f"/tokenizer/policies/access/{access_policy.id}",
            json_data={"access_policy": access_policy.__dict__},
        )
        return AccessPolicy.from_json(resp_json)

    def DeleteAccessPolicy(self, id: uuid.UUID, version: int):
        return self._delete(
            f"/tokenizer/policies/access/{id}",
            params={"policy_version": str(version)},
        )

    # Transformers

    def CreateTransformer(self, transformer: Transformer, if_not_exists: bool = False):
        try:
            resp_json = self._post(
                "/tokenizer/policies/transformation",
                json_data={"transformer": transformer.__dict__},
            )
            return Transformer.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                transformer.id = _id_from_identical_conflict(err)
                return transformer
            raise err

    def GetTransformer(self, rid: ResourceID):
        if hasattr(rid, "id"):
            resp_json = self._get(f"/tokenizer/policies/transformation/{rid.id}")
            return Transformer.from_json(resp_json)
        elif hasattr(rid, "name"):
            resp_json = self._get(
                f"/tokenizer/policies/transformation?transformer_name={rid.name}"
            )
            if len(resp_json["data"]) == 1:
                return Transformer.from_json(resp_json["data"][0])
            raise UserCloudsSDKError(f"Transformer with name {rid.name} not found", 404)
        raise UserCloudsSDKError("Invalid ResourceID", 400)

    def ListTransformers(
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
        resp_json = self._get("/tokenizer/policies/transformation", params=params)
        transformers = [Transformer.from_json(tf) for tf in resp_json["data"]]
        return transformers

    def UpdateTransformer(self, transformer: Transformer):
        resp_json = self._put(
            f"/tokenizer/policies/transformation/{transformer.id}",
            json_data={"transformer": transformer.__dict__},
        )
        return Transformer.from_json(resp_json)

    def DeleteTransformer(self, id: uuid.UUID):
        return self._delete(f"/tokenizer/policies/transformation/{id}")

    # Accessor Operations

    def CreateAccessor(
        self, accessor: Accessor, if_not_exists: bool = False
    ) -> Accessor:
        try:
            resp_json = self._post(
                "/userstore/config/accessors", json_data={"accessor": accessor.__dict__}
            )
            return Accessor.from_json(resp_json)
        except UserCloudsSDKError as err:
            if if_not_exists:
                accessor.id = _id_from_identical_conflict(err)
                return accessor
            raise err

    def DeleteAccessor(self, id: uuid.UUID) -> bool:
        return self._delete(f"/userstore/config/accessors/{id}")

    def GetAccessor(self, id: uuid.UUID) -> Accessor:
        j = self._get(f"/userstore/config/accessors/{id}")
        return Accessor.from_json(j)

    def ListAccessors(
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
        resp_json = self._get("/userstore/config/accessors", params=params)

        accessors = [Accessor.from_json(acs) for acs in resp_json["data"]]
        return accessors

    def UpdateAccessor(self, accessor: Accessor) -> Accessor:
        resp_json = self._put(
            f"/userstore/config/accessors/{accessor.id}",
            json_data={"accessor": accessor.__dict__},
        )
        return Accessor.from_json(resp_json)

    def ExecuteAccessor(
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

        return self._post("/userstore/api/accessors", json_data=body, params=params)

    # Mutator Operations

    def CreateMutator(self, mutator: Mutator, if_not_exists: bool = False) -> Mutator:
        try:
            resp_json = self._post(
                "/userstore/config/mutators", json_data={"mutator": mutator.__dict__}
            )
            return Mutator.from_json(resp_json)
        except UserCloudsSDKError as e:
            if if_not_exists:
                mutator.id = _id_from_identical_conflict(e)
                return mutator
            raise e

    def DeleteMutator(self, id: uuid.UUID) -> bool:
        return self._delete(f"/userstore/config/mutators/{id}")

    def GetMutator(self, id: uuid.UUID) -> Mutator:
        resp_json = self._get(f"/userstore/config/mutators/{id}")
        return Mutator.from_json(resp_json)

    def ListMutators(
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
        j = self._get("/userstore/config/mutators", params=params)

        mutators = [Mutator.from_json(m) for m in j["data"]]
        return mutators

    def UpdateMutator(self, mutator: Mutator) -> Mutator:
        json_resp = self._put(
            f"/userstore/config/mutators/{mutator.id}",
            json_data={"mutator": mutator.__dict__},
        )
        return Mutator.from_json(json_resp)

    def ExecuteMutator(
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

        j = self._post("/userstore/api/mutators", json_data=body)
        return j

    # Token Operations

    def CreateToken(
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

        json_resp = self._post("/tokenizer/tokens", json_data=body)
        return json_resp["data"]

    def LookupOrCreateTokens(
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

        j = self._post("/tokenizer/tokens/actions/lookuporcreate", json_data=body)
        return j["tokens"]

    def ResolveTokens(
        self, tokens: list[str], context: dict, purposes: list[ResourceID]
    ) -> list[str]:
        body = {"tokens": tokens, "context": context, "purposes": purposes}

        j = self._post("/tokenizer/tokens/actions/resolve", json_data=body)
        return j

    def DeleteToken(self, token: str) -> bool:
        return self._delete("/tokenizer/tokens", params={"token": token})

    def InspectToken(self, token: str) -> InspectTokenResponse:
        body = {"token": token}

        j = self._post("/tokenizer/tokens/actions/inspect", json_data=body)
        return InspectTokenResponse.from_json(j)

    def LookupToken(
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

        j = self._post("/tokenizer/tokens/actions/lookup", json_data=body)
        return j["tokens"]

    # AuthZ Operations

    def ListObjects(
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
        j = self._get("/authz/objects", params=params)

        objects = [Object.from_json(o) for o in j["data"]]
        return objects

    def CreateObject(self, object: Object, if_not_exists: bool = False) -> Object:
        try:
            j = self._post("/authz/objects", json_data={"object": object.__dict__})
            return Object.from_json(j)
        except UserCloudsSDKError as e:
            if if_not_exists:
                object.id = _id_from_identical_conflict(e)
                return object
            raise e

    def GetObject(self, id: uuid.UUID) -> Object:
        j = self._get(f"/authz/objects/{id}")
        return Object.from_json(j)

    def DeleteObject(self, id: uuid.UUID):
        return self._delete(f"/authz/objects/{id}")

    def ListEdges(
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
        j = self._get("/authz/edges", params=params)

        edges = [Edge.from_json(e) for e in j["data"]]
        return edges

    def CreateEdge(self, edge: Edge, if_not_exists: bool = False) -> Edge:
        try:
            j = self._post("/authz/edges", json_data={"edge": edge.__dict__})
            return Edge.from_json(j)
        except UserCloudsSDKError as e:
            if if_not_exists:
                edge.id = _id_from_identical_conflict(e)
                return edge
            raise e

    def GetEdge(self, id: uuid.UUID) -> Edge:
        j = self._get(f"/authz/edges/{id}")
        return Edge.from_json(j)

    def DeleteEdge(self, id: uuid.UUID):
        return self._delete(f"/authz/edges/{id}")

    def ListObjectTypes(
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
        j = self._get("/authz/objecttypes", params=params)

        object_types = [ObjectType.from_json(ot) for ot in j["data"]]
        return object_types

    def CreateObjectType(
        self, object_type: ObjectType, if_not_exists: bool = False
    ) -> ObjectType:
        try:
            j = self._post(
                "/authz/objecttypes", json_data={"object_type": object_type.__dict__}
            )
            return ObjectType.from_json(j)
        except UserCloudsSDKError as e:
            if if_not_exists:
                object_type.id = _id_from_identical_conflict(e)
                return object_type
            raise e

    def GetObjectType(self, id: uuid.UUID) -> ObjectType:
        j = self._get(f"/authz/objecttypes/{id}")
        return ObjectType.from_json(j)

    def DeleteObjectType(self, id: uuid.UUID):
        return self._delete(f"/authz/objecttypes/{id}")

    def ListEdgeTypes(
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
        j = self._get("/authz/edgetypes", params=params)

        edge_types = [EdgeType.from_json(et) for et in j["data"]]
        return edge_types

    def CreateEdgeType(
        self, edge_type: EdgeType, if_not_exists: bool = False
    ) -> EdgeType:
        try:
            j = self._post(
                "/authz/edgetypes", json_data={"edge_type": edge_type.__dict__}
            )
            return EdgeType.from_json(j)
        except UserCloudsSDKError as e:
            if if_not_exists:
                edge_type.id = _id_from_identical_conflict(e)
                return edge_type
            raise e

    def GetEdgeType(self, id: uuid.UUID) -> EdgeType:
        j = self._get(f"/authz/edgetypes/{id}")
        return EdgeType.from_json(j)

    def DeleteEdgeType(self, id: uuid.UUID) -> bool:
        return self._delete(f"/authz/edgetypes/{id}")

    def ListOrganizations(
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
        j = self._get("/authz/organizations", params=params)

        organizations = [Organization.from_json(o) for o in j["data"]]
        return organizations

    def CreateOrganization(
        self, organization: Organization, if_not_exists: bool = False
    ) -> Organization:
        try:
            json_data = self._post(
                "/authz/organizations",
                json_data={"organization": organization.__dict__},
            )
            return Organization.from_json(json_data)
        except UserCloudsSDKError as e:
            if if_not_exists:
                organization.id = _id_from_identical_conflict(e)
                return organization
            raise e

    def GetOrganization(self, id: uuid.UUID) -> Organization:
        json_data = self._get(f"/authz/organizations/{id}")
        return Organization.from_json(json_data)

    def DeleteOrganization(self, id: uuid.UUID) -> bool:
        return self._delete(f"/authz/organizations/{id}")

    def CheckAttribute(
        self,
        source_object_id: uuid.UUID,
        target_object_id: uuid.UUID,
        attribute_name: str,
    ) -> bool:
        j = self._get(
            f"/authz/checkattribute?source_object_id={source_object_id}&target_object_id={target_object_id}&attribute={attribute_name}"
        )
        return j.get("has_attribute")

    def DownloadUserstoreSDK(self, include_example=True) -> str:
        return self._download(
            f"/userstore/download/codegensdk.py?include_example={include_example and 'true' or 'false'}"
        )

    def SaveUserstoreSDK(self, path: Path, include_example: bool = False) -> None:
        sdk = self.DownloadUserstoreSDK(include_example=include_example)
        path.write_text(sdk)

    def GetExternalOIDCIssuers(self) -> list[str]:
        return self._get("/userstore/oidcissuers")

    def UpdateExternalOIDCIssuers(self, issuers: list[str]) -> list[str]:
        return self._put("/userstore/oidcissuers", json_data=issuers)

    def UploadDataImportFile(
        self, file_path: Path, import_type: str = "executemutator"
    ) -> uuid.UUID:
        json_data = self._get(f"/userstore/upload/dataimport?import_type={import_type}")
        import_id = uuid.UUID(json_data["import_id"])
        with open(file_path, "rb") as f:
            resp = self._client.put(json_data["presigned_url"], content=f)
            if resp.status_code >= 400:
                raise UserCloudsSDKError.from_response(resp)
        return import_id

    def CheckDataImportStatus(self, import_id: uuid.UUID) -> dict:
        json_data = self._get(f"/userstore/upload/dataimport/{import_id}")
        return json_data

    # Access Token Helpers

    def _get_access_token(self) -> str:
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
        resp = self._client.post("/oidc/token", headers=headers, content=body)
        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        json_data = ucjson.loads(resp.text)
        token = json_data.get("access_token")
        if self._use_global_cache_for_token:
            cache_token(self._authorization, token)
        return token

    def _refresh_access_token_if_needed(self) -> None:
        if self._access_token is None:
            self._access_token = self._get_access_token()
            return

        if is_token_expiring(self._access_token):
            self._access_token = self._get_access_token()

    # Request Helpers

    def _get_headers(self) -> dict:
        headers = {"Authorization": f"Bearer {self._access_token}"}
        headers.update(self._common_headers)
        return headers

    def _prep_json_data(self, json_data: dict | str | None) -> tuple[dict, str | None]:
        self._refresh_access_token_if_needed()
        headers = self._get_headers()
        content = None
        if json_data is not None:
            headers["Content-Type"] = _JSON_CONTENT_TYPE
            content = (
                json_data if isinstance(json_data, str) else ucjson.dumps(json_data)
            )
        return headers, content

    def _get(self, url, params: dict[str, str | int] | None = None) -> dict:
        self._refresh_access_token_if_needed()
        resp = self._client.get(url, params=params, headers=self._get_headers())
        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        return ucjson.loads(resp.text)

    def _post(
        self,
        url,
        json_data: dict | str | None = None,
        params: dict[str, str | int] | None = None,
    ) -> dict | list:
        headers, content = self._prep_json_data(json_data)
        resp = self._client.post(url, params=params, headers=headers, content=content)
        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        return ucjson.loads(resp.text)

    def _put(self, url, json_data: dict | str | None = None) -> dict | list:
        headers, content = self._prep_json_data(json_data)
        resp = self._client.put(url, headers=headers, content=content)
        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        return ucjson.loads(resp.text)

    def _delete(self, url, params: dict[str, str | int] | None = None) -> bool:
        self._refresh_access_token_if_needed()
        resp = self._client.delete(url, params=params, headers=self._get_headers())

        if resp.status_code == 404:
            return False

        if resp.status_code >= 400:
            raise UserCloudsSDKError.from_response(resp)
        return resp.status_code == 204

    def _download(self, url) -> str:
        self._refresh_access_token_if_needed()
        resp = self._client.get(url, headers=self._get_headers())
        return resp.text
