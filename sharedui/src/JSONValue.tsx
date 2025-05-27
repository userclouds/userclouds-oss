type JSONValue =
  | string
  | number
  | boolean
  | null
  | { [x: string]: JSONValue | undefined }
  | Array<JSONValue>;

export default JSONValue;
