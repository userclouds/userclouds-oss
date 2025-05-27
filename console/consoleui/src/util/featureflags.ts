import { FeatureFlags, FeatureFlag } from '../models/FeatureFlag';

const fastHash = (value: string): number => {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    const character = value.charCodeAt(i);
    hash = (hash << 5) - hash + character;
    hash &= hash; // Convert to 32bit integer
  }
  return hash;
};

export const hashFlagName = (flagName: string): string =>
  String(fastHash(flagName) >>> 0);

export const featureIsEnabled = (
  flagName: string,
  flags: FeatureFlags | undefined
) => {
  const flagHash = hashFlagName(flagName);

  let enabled = false;
  if (flags) {
    const flag: FeatureFlag | undefined = flags[flagHash];
    if (flag) {
      enabled = flag.value;
    }
  }
  return enabled;
};
