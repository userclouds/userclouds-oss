import UserSelectorConfig from './user_selector';
import ResourceID from './resource_id';

type ColumnInputConfig = {
  column: ResourceID;
  normalizer: ResourceID;
};

type Mutator = {
  id: string;
  name: string;
  description: string;
  columns: ColumnInputConfig[];
  access_policy: ResourceID;
  selector_config: UserSelectorConfig;
  version: number;
};

export default Mutator;
export type { ColumnInputConfig };
