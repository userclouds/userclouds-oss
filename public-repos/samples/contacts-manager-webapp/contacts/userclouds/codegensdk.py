# *** WARNING! ***
# This file is auto-generated and will be overwritten when a schema is modified.
# DO NOT EDIT.
#
# Origin: seinfeldenterprises-contacts.tenant.dev.userclouds.tools:3333
# Generated at: 2024-02-12T18:03:28Z
from __future__ import annotations

import datetime
import enum
import json
import uuid
from dataclasses import dataclass

from usercloudssdk.client import Client


@dataclass
class TokenizedPIIObject:

    email: str | None = None
    phone_number: str | None = None


@dataclass
class TokenizedUserDataObject:

    email: str | None = None
    id: uuid.UUID | None = None
    name: str | None = None
    nickname: str | None = None
    phone_number: str | None = None


@dataclass
class UsersObject:

    created: datetime.datetime | None = None
    email: str | None = None
    id: uuid.UUID | None = None
    name: str | None = None
    nickname: str | None = None
    organization_id: uuid.UUID | None = None
    phone_number: str | None = None
    picture: str | None = None
    updated: datetime.datetime | None = None
    version: int | None = None


@dataclass
class SaveUserObject:

    email: str | None = None
    name: str | None = None
    nickname: str | None = None
    phone_number: str | None = None


@dataclass
class UserObject:

    email: str | None = None
    name: str | None = None
    nickname: str | None = None
    phone_number: str | None = None
    picture: str | None = None


def GetTokenizedPII(self, id: str) -> [TokenizedPIIObject]:
    resp = self.ExecuteAccessor("8f3707e5-b0f4-420f-95a5-a6afb4faf11d", {}, [id])

    ret = []
    for resp_data in resp["data"]:
        json_data = json.loads(resp_data)
        obj = TokenizedPIIObject()
        if json_data.get("email"):
            obj.email = json_data["email"]
        if json_data.get("phone_number"):
            obj.phone_number = json_data["phone_number"]

        ret.append(obj)

    return ret


Client.GetTokenizedPII = GetTokenizedPII


def GetTokenizedUserData(self, id: list[str]) -> [TokenizedUserDataObject]:
    resp = self.ExecuteAccessor("8a83be40-59be-4a8a-89d8-dd82c1f83412", {}, [id])

    ret = []
    for resp_data in resp["data"]:
        json_data = json.loads(resp_data)
        obj = TokenizedUserDataObject()
        if json_data.get("email"):
            obj.email = json_data["email"]
        if json_data.get("id"):
            obj.id = uuid.UUID(json_data["id"])
        if json_data.get("name"):
            obj.name = json_data["name"]
        if json_data.get("nickname"):
            obj.nickname = json_data["nickname"]
        if json_data.get("phone_number"):
            obj.phone_number = json_data["phone_number"]

        ret.append(obj)

    return ret


Client.GetTokenizedUserData = GetTokenizedUserData


def GetUsers(self, id: list[str]) -> [UsersObject]:
    resp = self.ExecuteAccessor("28bf0486-9eea-4db5-ba40-5cef12dd48db", {}, [id])

    ret = []
    for resp_data in resp["data"]:
        json_data = json.loads(resp_data)
        obj = UsersObject()
        if json_data.get("created"):
            obj.created = datetime.datetime.strptime(
                json_data["created"], "%Y-%m-%dT%H:%M:%SZ"
            )
        if json_data.get("email"):
            obj.email = json_data["email"]
        if json_data.get("id"):
            obj.id = uuid.UUID(json_data["id"])
        if json_data.get("name"):
            obj.name = json_data["name"]
        if json_data.get("nickname"):
            obj.nickname = json_data["nickname"]
        if json_data.get("organization_id"):
            obj.organization_id = uuid.UUID(json_data["organization_id"])
        if json_data.get("phone_number"):
            obj.phone_number = json_data["phone_number"]
        if json_data.get("picture"):
            obj.picture = json_data["picture"]
        if json_data.get("updated"):
            obj.updated = datetime.datetime.strptime(
                json_data["updated"], "%Y-%m-%dT%H:%M:%SZ"
            )
        if json_data.get("version"):
            obj.version = int(json_data["version"])

        ret.append(obj)

    return ret


