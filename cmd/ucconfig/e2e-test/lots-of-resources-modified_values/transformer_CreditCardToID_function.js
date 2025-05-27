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
  // there a lot of different regexes per type of card
  // for now just test the length
  return str.length === 16;
}

function transform(data, params) {
  // Strip non numeric characters if present
  const origData = data;
  const stripped = data.replace(/\D/g, '');
  if (!validate(stripped)) {
    throw new Error('Invalid Credit Card Number Provided');
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

  const seg1 = stripped.slice(0, 4);
  const seg2 = stripped.slice(4, 8);
  const seg3 = stripped.slice(8, 12);
  const seg4 = stripped.slice(12, 16);
  return `${constructSegment(
    seg1,
    params.DecimalOnly,
    params.PreserveCharsStart,
    params.PreserveCharsTrailing - 12
  )}-${constructSegment(
    seg2,
    params.DecimalOnly,
    params.PreserveCharsStart - 4,
    params.PreserveCharsTrailing - 8
  )}-${constructSegment(
    seg3,
    params.DecimalOnly,
    params.PreserveCharsStart - 8,
    params.PreserveCharsTrailing - 4
  )}-${constructSegment(
    seg4,
    params.DecimalOnly,
    params.PreserveCharsStart - 12,
    params.PreserveCharsTrailing
  )}`;
}
