class HTTPError extends Error {
  statusCode: number;

  constructor(message: string, statusCode: number) {
    super(message);
    Object.setPrototypeOf(this, HTTPError.prototype);
    this.statusCode = statusCode;
  }
}

export default HTTPError;