Client.GetUsers = GetUsers


def SetSaveUserObjectForOperationalPurpose(self, obj: SaveUserObject, id: str) -> list:
    row_data = {}
    row_data["email"] = {
        "value": obj.email,
        "purpose_additions": [{"name": "operational"}],
    }
    row_data["name"] = {
        "value": obj.name,
        "purpose_additions": [{"name": "operational"}],
    }
    row_data["nickname"] = {
        "value": obj.nickname,
        "purpose_additions": [{"name": "operational"}],
    }
    row_data["phone_number"] = {
        "value": obj.phone_number,
        "purpose_additions": [{"name": "operational"}],
    }

    resp = self.ExecuteMutator(
        "17e37de3-c1dc-40d8-9087-e02044ce1470", {}, [id], row_data
    )
    return resp


Client.SetSaveUserObjectForOperationalPurpose = SetSaveUserObjectForOperationalPurpose


def SetSaveUserObjectForMarketingPurpose(self, obj: SaveUserObject, id: str) -> list:
    row_data = {}
    row_data["email"] = {
        "value": obj.email,
        "purpose_additions": [{"name": "marketing"}],
    }
    row_data["name"] = {"value": obj.name, "purpose_additions": [{"name": "marketing"}]}
    row_data["nickname"] = {
        "value": obj.nickname,
        "purpose_additions": [{"name": "marketing"}],
    }
    row_data["phone_number"] = {
        "value": obj.phone_number,
        "purpose_additions": [{"name": "marketing"}],
    }

    resp = self.ExecuteMutator(
        "17e37de3-c1dc-40d8-9087-e02044ce1470", {}, [id], row_data
    )
    return resp


Client.SetSaveUserObjectForMarketingPurpose = SetSaveUserObjectForMarketingPurpose


def SetSaveUserObjectForFraudPurpose(self, obj: SaveUserObject, id: str) -> list:
    row_data = {}
    row_data["email"] = {"value": obj.email, "purpose_additions": [{"name": "fraud"}]}
    row_data["name"] = {"value": obj.name, "purpose_additions": [{"name": "fraud"}]}
    row_data["nickname"] = {
        "value": obj.nickname,
        "purpose_additions": [{"name": "fraud"}],
    }
    row_data["phone_number"] = {
        "value": obj.phone_number,
        "purpose_additions": [{"name": "fraud"}],
    }

    resp = self.ExecuteMutator(
        "17e37de3-c1dc-40d8-9087-e02044ce1470", {}, [id], row_data
    )
    return resp


Client.SetSaveUserObjectForFraudPurpose = SetSaveUserObjectForFraudPurpose


def UpdateUserForOperationalPurpose(self, obj: UserObject, id: str) -> list:
    row_data = {}
    row_data["email"] = {
        "value": obj.email,
        "purpose_additions": [{"name": "operational"}],
    }
    row_data["name"] = {
        "value": obj.name,
        "purpose_additions": [{"name": "operational"}],
    }
    row_data["nickname"] = {
        "value": obj.nickname,
        "purpose_additions": [{"name": "operational"}],
    }
    row_data["phone_number"] = {
        "value": obj.phone_number,
        "purpose_additions": [{"name": "operational"}],
    }
    row_data["picture"] = {
        "value": obj.picture,
        "purpose_additions": [{"name": "operational"}],
    }

    resp = self.ExecuteMutator(
        "45c07eb8-f042-4cc8-b50e-3e6eb7b34bdc", {}, [id], row_data
    )
    return resp


Client.UpdateUserForOperationalPurpose = UpdateUserForOperationalPurpose


