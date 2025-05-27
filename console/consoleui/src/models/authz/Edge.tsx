import { v4 as uuidv4 } from 'uuid';

type Edge = {
  id: string;
  edge_type_id: string;
  source_object_id: string;
  target_object_id: string;
};

export const blankEdge = () => {
  return {
    id: uuidv4(),
    edge_type_id: '',
    source_object_id: '',
    target_object_id: '',
  };
};

export const edgeColumns = [
  'id',
  'source_object_id',
  'target_object_id',
  'created',
  'updated',
];

export const edgesPrefix = 'edges_';

export default Edge;
