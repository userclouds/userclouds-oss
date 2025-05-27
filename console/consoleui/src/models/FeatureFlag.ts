import { getEnvData } from './EnvData';

// "feature gate" is the term StatSig uses,
// and what appears in payloads from their API
type FeatureGate = {
  name: string;
  value: boolean;
  rule_id: string;
  group_name: string;
  id_type: string;
};
export type FeatureFlag = FeatureGate;

type FeatureGates = Record<string, FeatureGate>;
export type FeatureFlags = FeatureGates;

export const featureFlagsAreEnabled = () => {
  const { StatsSigAPIKey } = getEnvData();

  return StatsSigAPIKey !== '';
};
