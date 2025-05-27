export const truncateWithEllipsis = (
  str: string,
  maxLength: number
): string => {
  return str.length > maxLength ? str.substring(0, maxLength - 3) + '...' : str;
};
