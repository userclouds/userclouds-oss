function id(len) {
  const s = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  return Array(len)
    .join()
    .split(',')
    .map(() => s.charAt(Math.floor(Math.random() * s.length)))
    .join('');
}

function constructSegment(seg, config) {
  if (config.PreserveValue) {
    return seg;
  }
  const preserveCount = Math.min(config.PreserveChars, seg.length);
  const newSeg = seg.slice(0, preserveCount);
  return newSeg + id(config.FinalLength - preserveCount);
}

function transform(data, params) {
  const nameParts = data.split(' ');

  // Assume that if we have a single name, treat it as a first name
  let firstName = data;
  let lastName = '';
  if (nameParts.length > 0) {
    firstName = nameParts[0];
  }

  // Skip middle name if provided
  if (nameParts.length > 1) {
    lastName = nameParts[nameParts.length - 1];
  }

  if (params.length !== 2) {
    throw new Error('Invalid Params');
  }

  return `${constructSegment(firstName, params[0])} ${constructSegment(
    lastName,
    params[1]
  )}`;
}
