import { v4 as uuidv4 } from 'uuid';

export type PolicySecret = {
  id: string;
  name: string;
  value: string;
  created: number;
};

export const blankPolicySecret = (): PolicySecret => ({
  id: uuidv4(),
  name: '',
  value: '',
  created: 0,
});

export const POLICY_SECRET_PREFIX = 'secret_';