def UpdateUserForMarketingPurpose(self, obj: UserObject, id: str) -> list:
    row_data = {}
    row_data["email"] = {
        "value": obj.email,
        "purpose_additions": [{"name": "marketing"}],
    }
    row_data["name"] = {"value": obj.name, "purpose_additions": [{"name": "marketing"}]}
    row_data["nickname"] = {
        "value": obj.nickname,
        "purpose_additions": [{"name": "marketing"}],
    }
    row_data["phone_number"] = {
        "value": obj.phone_number,
        "purpose_additions": [{"name": "marketing"}],
    }
    row_data["picture"] = {
        "value": obj.picture,
        "purpose_additions": [{"name": "marketing"}],
    }

    resp = self.ExecuteMutator(
        "45c07eb8-f042-4cc8-b50e-3e6eb7b34bdc", {}, [id], row_data
    )
    return resp


Client.UpdateUserForMarketingPurpose = UpdateUserForMarketingPurpose


def UpdateUserForFraudPurpose(self, obj: UserObject, id: str) -> list:
    row_data = {}
    row_data["email"] = {"value": obj.email, "purpose_additions": [{"name": "fraud"}]}
    row_data["name"] = {"value": obj.name, "purpose_additions": [{"name": "fraud"}]}
    row_data["nickname"] = {
        "value": obj.nickname,
        "purpose_additions": [{"name": "fraud"}],
    }
    row_data["phone_number"] = {
        "value": obj.phone_number,
        "purpose_additions": [{"name": "fraud"}],
    }
    row_data["picture"] = {
        "value": obj.picture,
        "purpose_additions": [{"name": "fraud"}],
    }

    resp = self.ExecuteMutator(
        "45c07eb8-f042-4cc8-b50e-3e6eb7b34bdc", {}, [id], row_data
    )
    return resp


Client.UpdateUserForFraudPurpose = UpdateUserForFraudPurpose


class Purpose(enum.Enum):
    OPERATIONAL = "operational"
    MARKETING = "marketing"
    FRAUD = "fraud"


def SetSaveUserObjectForPurposes(
    self, purposes: list[Purpose], obj: SaveUserObject, id: str
) -> list:
    row_data = {}
    purpose_additions = list(map(lambda p: {"name": p.value}, purposes))
    row_data["email"] = {"value": obj.email, "purpose_additions": purpose_additions}
    row_data["name"] = {"value": obj.name, "purpose_additions": purpose_additions}
    row_data["nickname"] = {
        "value": obj.nickname,
        "purpose_additions": purpose_additions,
    }
    row_data["phone_number"] = {
        "value": obj.phone_number,
        "purpose_additions": purpose_additions,
    }

    resp = self.ExecuteMutator(
        "17e37de3-c1dc-40d8-9087-e02044ce1470", {}, [id], row_data
    )
    return resp


Client.SetSaveUserObjectForPurposes = SetSaveUserObjectForPurposes


def UpdateUserForPurposes(
    self, purposes: list[Purpose], obj: UserObject, id: str
) -> list:
    row_data = {}
    purpose_additions = list(map(lambda p: {"name": p.value}, purposes))
    row_data["email"] = {"value": obj.email, "purpose_additions": purpose_additions}
    row_data["name"] = {"value": obj.name, "purpose_additions": purpose_additions}
    row_data["nickname"] = {
        "value": obj.nickname,
        "purpose_additions": purpose_additions,
    }
    row_data["phone_number"] = {
        "value": obj.phone_number,
        "purpose_additions": purpose_additions,
    }
    row_data["picture"] = {"value": obj.picture, "purpose_additions": purpose_additions}

    resp = self.ExecuteMutator(
        "45c07eb8-f042-4cc8-b50e-3e6eb7b34bdc", {}, [id], row_data
    )
    return resp


Client.UpdateUserForPurposes = UpdateUserForPurposes


def example():
    # Init client from env variables: TENANT_URL, CLIENT_ID & CLIENT_SECRET
    client = Client.from_env()

    uid = client.CreateUser()
    print(f"Created user with id: {uid}")

    users = client.GetUsers([uid])
    user = users[0]
    print(f"User: {user}")

    user.name = "New Name"
    client.UpdateUserForOperationalPurpose(user, uid)
    users = client.GetUsers([uid])
    user = users[0]
    print(f"User: {user}")
