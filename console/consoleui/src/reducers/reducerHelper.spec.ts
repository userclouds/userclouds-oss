import { getNewToggleEditValue, deepEqual } from './reducerHelper';

describe('getNewToggleEditValue', () => {
  it('should toggle when action data is null or undefined', () => {
    expect(getNewToggleEditValue(null, false)).toBe(true);
    expect(getNewToggleEditValue(null, true)).toBe(false);
    expect(getNewToggleEditValue(undefined, false)).toBe(true);
    expect(getNewToggleEditValue(undefined, true)).toBe(false);
  });

  it('should toggle when action data is not a boolean', () => {
    expect(getNewToggleEditValue(1234, false)).toBe(true);
    expect(getNewToggleEditValue(1234, true)).toBe(false);
    expect(getNewToggleEditValue('1234', false)).toBe(true);
    expect(getNewToggleEditValue('1234', true)).toBe(false);
    expect(getNewToggleEditValue({ a: '1234' }, false)).toBe(true);
    expect(getNewToggleEditValue({ a: '1234' }, true)).toBe(false);
  });

  it('should accept action data when it is a boolean regardless of edit value', () => {
    expect(getNewToggleEditValue(true, false)).toBe(true);
    expect(getNewToggleEditValue(true, true)).toBe(true);
    expect(getNewToggleEditValue(false, false)).toBe(false);
    expect(getNewToggleEditValue(false, true)).toBe(false);
  });
});

describe('isEqualDeep', () => {
  const o1 = { a: 'a' };
  const o2 = { a: 'a' };
  const o3 = { a: 'a', b: { a: 'a', b: 'b' } };
  const o4 = { a: 'a', b: { a: 'a', b: 'b' } };
  const o5 = { a: 'a', b: { b: 'b', a: 'a' } };

  it('should check basic equivalency', () => {
    expect(deepEqual(o1, o2)).toBe(true);
  });
  it('should check deep equivalency', () => {
    expect(deepEqual(o3, o4)).toBe(true);
  });
  it('should check deep equivalency regardless of order', () => {
    expect(deepEqual(o3, o5)).toBe(true);
  });
  it('should handle null and undefined values', () => {
    expect(deepEqual(undefined, undefined)).toBe(true);
    expect(deepEqual(null, null)).toBe(true);
    expect(deepEqual(null, undefined)).toBe(false);
  });
  it('should check basic unequivalency', () => {
    expect(deepEqual(o1, o3)).toBe(false);
    expect(deepEqual(o1, o5)).toBe(false);
  });

  it('should check unequivalency with null types', () => {
    expect(deepEqual(o1, null)).toBe(false);
    expect(deepEqual(o1, undefined)).toBe(false);
  });
});
