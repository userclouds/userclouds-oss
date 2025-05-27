function id(len) {
  const s = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';

  return Array(len)
    .join()
    .split(',')
    .map(() => s.charAt(Math.floor(Math.random() * s.length)))
    .join('');
}

const commonValues = [
  'gmail',
  'hotmail',
  'yahoo',
  'msn',
  'aol',
  'orange',
  'wanadoo',
  'comcast',
  'live',
  'apple',
  'proton',
  'yandex',
  'ymail',
];

function constructSegment(seg, config) {
  if (config.PreserveValue) {
    return seg;
  }

  if (config.PreserveCommonValue && commonValues.includes(seg)) {
    return seg;
  }

  const preserveCount = Math.min(config.PreserveChars, seg.length);
  const newSeg = seg.slice(0, preserveCount);
  return newSeg + id(config.FinalLength - preserveCount);
}

function transform(data, params) {
  const emailParts = data.split('@');

  // Make sure we have a username and a domain
  if (emailParts.length !== 2) {
    throw new Error('Invalid Data');
  }

  const username = emailParts[0];
  const domainParts = emailParts[1].split('.');

  // Check if the domain is valid
  if (domainParts.length < 2) {
    throw new Error('Invalid Data');
  }
  const domainName = domainParts[0];
  const domainExt = domainParts[1];

  if (params.length !== 3) {
    throw new Error('Invalid Params');
  }
  return `${constructSegment(username, params[0])}@${constructSegment(
    domainName,
    params[1]
  )}.${constructSegment(domainExt, params[2])}`;
}
