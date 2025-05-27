import { SAMLEntityDescriptor } from './SAMLEntityDescriptor';

// types to make it easier to keep them in sync and visually catch bugs/omissions.
export type SAMLIDP = {
  metadata_url: string;
  sso_url: string;

  certificate: string;
  private_key?: string;

  // NB: this is currently just a list of metadata URLs, but eventually we might need more fidelity here
  trusted_service_providers?: SAMLEntityDescriptor[] | undefined;
};
