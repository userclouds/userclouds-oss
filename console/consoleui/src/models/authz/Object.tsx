import { v4 as uuidv4 } from 'uuid';

type UCObject = {
  id: string;
  type_id: string;
  alias: string;
};

export const blankObject = () => ({
  id: uuidv4(),
  type_id: '',
  alias: '',
});

export default UCObject;

export const objectColumns = ['id', 'alias', 'type_id', 'created', 'updated'];
