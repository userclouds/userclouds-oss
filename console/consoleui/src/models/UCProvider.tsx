import ProviderApp from './ProviderApp';

type UCProvider = {
  idp_url: string;
  apps: ProviderApp[];
};

export const blankUCProvider = () =>
  ({
    idp_url: '',
    apps: [],
  }) as UCProvider;

export default UCProvider;
