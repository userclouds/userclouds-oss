import { v4 as uuidv4 } from 'uuid';

export type SqlshimDatabase = {
  id: string;
  name: string;
  type: string;
  host: string;
  port: number;
  username: string;
  password: string;
  proxy_port: number;
  proxy_host: string;
};

export const getDatabaseError = (
  database: SqlshimDatabase | undefined
): string => {
  if (!database) {
    return '';
  }
  if (!database.host) {
    return 'Database Host is required';
  }
  if (!database.port) {
    return 'Database Port is required';
  }
  if (!database.name) {
    return 'Database Name is required';
  }
  if (!database.username) {
    return 'Username is required';
  }
  return '';
};

export const blankSqlShimDatabase = (): SqlshimDatabase => {
  return {
    id: uuidv4(),
    name: '',
    type: 'postgres',
    host: '',
    port: 0,
    username: '',
    password: '',
    proxy_port: 0,
    proxy_host: '',
  };
};
