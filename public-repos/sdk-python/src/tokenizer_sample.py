from __future__ import annotations

import functools
import os

from usercloudssdk.client import Client
from usercloudssdk.constants import DataType, PolicyType, TransformType
from usercloudssdk.data_types import ColumnDataTypeString
from usercloudssdk.errors import UserCloudsSDKError
from usercloudssdk.models import (
    AccessPolicy,
    AccessPolicyComponent,
    AccessPolicyTemplate,
    ResourceID,
    Transformer,
)
from usercloudssdk.policies import AccessPolicyOpen, TransformerUUID
from usercloudssdk.uchttpclient import (
    create_default_uc_http_client,
    create_no_ssl_http_client,
)

client_id = "<REPLACE ME>"
client_secret = "<REPLACE ME>"
url = "<REPLACE ME>"


def test_access_policies(client: Client):
    new_apt = AccessPolicyTemplate(
        name="test_template",
        function=f"function policy(x, y) {{ return false /* {id} */}};",
    )

    try:
        created_apt = client.CreateAccessPolicyTemplate(new_apt, if_not_exists=True)
        # no op, but illustrates how to get a policy template
        created_apt = client.GetAccessPolicyTemplate(ResourceID(id=created_apt.id))
        created_apt.description = "updated description"
        client.UpdateAccessPolicyTemplate(created_apt)
    except UserCloudsSDKError as e:
        print("failed to create new access policy template: ", e)
        raise

    new_ap = AccessPolicy(
        name="test_access_policy",
        policy_type=PolicyType.COMPOSITE_AND,
        components=[
            AccessPolicyComponent(
                template=ResourceID(name="test_template"), template_parameters="{}"
            )
        ],
    )

    try:
        created_ap = client.CreateAccessPolicy(new_ap, if_not_exists=True)
        # no op, but illustrates how to get a policy
        created_ap = client.GetAccessPolicy(ResourceID(id=created_ap.id))
    except UserCloudsSDKError as e:
        print("failed to create new access policy: ", e)
        raise

    aps = []
    try:
        aps = client.ListAccessPolicies()
    except UserCloudsSDKError as e:
        print("failed to list access policies: ", e)
        raise

    if not functools.reduce(
        lambda found, ap: found or (ap.id == AccessPolicyOpen.id), aps
    ):
        print("missing AccessPolicyOpen in list: ", aps)
    if not functools.reduce(lambda found, ap: found or (ap.id == created_ap.id), aps):
        print("missing new access policy in list: ", aps)

    created_ap.components[0].template_parameters = '{"foo": "bar"}'

    try:
        update = client.UpdateAccessPolicy(created_ap)
        if update.version != created_ap.version + 1:
            print(
                f"update changed version from {created_ap.version} to {update.version},\
 expected +1"
            )
    except UserCloudsSDKError as e:
        print("failed to update access policy: ", e)
        raise

    try:
        if not client.DeleteAccessPolicy(update.id, update.version):
            print("failed to delete access policy but no error?")
    except UserCloudsSDKError as e:
        print("failed to delete access policy: ", e)
        raise

    try:
        aps = client.ListAccessPolicies()
        for ap in aps:
            if ap.id == update.id:
                if ap.version != 0:
                    print(f"got access policy with version {ap.version}, expected 0")
        if len(aps) == 0:
            print("found no policies, expected to find version 0")
    except UserCloudsSDKError as e:
        print("failed to get access policy: ", e)
        raise

    # clean up the original AP and Template so you can re-run the sample repeatedly
    # without an error
    try:
        if not client.DeleteAccessPolicy(update.id, 0):
            print("failed to delete access policy but no error?")
    except UserCloudsSDKError as e:
        print("failed to delete access policy: ", e)
        raise

    try:
        if not client.DeleteAccessPolicyTemplate(
            created_apt.id, 1
        ) or not client.DeleteAccessPolicyTemplate(created_apt.id, 0):
            print("failed to delete access policy template but no error?")
    except UserCloudsSDKError as e:
        print("failed to delete access policy template: ", e)
        raise


