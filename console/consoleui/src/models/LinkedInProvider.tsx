import { JSONValue } from '@userclouds/sharedui';

class LinkedInProvider {
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

  static fromJSON(jsonStr?: JSONValue): LinkedInProvider {
    if (jsonStr === undefined) {
      return new LinkedInProvider('', '', '');
    }

    const lip = jsonStr as {
      client_id: string;
      client_secret: string;
      additional_scopes: string;
    };
    return new LinkedInProvider(
      lip.client_id,
      lip.client_secret,
      lip.additional_scopes
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

export default LinkedInProvider;
