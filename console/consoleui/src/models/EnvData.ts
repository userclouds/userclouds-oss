type EnvData = {
  Universe: string;
  VersionSha: string;
  SentryDsn: string;
  StatsSigAPIKey: string;
};

declare global {
  interface Window {
    ucAppInitData: EnvData;
  }
}

const defaultEnvData = {
  Universe: 'dev',
  VersionSha: '',
  SentryDsn: '',
  StatsSigAPIKey: 'client-aVjsk13JrCU2En6fOOGeB4z5qiHVg9Fc5v0Db5RFrJX',
};

export const getEnvData = () => {
  if (typeof window === 'undefined') {
    return defaultEnvData;
  }
  let envData: EnvData = window.ucAppInitData;
  if (!envData || !Object.keys(envData).length) {
    envData = defaultEnvData;
  }
  return envData;
};
