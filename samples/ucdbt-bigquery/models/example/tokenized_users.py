from usercloudssdk.client import Client
from usercloudssdk.models import (
    ResourceID,
)


def model(dbt, session):
    dbt.config(submission_method="cluster", dataproc_cluster_name="dbt-python")
    data = dbt.ref("stg_users")

    client = Client(
        url=dbt.config.get("tenant_url"),
        client_id=dbt.config.get("client_id"),
        client_secret=dbt.config.get("client_secret"),
    )

    data_collect = data.collect()
    for row in data_collect:
        token = client.CreateToken(
            row["email"],
            ResourceID(id=dbt.config.get("email_transformer_id")),
            ResourceID(id=dbt.config.get("allow_all_access_policy_id")),
        )
        data = data.replace(row["email"], token)

    return data
