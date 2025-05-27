function transform(data, params) {
  return markString(data);
}
function markString(data) {
  
  const characterPool = "abcdefghijklmnopqrstuvwxyz";

  return Array.from({ length: data.length })
    .map(() => characterPool[Math.floor(Math.random() * characterPool.length)])
    .join("");  
}

