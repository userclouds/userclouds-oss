// TODO: UC prefix
type UCObject = {
  id: string;

  type_id: string;

  alias: string;

  organization_id: string;
};

type UCObjectType = {
  id: string;
  type_name: string;
  organization_id: string;
};

export default UCObject;
export type { UCObjectType };
