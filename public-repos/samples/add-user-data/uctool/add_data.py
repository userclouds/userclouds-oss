from __future__ import annotations

import argparse
import sys
import time
import uuid

from faker import Faker
from usercloudssdk.client import Client, _read_env
from usercloudssdk.errors import UserCloudsSDKError
from usercloudssdk.models import Column, ResourceID
from usercloudssdk.uchttpclient import (
    create_default_uc_http_client,
    create_no_ssl_http_client,
)

UpdateUserMutatorID = uuid.UUID("45c07eb8-f042-4cc8-b50e-3e6eb7b34bdc")

_COLS_TO_IGNORE = [
    # used by system/UserClouds
    "id",
    "created",
    "updated",
    "organization_id",
    "version",
]

_TYPE_MAP = {  # maps UC column types to faker methods
    "phonenumber": "basic_phone_number",
    "birthdate": "date",
    "date": "date",
    "address": "uc_address",
    "email": "free_email",
    "uuid": "uuid4",
    "boolean": "boolean",
    "ssn": "ssn",
    "integer": "random_int",
    "timestamp": "uc_timestamp",
    "e164_phonenumber": "uc_e164_phone_number",
}
_NAME_MAP = {  # maps UC column names to faker methods
    "nickname": "user_name",
    "fullname": "name",
    "full_name": "name",
    "first_name": "first_name",
    "firstname": "first_name",
    "given_name": "first_name",
    "givenname": "first_name",
    "last_name": "last_name",
    "lastname": "last_name",
    "surname": "last_name",
    "ssn": "ssn",
    "name": "name",
    "email": "free_email",
    "picture": "uri",
    "external_alias": "uuid4",
}


def uc_address(self) -> dict[str, str]:
    state = self.state_abbr()
    return {
        "street_address_line_1": self.street_address(),
        "street_address_line_2": self.secondary_address(),
        "country": "US",
        "locality": self.city(),
        "post_code": self.zipcode_in_state(state),
        "administrative_area": state,
    }


def uc_timestamp(self) -> str:
    return self.date_time().strftime("%Y-%m-%dT%H:%M:%SZ")


def uc_e164_phone_number(self) -> str:
    return (
        f"+1{ self.basic_phone_number()}".replace("-", "")
        .replace("(", "")
        .replace(")", "")
        .replace(" ", "")
    )


Faker.uc_address = uc_address
Faker.uc_timestamp = uc_timestamp
Faker.uc_e164_phone_number = uc_e164_phone_number


def get_array_faker_func(faker_func: callable, fk: Faker) -> callable:
    def array_faker() -> list[str]:
        return [faker_func() for _ in range(fk.random_int(min=1, max=15))]

    return array_faker


class ColumnHelper:
    def __init__(self, columns: list[Column]) -> None:
        self._faker = Faker("en-US")

        self._column_fakers = self._get_column_fakers(columns)

    def get_user(self) -> dict[str, str]:
        return {col: faker() for col, faker in self._column_fakers.items()}

    def _get_column_fakers(self, columns: list[Column]) -> dict[str, callable]:
        missing: list[str] = []
        fakers: dict[str, callable] = {}
        for col in columns:
            faker = self._get_column_faker(col)
            if isinstance(faker, str):
                missing.append(faker)
            elif faker:
                fakers[col.name.lower()] = faker
        self._handle_missing_fakers(missing)
        return fakers

    def _handle_missing_fakers(self, missing: list[str]) -> None:
        if not missing:
            return
        print("*" * 30)
        print("Missing faker methods:")
        print("\n".join(missing))
        print("*" * 30)
        raise ValueError("Missing, update code to add matchers")

    def _get_column_faker(self, col: Column) -> callable | None | str:
        if col.name in _COLS_TO_IGNORE:
            return None
        fm = _TYPE_MAP.get(col.type.value.lower()) or _NAME_MAP.get(col.name.lower())
        if fm:
            faker_func = getattr(self._faker, fm)
            if not col.is_array:
                return faker_func
            return get_array_faker_func(faker_func, self._faker)
        return f"{col.name} [{col.type.value.lower()}]"


def get_client() -> Client:
    # no ssl in dev, since we don't use self signed certs w/ python
    disable_ssl_verify = _read_env("USERCLOUDS_TENANT_URL", "Tenant URL").endswith(
        ":3333"
    )
    factory = (
        create_no_ssl_http_client
        if disable_ssl_verify
        else create_default_uc_http_client
    )
    return Client.from_env(client_factory=factory)


def get_args() -> None:
    parser = argparse.ArgumentParser(description="Add data to UserCloud")
    parser.add_argument(
        "num_of_users",
        type=int,
        default=1,
        help="Number of users to add",
    )
    parser.add_argument(
        "purposes_args",
        metavar="purposes",
        nargs="*",
        type=str,
        help="Purposes (consents) to add to the users",
    )
    args = parser.parse_args()
    return args.num_of_users, args.purposes_args


def add_fake_data() -> int:
    num_of_users, purposes_names = get_args()
    client = get_client()
    purposes_names = purposes_names or [p.name for p in client.ListPurposes()]
    print(f"Adding {num_of_users} users with purposes: {purposes_names}")
    cols = client.ListColumns()
    ch = ColumnHelper(cols)
    purposes = [ResourceID(name=pn) for pn in purposes_names]
    for i in range(num_of_users):
        user_row = ch.get_user()
        start = time.time()
        uc_user_row = {
            k: {"value": v, "purpose_additions": purposes} for k, v in user_row.items()
        }
        try:
            user_id = client.CreateUserWithMutator(
                mutator_id=UpdateUserMutatorID, context={}, row_data=uc_user_row
            )
        except UserCloudsSDKError as err:
            print(f"Error creating user: {err!r}")
            print(f"response headers: {err.headers}")
            raise
        print(
            f"Created user took {time.time() - start:.3}sec [{i+1:,}/{num_of_users:,}] {user_id} - {user_row}"
        )
    return 0


if __name__ == "__main__":
    sys.exit(add_fake_data())
