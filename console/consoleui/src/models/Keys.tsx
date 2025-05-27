import { JSONValue } from '@userclouds/sharedui';

class Keys {
  keyID: string;

  publicKey: string;

  privateKey: string;

  constructor(keyID: string, publicKey: string, privateKey: string) {
    this.keyID = keyID;
    this.publicKey = publicKey;
    this.privateKey = privateKey;
  }

  toJSON(): JSONValue {
    return {
      key_id: this.keyID,
      public_key: this.publicKey,
      private_key: this.privateKey,
    };
  }

  static fromJSON(jsonStr: JSONValue): Keys {
    const ks = jsonStr as {
      key_id: string;
      public_key: string;
      private_key: string;
    };

    return new Keys(ks.key_id, ks.public_key, ks.private_key);
  }
}

export default Keys;
