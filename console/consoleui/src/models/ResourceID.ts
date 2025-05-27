import { NilUuid } from './Uuids';

export type ResourceID = {
  id: string;
  name: string;
};

export const blankResourceID = () => ({
  id: NilUuid,
  name: '',
});
