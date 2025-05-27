import { JSONValue } from '@userclouds/sharedui';

class MicrosoftProvider {
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

  static fromJSON(jsonStr?: JSONValue): MicrosoftProvider {
    if (jsonStr === undefined) {
      return new MicrosoftProvider('', '', '');
    }

    const lip = jsonStr as {
      client_id: string;
      client_secret: string;
      additional_scopes: string;
    };
    return new MicrosoftProvider(
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

export default MicrosoftProvider;
