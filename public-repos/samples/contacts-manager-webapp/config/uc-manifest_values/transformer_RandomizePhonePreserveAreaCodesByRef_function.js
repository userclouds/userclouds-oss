function transform(data, params) {
  return randomizeDigitsAfterIndex(data,3);
}
function randomizeDigitsAfterIndex(numberStr, startIndex) {
  return numberStr.slice(0, startIndex) + numberStr.slice(startIndex).replace(/\d/g, () => Math.floor(Math.random() * 10));
}

