type JSONValue =
  | string
  | number
  | boolean
  | { [x: string]: JSONValue | undefined }
  | Array<JSONValue>;

export default JSONValue;
