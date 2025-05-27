function id(len, decimalonly) {
  const s = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
  const d = '0123456789';
  if (decimalonly) {
    return Array(len)
      .join()
      .split(',')
      .map(() => d.charAt(Math.floor(Math.random() * d.length)))
      .join('');
  }
  return Array(len)
    .join()
    .split(',')
    .map(() => s.charAt(Math.floor(Math.random() * s.length)))
    .join('');
}

function constructSegment(seg, decimalonly, preserveS, preserveT) {
  const preserveCountS = Math.min(Math.max(preserveS, 0), seg.length);
  const preserveCountT = Math.min(Math.max(preserveT, 0), seg.length);

  const preserveCount = preserveCountS + preserveCountT;
  if (preserveCount >= seg.length) {
    return seg;
  }

  const newSegS = seg.slice(0, preserveCountS);
  const newSegT = seg.slice(seg.length - preserveCountT, seg.length);
  return newSegS + id(seg.length - preserveCount, decimalonly) + newSegT;
}

function validate(str) {
  const regexp = /^(?!000|666)[0-8][0-9]{2}(?!00)[0-9]{2}(?!0000)[0-9]{4}$/;

  return regexp.test(str);
}

function transform(data, params) {
  // Strip non numeric characters if present
  const origData = data;
  const stripped = data.replace(/\D/g, '');
  if (!validate(stripped)) {
    throw new Error('Invalid SSN Provided');
  }

  if (
    params.PreserveCharsTrailing + params.PreserveCharsStart > 9 ||
    params.PreserveCharsTrailing < 0 ||
    params.PreserveCharsStart < 0
  ) {
    throw new Error('Invalid Params Provided');
  }

  if (params.PreserveValue) {
    return origData;
  }

  const seg1 = stripped.slice(0, 3);
  const seg2 = stripped.slice(3, 5);
  const seg3 = stripped.slice(5, 9);
  return `${constructSegment(
    seg1,
    params.DecimalOnly,
    params.PreserveCharsStart,
    params.PreserveCharsTrailing - 6
  )}-${constructSegment(
    seg2,
    params.DecimalOnly,
    params.PreserveCharsStart - 3,
    params.PreserveCharsTrailing - 4
  )}-${constructSegment(
    seg3,
    params.DecimalOnly,
    params.PreserveCharsStart - 5,
    params.PreserveCharsTrailing
  )}`;
}
