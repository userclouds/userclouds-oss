export const DataTypes = {
  TIMESTAMP: 'timestamp',
  UUID: 'uuid',
  STRING: 'string',
};
export const Operators = {
  LIKE: 'LK',
  NOT_LIKE: 'NL',
  LESS_THAN: 'LT',
  GREATER_THAN: 'GT',
  EQUAL: 'EQ',
  NOT_EQUAL: 'NE',
  GREATER_THAN_EQUAL: 'GE',
  LESS_THAN_EQUAL: 'LE',
  HAS: 'HAS',
};

export type Filter = {
  columnName: string;
  operator: string;
  value: string;
  operator2?: string;
  value2?: string;
};
