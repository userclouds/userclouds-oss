from userclouds import resolve

def model(dbt, session):
    data = dbt.ref('ltv')

    return resolve(dbt.config.get("userclouds"), data, "email")
