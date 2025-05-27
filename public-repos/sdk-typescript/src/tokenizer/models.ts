import { AccessPolicy, Transformer } from '../userstore/models';

type InspectTokenResponse = {
  token: string;
  id: string;
  transformer: Transformer;
  access_policy: AccessPolicy;
};

export default InspectTokenResponse;
