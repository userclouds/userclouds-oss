export type OIDCProvider = {
  type: string;
  name: string;
  description: string;
  issuer_url: string;
  client_id: string;
  client_secret: string;
  can_use_local_host_redirect: boolean;
  use_local_host_redirect: boolean;
  default_scopes: string;
  additional_scopes: string;
  is_native: boolean;
};

export enum OIDCProviderType {
  Facebook = 'facebook',
  Google = 'google',
  LinkedIn = 'linkedin',
  Custom = 'custom',
  Microsoft = 'microsoft',
}

export const getBlankCustomOIDCProvider = () => {
  return {
    type: 'custom',
    name: '',
    description: '',
    issuer_url: '',
    client_id: '',
    client_secret: '',
    can_use_local_host_redirect: false,
    use_local_host_redirect: false,
    default_scopes: 'openid profile email',
    additional_scopes: '',
    is_native: false,
  } as OIDCProvider;
};

export const getOIDCProviderScopeDescription = (p: OIDCProvider) => {
  return 'Additional scopes("' + p.default_scopes + '" always included)';
};

export const getOIDCProviderNameString = (p: OIDCProvider, suffix: string) => {
  return p.name + suffix;
};

export const getOIDCProviderDescriptionString = (
  p: OIDCProvider,
  suffix: string
) => {
  return p.description + suffix;
};

export const isOIDCProviderConfigured = (p: OIDCProvider) => {
  return (
    p.type !== '' &&
    p.name !== '' &&
    p.description !== '' &&
    p.issuer_url !== '' &&
    p.client_id !== '' &&
    p.client_secret !== '' &&
    p.default_scopes !== ''
  );
};
