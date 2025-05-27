import ProviderApp from './ProviderApp';
import Auth0Management from './Auth0Management';

type Auth0Provider = {
  domain: string;
  apps: ProviderApp[];
  management: Auth0Management;
  redirect: boolean;
};

export const blankAuth0Provider = () =>
  ({
    domain: '',
    apps: [],
    management: {
      client_id: '',
      client_secret: '',
      audience: '',
    },
    redirect: false,
  }) as Auth0Provider;

export default Auth0Provider;
