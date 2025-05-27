import Auth0Provider from './Auth0Provider';
import CognitoProvider from './CognitoProvider';
import UCProvider from './UCProvider';

enum ProviderType {
  auth0 = 'auth0',
  uc = 'uc',
  cognito = 'cognito',
}

export type ProviderApp = {
  id: string;
  name: string;
};

type Provider = {
  id: string; // TODO: UUID
  name: string;
  type: ProviderType;
  auth0?: Auth0Provider;
  uc?: UCProvider;
  cognito?: CognitoProvider;
};

export default Provider;

export { ProviderType };
