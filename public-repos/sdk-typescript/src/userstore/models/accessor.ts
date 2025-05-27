import UserSelectorConfig from './user_selector';
import ResourceID from './resource_id';

type ColumnOutputConfig = {
  column: ResourceID;
  transformer: ResourceID;
};

type Accessor = {
  id: string;
  name: string;
  description: string;
  columns: ColumnOutputConfig[];
  access_policy: ResourceID;
  token_access_policy: ResourceID;
  selector_config: UserSelectorConfig;
  purposes: ResourceID[];
  version: number;
};

export default Accessor;
export type { ColumnOutputConfig };
