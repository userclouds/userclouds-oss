import { JSONValue } from '@userclouds/sharedui';
import { validateAccessPolicyContext } from './AccessPolicyContext';

describe('AccessPolicyContext', () => {
  describe('validateAccessPolicyContext', () => {
    let testData: JSONValue = {};
    it('should return an error if the object has invalid keys', () => {
      testData = {
        client: {},
        server: {},
        user: {},
        foo: {},
      };
      expect(validateAccessPolicyContext(testData)).toBe('Invalid key: foo');

      testData = {
        client: {},
        bar: {},
        user: {},
        foo: {},
      };
      expect(validateAccessPolicyContext(testData)).toBe('Invalid key: foo');
    });

    it('should return an error if the object has three keys, but not the right ones', () => {
      testData = {
        server: {},
        client: {},
        foo: {},
      };
      expect(validateAccessPolicyContext(testData)).toBe('Invalid key: foo');

      testData = {
        bar: {},
        baz: {},
        foo: {},
      };
      expect(validateAccessPolicyContext(testData)).toBe('Invalid key: foo');
    });

    it('should return an error if the object has two keys, but not the right ones', () => {
      testData = {
        client: {},
        sarver: {},
      };
      expect(validateAccessPolicyContext(testData)).toBe('Invalid key: sarver');
    });

    it('should not validate server context', () => {
      testData = {
        client: {},
        server: {
          ip_addy: '1.2.4.3',
          claims: {},
        },
      };
      expect(validateAccessPolicyContext(testData)).toBe('');

      testData = {
        client: {},
        server: Infinity,
      };
      expect(validateAccessPolicyContext(testData)).toBe('');

      testData = {
        client: {
          foo: 'bar',
        },
        server: {
          ip_address: '127.0.0.1',
          claims: {
            sub: 'bob',
          },
          action: 'resolve',
        },
      };
      expect(validateAccessPolicyContext(testData)).toBe('');
    });

    it('should not return an error if the user property is not present', () => {
      testData = {
        client: {
          foo: 'bar',
        },
        server: {
          ip_address: '127.0.0.1',
          claims: {
            sub: 'bob',
          },
          action: 'resolve',
        },
      };
      expect(validateAccessPolicyContext(testData)).toBe('');
    });

    it('should not return an error for a well-formed context object', () => {
      testData = {
        client: {
          foo: 'bar',
        },
        server: {
          ip_address: '127.0.0.1',
          claims: {
            sub: 'bob',
          },
          action: 'resolve',
        },
        user: {
          email: 'foo@bar.com',
        },
      };
      expect(validateAccessPolicyContext(testData)).toBe('');
    });
  });
});
