import { JSONValue } from '@userclouds/sharedui';
import ServerContext from './ServerContext';

type AccessPolicyContext = {
  server: ServerContext;
  client: Record<string, JSONValue>;
  user?: Record<string, JSONValue>;
  query?: Record<string, string>;
  row_data?: Record<string, string>;
};

const validKeys = ['server', 'client', 'user', 'query', 'row_data'];

export const validateAccessPolicyContext = (parsedJSON: JSONValue) => {
  const keys = Object.getOwnPropertyNames(parsedJSON);

  let err: string = '';
  if (keys.length > validKeys.length) {
    err =
      'Context object contains too many properties. Permitted properties are ' +
      validKeys.join(', ') +
      '.';
  }

  for (const key of keys) {
    if (!validKeys.includes(key)) {
      err = `Invalid key: ${key}`;
    }
  }

  // we can validate ServerContext, but it's probably not worth making them type out ip_address, claims, etc, if their policy doesn't need it
  return err;
};

export default AccessPolicyContext;
