export const truncatedID = (id: string) => {
  return id
    ? ' (' + (id.length > 6 ? id.substring(0, 6) + '...' : id) + ')'
    : '';
};
