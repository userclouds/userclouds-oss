import json
import os
from urllib import parse, request

def tokenize(dbt_config, data, col): 
    token = _get_token()

    c = dbt_config.get(col)

    url = os.environ.get('UC_TENANT_URL')

    for i, row in data.iterrows():
        d = {  
            "data": str(row.loc[col]),
            "transformer_rid": {"id": c.get("transformer_id")}, 
            "access_policy_rid": {"id": c.get("access_policy_id")},
        }
        s = json.dumps(d)

        r = request.Request(url + '/tokenizer/tokens',
                            method='POST',
                            data=s.encode('ascii'),
                            headers={'Authorization': f'Bearer {token}'}
                            )
        
        jdata = request.urlopen(r).read()
        t = json.loads(jdata).get('data')
        
        data.loc[i, col] = t

    return data

def resolve(dbt_config, data, col): 
    token = _get_token()

    c = dbt_config.get(col)

    url = os.environ.get('UC_TENANT_URL')

    for i, row in data.iterrows():
        d = {  
            "tokens": [str(row.loc[col])],
            "context": c.get('context'),
            "purposes": c.get('purposes'),
        }
        s = json.dumps(d)

        r = request.Request(url + '/tokenizer/tokens/actions/resolve',
                            method='POST',
                            data=s.encode('ascii'),
                            headers={'Authorization': f'Bearer {token}'}
                            )
        
        jdata = request.urlopen(r).read()
        t = json.loads(jdata)[0].get('data')
        
        data.loc[i, col] = t

    return data

def _get_token() -> str:
    cid = os.environ.get('UC_CLIENT_ID')
    csecret = os.environ.get('UC_CLIENT_SECRET')
    url = os.environ.get('UC_TENANT_URL')

    d = {
        "client_id": cid,
        "client_secret": csecret,
        "grant_type": "client_credentials",
        }
    
    r = request.Request(url + '/oidc/token', 
                        method='POST', 
                        data=parse.urlencode(d).encode('ascii'))

    jtoken = request.urlopen(r).read()
    token = json.loads(jtoken)['access_token']

    return token