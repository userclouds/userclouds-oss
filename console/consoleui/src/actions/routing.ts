export const NAVIGATE = 'NAVIGATE';
export const navigate = (
  location: URL,
  handler: Function,
  pattern: string,
  params: Record<string, string>
) => ({
  type: NAVIGATE,
  data: {
    location,
    handler,
    pattern,
    params,
  },
});
