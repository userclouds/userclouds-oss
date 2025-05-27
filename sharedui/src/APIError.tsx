class APIError extends Error {
  code: number;

  e: Error | undefined;

  constructor(message: string, code: number, e: Error | undefined) {
    super(message);
    Object.setPrototypeOf(this, APIError.prototype);
    this.code = code;
    this.e = e;
  }
}

export default APIError;
