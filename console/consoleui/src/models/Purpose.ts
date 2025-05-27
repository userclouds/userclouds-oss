import { v4 as uuidv4 } from 'uuid';

type Purpose = {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
};

export const blankPurpose = () => ({
  id: uuidv4(),
  name: '',
  description: '',
  is_system: false,
});

export const PURPOSE_PREFIX = 'purposes_';
export const PURPOSE_COLUMNS = ['id', 'name', 'created', 'updated'];

export default Purpose;
