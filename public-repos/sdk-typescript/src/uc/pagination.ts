import JSONValue from './json';

const paginationStart = '';
const defaultLimit = 100;

type PaginatedResult = {
  data: JSONValue[];
  has_next: boolean;
  next: string;
  has_prev: boolean;
  prev: string;
};

export default PaginatedResult;
export { defaultLimit, paginationStart };
