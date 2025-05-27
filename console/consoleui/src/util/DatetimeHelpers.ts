export const timestampToDatetimeLocal = (timestamp: string) => {
  try {
    const d = new Date(timestamp);
    return new Date(d.getTime() - d.getTimezoneOffset() * 60000)
      .toISOString()
      .slice(0, -1);
  } catch {
    return '';
  }
};
export const datetimeLocalToTimestamp = (datetime: string) => {
  const d = new Date(datetime + 'Z');
  return new Date(d.getTime() + d.getTimezoneOffset() * 60000).toISOString();
};
