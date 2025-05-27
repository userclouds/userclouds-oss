export type DataAccessLogFilter = {
  column_id: string;
  accessor_id: string;
  actor_id: string;
  selector_value: string;
};

export const blankDataAccessLogFilter = {
  column_id: '',
  accessor_id: '',
  actor_id: '',
  selector_value: '',
};
