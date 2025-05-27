import AWSConfig from './AWSConfig';
import ProviderApp from './ProviderApp';

type CognitoProvider = {
  aws_config: AWSConfig;
  user_pool_id: string;

  apps: ProviderApp[];
};

export const blankCognitoProvider = () =>
  ({
    aws_config: {
      access_key: '',
      secret_key: '',
      region: '',
    },
    user_pool_id: '',
    apps: [],
  }) as CognitoProvider;

export default CognitoProvider;
