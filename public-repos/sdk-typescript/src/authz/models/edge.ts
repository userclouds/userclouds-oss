type Edge = {
  id: string;
  edge_type_id: string;
  source_object_id: string;
  target_object_id: string;
};

type EdgeType = {
  id: string;
  type_name: string;
  source_object_type_id: string;
  target_object_type_id: string;
  attributes: Attribute[];
};

type Attribute = {
  name: string;
  direct: boolean;
  inherit: boolean;
  propagate: boolean;
};

export default Edge;
export type { Attribute, EdgeType };
