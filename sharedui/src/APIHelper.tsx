import APIError from './APIError';
import HTTPError from './HTTPError';
import JSONValue from './JSONValue';

interface ObjectWithErrorProperty {
  error: JSONValue;
}

const extractErrorMessage = async (response: Response): Promise<string> => {
  let errMsg: JSONValue = '';
  try {
    const responseText = await response.text();
    errMsg = responseText;

    const responseJSON = JSON.parse(responseText);

    // OIDC/OAuth methods use `error_description` for human readable messages,
    // other methods use `error`, and if all else fails go for raw response.
    if (responseJSON.error_description) {
      errMsg = responseJSON.error_description;
    } else if (responseJSON.error) {
      errMsg = responseJSON.error;
      let o = errMsg as unknown as ObjectWithErrorProperty;
      while (o.error) {
        errMsg = o.error;
        o = errMsg as unknown as ObjectWithErrorProperty;
      }
    }
  } catch {
    // Do nothing
  }

  if (typeof errMsg !== 'string') {
    errMsg = JSON.stringify(errMsg);
  }

  return errMsg || '[no error returned]';
};
const tryValidate = async (response: Response): Promise<Response> => {
  // Handle failed requests as best as possible
  if (!response.ok) {
    const errMsg = await extractErrorMessage(response);

    throw new HTTPError(errMsg, response.status);
  }
  return response;
};

const tryGetJSON = async (response: Response): Promise<JSONValue> => {
  const validatedResponse = await tryValidate(response);
  return validatedResponse.json();
};

const makeAPIError = (e: unknown, errorMsg = ''): APIError => {
  let message = errorMsg;
  if (e instanceof Error) {
    message = errorMsg ? `${errorMsg}: ${e.message}` : e.message;
  }
  return new APIError(
    message,
    e instanceof HTTPError ? e.statusCode : 0,
    e instanceof Error ? e : undefined
  );
};

export { extractErrorMessage, tryValidate, tryGetJSON, makeAPIError };
