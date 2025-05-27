import { ParsedExecuteAccessorData } from '../models/Accessor';
import { tallyFrequencies, tallySensitiveDataUniqueness } from './DataAnalysis';

describe('tallyFrequencies', () => {
  const record: ParsedExecuteAccessorData = [
    { a: 'foo', b: 'bar', c: 'baz' },
    { a: 'baz', b: 'bar', c: 'foo' },
    { a: 'bar', b: 'foo', c: 'baz' },
    { a: 'bar', b: 'baz', c: 'bar' },
    { a: 'foo', b: 'bar', c: 'baz' },
    { a: 'bar', b: 'baz', c: 'foo' },
    { a: 'foo', b: 'bar', c: 'baz' },
  ];

  it('should not have entries that appear multiple times', () => {
    expect(tallyFrequencies(record, ['a', 'b', 'c'])).toEqual({
      'foo / bar / baz': 3,
      'baz / bar / foo': 1,
      'bar / foo / baz': 1,
      'bar / baz / bar': 1,
      'bar / baz / foo': 1,
    });
  });

  it('should not count fields not marked as PII', () => {
    expect(tallyFrequencies(record, ['a', 'b'])).toEqual({
      'foo / bar': 3,
      'baz / bar': 1,
      'bar / foo': 1,
      'bar / baz': 2,
    });
  });

  it('should not use delimiters if only one PII field is passed', () => {
    expect(tallyFrequencies(record, ['a'])).toEqual({
      foo: 3,
      baz: 1,
      bar: 3,
    });
  });

  it("should ignore any PII fields that aren't present in the data set", () => {
    expect(tallyFrequencies(record, ['a', 'd'])).toEqual({
      foo: 3,
      baz: 1,
      bar: 3,
    });
  });

  it('should return an empty object if passed an empty data set', () => {
    expect(tallyFrequencies([], ['a', 'b'])).toEqual({});
  });

  it('should return an empty object if passed an empty PII fields array', () => {
    expect(tallyFrequencies(record, [])).toEqual({});
  });

  it("should return an empty object if PII fields don't match any data set fields", () => {
    expect(tallyFrequencies(record, ['d'])).toEqual({});
  });
});

describe('tallySensitiveDataUniqueness', () => {
  const record: ParsedExecuteAccessorData = [
    { a: 'foo', b: 'bar', c: 'baz', d: 'bip' },
    { a: 'baz', b: 'bar', c: 'foo', d: 'boop' },
    { a: 'bar', b: 'foo', c: 'baz', d: 'bip' },
    { a: 'bar', b: 'baz', c: 'bar', d: 'bip' },
    { a: 'foo', b: 'bar', c: 'baz', d: 'bip' },
    { a: 'bar', b: 'baz', c: 'foo', d: 'boop' },
    { a: 'foo', b: 'bar', c: 'baz', d: 'bip' },
    { a: 'foo', b: 'bar', c: 'bar', d: 'boop' },
    { a: 'bar', b: 'baz', c: 'foo', d: 'bip' },
  ];

  it('should create buckets consisting of all values from fields PII fields not labeled sensitive', () => {
    const result = tallySensitiveDataUniqueness(record, ['c'], ['a', 'b']);

    expect(Object.keys(result)).toEqual(['c']);
    expect(Object.keys(result.c)).toEqual([
      'foo / bar',
      'baz / bar',
      'bar / foo',
      'bar / baz',
    ]);
  });

  it('should separate reuslts for each sensitive field', () => {
    const result = tallySensitiveDataUniqueness(record, ['c', 'd'], ['a', 'b']);
    expect(Object.keys(result)).toEqual(['c', 'd']);
    expect(Object.keys(result.c)).toEqual([
      'foo / bar',
      'baz / bar',
      'bar / foo',
      'bar / baz',
    ]);
    expect(Object.keys(result.d)).toEqual([
      'foo / bar',
      'baz / bar',
      'bar / foo',
      'bar / baz',
    ]);
  });

  it('it should tally unique values for each of the sensitive fields', () => {
    expect(tallySensitiveDataUniqueness(record, ['c'], ['a', 'b'])).toEqual({
      c: {
        'foo / bar': {
          baz: 3,
          bar: 1,
        },
        'baz / bar': {
          foo: 1,
        },
        'bar / foo': {
          baz: 1,
        },
        'bar / baz': {
          bar: 1,
          foo: 2,
        },
      },
    });

    expect(
      tallySensitiveDataUniqueness(record, ['c', 'd'], ['a', 'b'])
    ).toEqual({
      c: {
        'foo / bar': {
          baz: 3,
          bar: 1,
        },
        'baz / bar': {
          foo: 1,
        },
        'bar / foo': {
          baz: 1,
        },
        'bar / baz': {
          bar: 1,
          foo: 2,
        },
      },
      d: {
        'foo / bar': {
          bip: 3,
          boop: 1,
        },
        'baz / bar': {
          boop: 1,
        },
        'bar / foo': {
          bip: 1,
        },
        'bar / baz': {
          bip: 2,
          boop: 1,
        },
      },
    });
  });

  it('should not use non-PII fields to construct buckets', () => {
    const result = tallySensitiveDataUniqueness(record, ['c'], ['a', 'd']);
    expect(Object.keys(result.c)).toEqual([
      'foo / bip',
      'baz / boop',
      'bar / bip',
      'bar / boop',
      'foo / boop',
    ]);
  });

  it('should not use have any repeated buckets for a given sensitive field', () => {
    const result = tallySensitiveDataUniqueness(record, ['c'], ['a', 'd']);
    expect(Object.keys(result.c)).toEqual([
      'foo / bip',
      'baz / boop',
      'bar / bip',
      'bar / boop',
      'foo / boop',
    ]);
  });

  it('should return an empty object if no sensitive fields are passed in', () => {
    expect(
      tallySensitiveDataUniqueness(record, [], ['a', 'b', 'c', 'd'])
    ).toEqual({});
    expect(tallySensitiveDataUniqueness(record, [], ['b', 'c'])).toEqual({});
    expect(tallySensitiveDataUniqueness(record, [], ['c'])).toEqual({});
    expect(tallySensitiveDataUniqueness(record, [], [])).toEqual({});
  });

  it('should create a single bucket for all results if no PII fields are passed in', () => {
    expect(tallySensitiveDataUniqueness(record, ['c', 'd'], [])).toEqual({
      c: {
        '[all results]': {
          baz: 4,
          foo: 3,
          bar: 2,
        },
      },
      d: {
        '[all results]': {
          bip: 6,
          boop: 3,
        },
      },
    });
  });
});
