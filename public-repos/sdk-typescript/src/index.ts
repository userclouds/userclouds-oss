import AuthZClient from './authz/client';
import UserstoreClient from './userstore/client';
import TokenizerClient from './tokenizer/client';
import PlexClient from './plex/client';
import {
  UCObject,
  UCObjectType,
  Edge,
  EdgeType,
  Organization,
  Region,
} from './authz/models';
import {
  AccessPolicy,
  AccessPolicyComponent,
  AccessPolicyTemplate,
  Accessor,
  Column,
  ColumnInputConfig,
  ColumnOutputConfig,
  Mutator,
  Normalizer,
  Purpose,
  Transformer,
  User,
  UserSelectorConfig,
  ResourceID,
} from './userstore/models';
import {
  DATA_TYPE_STRING,
  DATA_TYPE_TIMESTAMP,
  DATA_TYPE_UUID,
  DATA_TYPE_ADDRESS,
  DATA_TYPE_INTEGER,
  COLUMN_INDEX_TYPE_NONE,
  COLUMN_INDEX_TYPE_INDEXED,
  COLUMN_INDEX_TYPE_UNIQUE,
  AUTHN_TYPE_PASSWORD,
  POLICY_TYPE_COMPOSITE_AND,
  POLICY_TYPE_COMPOSITE_OR,
  TRANSFORM_TYPE_PASSTHROUGH,
  TRANSFORM_TYPE_TOKENIZE_BY_REFERENCE,
  TRANSFORM_TYPE_TOKENIZE_BY_VALUE,
  TRANSFORM_TYPE_TRANSFORM,
  MUTATOR_COLUMN_DEFAULT_VALUE,
  MUTATOR_COLUMN_CURRENT_VALUE,
} from './userstore/constants';

const getClientCredentialsToken = async (
  tenantUrl: string,
  clientId: string,
  clientSecret: string
): Promise<string> => {
  const body = new URLSearchParams({
    grant_type: 'client_credentials',
    client_id: clientId,
    client_secret: clientSecret,
    audience: tenantUrl,
  });

  return fetch(`${tenantUrl}/oidc/token`, {
    method: 'POST',
    headers: { 'content-type': 'application/x-www-form-urlencoded' },
    body,
  })
    .then((res) => res.json())
    .then((json) => json.access_token);
};

export {
  AccessPolicyComponent,
  AuthZClient,
  UserstoreClient,
  TokenizerClient,
  PlexClient,
  getClientCredentialsToken,
  Region,
  ResourceID,
  DATA_TYPE_STRING,
  DATA_TYPE_TIMESTAMP,
  DATA_TYPE_UUID,
  DATA_TYPE_ADDRESS,
  DATA_TYPE_INTEGER,
  COLUMN_INDEX_TYPE_NONE,
  COLUMN_INDEX_TYPE_INDEXED,
  COLUMN_INDEX_TYPE_UNIQUE,
  AUTHN_TYPE_PASSWORD,
  POLICY_TYPE_COMPOSITE_AND,
  POLICY_TYPE_COMPOSITE_OR,
  TRANSFORM_TYPE_PASSTHROUGH,
  TRANSFORM_TYPE_TOKENIZE_BY_REFERENCE,
  TRANSFORM_TYPE_TOKENIZE_BY_VALUE,
  TRANSFORM_TYPE_TRANSFORM,
  MUTATOR_COLUMN_DEFAULT_VALUE,
  MUTATOR_COLUMN_CURRENT_VALUE,
};

export type {
  UCObject,
  UCObjectType,
  Edge,
  EdgeType,
  Organization,
  AccessPolicy,
  AccessPolicyTemplate,
  Accessor,
  Column,
  ColumnInputConfig,
  ColumnOutputConfig,
  Mutator,
  Normalizer,
  Purpose,
  Transformer,
  User,
  UserSelectorConfig,
};
