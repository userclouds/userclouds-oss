type PaginatedResult<Type> = {
  data: Type[];
  has_next: boolean;
  next: string;
  has_prev: boolean;
  prev: string;
};

export default PaginatedResult;
