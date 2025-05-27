from userclouds import tokenize

from pandas import Series

def model(dbt, session):
    data = dbt.ref('raw_users')

    # normalize email to lowercase
    data = data.apply(lambda x: Series([x[0], x[1].lower()], index=["id", "email"]), axis=1)

    return tokenize(dbt.config.get("userclouds"), data, "email")
