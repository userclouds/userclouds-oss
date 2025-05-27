from usercloudssdk.client import Client, _read_env
from usercloudssdk.uchttpclient import (
    create_default_uc_http_client,
    create_no_ssl_http_client,
)


def get_client():
    # no ssl in dev, since we don't use self signed certs w/ python
    disable_ssl_verify = _read_env("USERCLOUDS_TENANT_URL", "Tenant URL").endswith(
        ":3333"
    )
    factory = (
        create_no_ssl_http_client
        if disable_ssl_verify
        else create_default_uc_http_client
    )
    # TODO: add session_name="contacts-app-demo" with new SDK version that supports it
    return Client.from_env(client_factory=factory)
