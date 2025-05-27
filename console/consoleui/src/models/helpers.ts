export const VALID_NAME_PATTERN = '^[a-zA-Z_][a-zA-Z0-9_\\-]*$';
export const VALID_NAME_REGEX = new RegExp(VALID_NAME_PATTERN);

export const nonZeroNumberPattern = /^(?:[1-9][0-9]*|)$/;

export const isNonZeroNumber = (num: number) => {
  return nonZeroNumberPattern.test(String(num));
};
