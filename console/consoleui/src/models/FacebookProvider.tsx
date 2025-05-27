import { JSONValue } from '@userclouds/sharedui';

class FacebookProvider {
  clientID: string;

  clientSecret: string;

  useLocalHostRedirect: boolean;

  additionalScopes: string;

  constructor(
    clientID: string,
    clientSecret: string,
    useLocalHostRedirect: boolean,
    additionalScopes: string
  ) {
    this.clientID = clientID;
    this.clientSecret = clientSecret;
    this.useLocalHostRedirect = useLocalHostRedirect;
    this.additionalScopes = additionalScopes;
  }

  static fromJSON(jsonStr?: JSONValue): FacebookProvider {
    if (jsonStr === undefined) {
      return new FacebookProvider('', '', false, '');
    }

    const fp = jsonStr as {
      client_id: string;
      client_secret: string;
      use_local_host_redirect: boolean;
      additional_scopes: string;
    };
    return new FacebookProvider(
      fp.client_id,
      fp.client_secret,
      fp.use_local_host_redirect,
      fp.additional_scopes
    );
  }

  isEmpty(): boolean {
    return this.clientID === '' && this.clientSecret === '';
  }

  toJSON(): JSONValue {
    return {
      client_id: this.clientID,
      client_secret: this.clientSecret,
      use_local_host_redirect: this.useLocalHostRedirect,
      additional_scopes: this.additionalScopes,
    };
  }
}

export default FacebookProvider;
