import { blankResourceID, ResourceID } from './ResourceID';

export type ObjectStore = {
  id: string;
  name: string;
  type: string;
  region: string;
  access_key_id: string;
  secret_access_key: string;
  role_arn: string;
  access_policy: ResourceID;
};

export const OBJECT_STORE_PREFIX = 'objectstore_';

export const blankObjectStore = () => ({
  id: '',
  name: '',
  type: '',
  region: '',
  access_key_id: '',
  secret_access_key: '',
  role_arn: '',
  access_policy: blankResourceID(),
});
