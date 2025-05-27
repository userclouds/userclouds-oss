import { JSONValue } from '@userclouds/sharedui';

class GoogleProvider {
  clientID: string;

  clientSecret: string;

  additionalScopes: string;

  constructor(
    clientID: string,
    clientSecret: string,
    additionalScopes: string
  ) {
    this.clientID = clientID;
    this.clientSecret = clientSecret;
    this.additionalScopes = additionalScopes;
  }

  static fromJSON(jsonStr?: JSONValue): GoogleProvider {
    if (jsonStr === undefined) {
      return new GoogleProvider('', '', '');
    }

    const gp = jsonStr as {
      client_id: string;
      client_secret: string;
      additional_scopes: string;
    };
    return new GoogleProvider(
      gp.client_id,
      gp.client_secret,
      gp.additional_scopes
    );
  }

  isEmpty(): boolean {
    return this.clientID === '' && this.clientSecret === '';
  }

  toJSON(): JSONValue {
    return {
      client_id: this.clientID,
      client_secret: this.clientSecret,
      additional_scopes: this.additionalScopes,
    };
  }
}

export default GoogleProvider;
