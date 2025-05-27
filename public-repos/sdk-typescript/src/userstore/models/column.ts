type Column = {
  id: string;
  table: string;
  name: string;
  type: string;
  is_array: boolean;
  default_value: string;
  index_type: string;
};

export default Column;
