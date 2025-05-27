from usercloudssdk.client import Client


def model(dbt, session):
    dbt.config(submission_method="cluster", dataproc_cluster_name="dbt-python")
    data = dbt.ref("tokenized_average")

    client = Client(
        url=dbt.config.get("tenant_url"),
        client_id=dbt.config.get("client_id"),
        client_secret=dbt.config.get("client_secret"),
    )

    data_collect = data.collect()
    for row in data_collect:
        resolved = client.ResolveTokens([row["email"]], {}, [])
        if resolved:
            data = data.replace(row["email"], resolved[0]["data"])

    return data
