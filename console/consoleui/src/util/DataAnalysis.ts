import { ParsedExecuteAccessorData } from '../models/Accessor';

export const tallyFrequencies = (
  dataSet: ParsedExecuteAccessorData,
  piiFields: string[]
) => {
  const seen: Record<string, number> = {};
  for (let i = 0; i < dataSet.length; i++) {
    const testStr = piiFields
      .filter((field: string) => dataSet[i].hasOwnProperty(field))
      .map((field: string) => dataSet[i][field])
      .join(' / ');
    if (testStr) {
      if (seen[testStr]) {
        seen[testStr] += 1;
      } else {
        seen[testStr] = 1;
      }
    }
  }

  return seen;
};

export const tallySensitiveDataUniqueness = (
  dataSet: ParsedExecuteAccessorData,
  sensitiveFields: string[],
  piiFields: string[]
) => {
  const sensitiveSeen: Record<
    string,
    Record<string, Record<string, number>>
  > = {};
  for (const field of sensitiveFields) {
    sensitiveSeen[field] = {};
    const remainingFields = piiFields.filter((f: string) => field !== f);
    for (let i = 0; i < dataSet.length; i++) {
      const sensitiveVal = dataSet[i][field];
      let testStr: string = remainingFields
        .map((f: string) => dataSet[i][f])
        .join(' / ');
      if (!testStr) {
        testStr = '[all results]';
      }
      sensitiveSeen[field][testStr] = sensitiveSeen[field][testStr] || {};
      if (sensitiveSeen[field][testStr][sensitiveVal]) {
        sensitiveSeen[field][testStr][sensitiveVal]++;
      } else {
        sensitiveSeen[field][testStr][sensitiveVal] = 1;
      }
    }
  }
  return sensitiveSeen;
};

export const measureDataPrivacyStats = (
  dataSet: ParsedExecuteAccessorData,
  piiFields: string[],
  sensitiveFields: string[]
) => {
  const frequencies = tallyFrequencies(dataSet, piiFields);
  const smallestNumber = Object.keys(frequencies).reduce((smallest, field) => {
    if (frequencies[field] < smallest) {
      smallest = frequencies[field];
    }
    return smallest;
  }, dataSet.length);
  frequencies.kAnonymity = smallestNumber;

  let uniqueness: Record<string, Record<string, Record<string, number>>> = {};
  if (sensitiveFields.length) {
    uniqueness = tallySensitiveDataUniqueness(
      dataSet,
      sensitiveFields,
      piiFields
    );
    for (const field of sensitiveFields) {
      let minDiversity = dataSet.length;
      for (const bucket in uniqueness[field]) {
        const diversity = Object.keys(uniqueness[field][bucket]).length;
        if (diversity < minDiversity) {
          minDiversity = diversity;
        }
      }
      uniqueness[field].lDiversity = { value: minDiversity };
    }
  }

  return { frequencies, uniqueness };
};