def test_transformers(client: Client):
    new_gp = Transformer(
        name="test_transformer",
        input_data_type=ColumnDataTypeString,
        input_type=DataType.STRING,
        output_data_type=ColumnDataTypeString,
        output_type=DataType.STRING,
        reuse_existing_token=False,
        transform_type=TransformType.PASSTHROUGH,
        function="function transform(x, y) { return 'token' };",
        parameters="{}",
    )

    try:
        created_gp = client.CreateTransformer(new_gp, if_not_exists=True)
    except UserCloudsSDKError as e:
        print("failed to create new transformer: ", e)
        raise

    gps = []
    try:
        gps = client.ListTransformers()
    except UserCloudsSDKError as e:
        print("failed to list transformers: ", e)
        raise

    if not functools.reduce(
        lambda found, gp: found or (gp.id == TransformerUUID.id), gps
    ):
        print("missing TransformerUUID in list: ", gps)
    if not functools.reduce(lambda found, gp: found or (gp.id == created_gp.id), gps):
        print("missing new transformer in list: ", gps)

    try:
        if not client.DeleteTransformer(created_gp.id):
            print("failed to delete transformer but no error?")
    except UserCloudsSDKError as e:
        print("failed to delete transformer: ", e)
        raise


def test_token_apis(client: Client) -> None:
    originalData = "something very secret"
    token = client.CreateToken(originalData, TransformerUUID, AccessPolicyOpen)
    print(f"Token: {token}")

    resp = client.ResolveTokens([token], {}, [])
    if len(resp) != 1 or resp[0]["data"] != originalData:
        print("something went wrong")
    else:
        print(f"Data: {resp[0]['data']}")

    lookup_tokens = None
    try:
        lookup_tokens = client.LookupToken(
            originalData, TransformerUUID, AccessPolicyOpen
        )
    except UserCloudsSDKError as e:
        print("failed to lookup token: ", e)
        raise

    if token not in lookup_tokens:
        print(
            f"expected lookup tokens {lookup_tokens} to contain created token {token}"
        )

    itr = None
    try:
        itr = client.InspectToken(token)
    except UserCloudsSDKError as e:
        print("failed to inspect token: ", e)
        raise

    if itr.token != token:
        print(f"expected inspect token {itr.token} to match created token {token}")
    if itr.transformer.id != TransformerUUID.id:
        print(
            f"expected inspect transformer {itr.transformer.id} to match created \
transformer {TransformerUUID.id}"
        )
    if itr.access_policy.id != AccessPolicyOpen.id:
        print(
            f"expected inspect access policy {itr.access_policy.id} to match created \
access policy {AccessPolicyOpen.id}"
        )

    try:
        if not client.DeleteToken(token):
            print("failed to delete token but no error?")
    except UserCloudsSDKError as e:
        print("failed to delete token: ", e)
        raise


def test_error_handling(client: Client) -> None:
    try:
        d = client.ResolveTokens(["not a token"], {}, [])
        if d[0]["data"] != "":
            print("expected nothing but got data: ", d)
    except UserCloudsSDKError as e:
        if e.code != 404:
            print("got unexpected error code (wanted 404): ", e.code)
            raise


def run_tokenizer_sample(client: Client) -> None:
    test_access_policies(client)
    test_transformers(client)
    test_token_apis(client)
    test_error_handling(client)


if __name__ == "__main__":
    disable_ssl_verify = (
        os.environ.get("DEV_ONLY_DISABLE_SSL_VERIFICATION", "") == "true"
    )

    client = Client(
        url=url,
        client_id=client_id,
        client_secret=client_secret,
        client_factory=(
            create_no_ssl_http_client
            if disable_ssl_verify
            else create_default_uc_http_client
        ),
        session_name=os.environ.get("UC_SESSION_NAME"),
    )
    run_tokenizer_sample(client)
