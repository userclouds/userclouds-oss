function transform(data, params) {
  return randomizeEmail(data);
}
function randomizeEmail(emailAddress) {
  const indexOfAt = emailAddress.indexOf("@");
  const emailName = emailAddress.slice(0, indexOfAt);
  const domain = emailAddress.slice(indexOfAt + 1);

  const indexOfDot = domain.lastIndexOf(".");
  const tld = domain.slice(indexOfDot + 1); // Extract the TLD
  const domainWithoutTld = domain.slice(0, indexOfDot);

  const characterPool = "abcdefghijklmnopqrstuvwxyz0123456789";

  const randomizedEmailName = Array.from({ length: emailName.length })
    .map(() => characterPool[Math.floor(Math.random() * characterPool.length)])
    .join("");

  const randomizedDomainWithoutTld = Array.from({ length: domainWithoutTld.length })
    .map(() => characterPool[Math.floor(Math.random() * characterPool.length)])
    .join("");

  return randomizedEmailName + "@" + randomizedDomainWithoutTld + "." + tld;
}


