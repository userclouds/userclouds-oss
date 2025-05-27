import { JSONValue } from '@userclouds/sharedui';

export type UserEvent = {
  id: string;
  created: string;
  type: string;
  user_alias: string;
  payload: JSONValue;
};

export type UserAuthnSerialized = {
  authn_type: string;

  // Fields present if authn_type == "password"
  username: string;

  // Fields present if authn_type == "social"
  oidc_provider: string;
  oidc_issuer_url: string;
  oidc_subject: string;
};

export type MFAChannelSerialized = {
  mfa_channel_type: string;
  mfa_channel_description: string;
  primary: boolean;
  verified: boolean;
};

// NOTE: try to keep the field order/naming close to Golang definitions in all of these
// types to make it easier to keep them in sync and visually catch bugs/omissions.
export type UserProfileSerialized = {
  id: string;
  profile: Record<string, JSONValue>;
  organization_id: string;
  authns: UserAuthnSerialized[];
  mfa_channels: MFAChannelSerialized[];
};

export class MyProfile {
  userProfile: UserProfile;

  impersonatorProfile: UserProfile | null;

  constructor(
    userProfile: UserProfile,
    impersonatorProfile: UserProfile | null
  ) {
    this.userProfile = userProfile;
    this.impersonatorProfile = impersonatorProfile;
  }

  static fromJSON(json: JSONValue): MyProfile {
    const myProfileJSON = json as {
      user_profile: UserProfileSerialized;
      impersonator_profile: UserProfileSerialized | null;
    };

    return new MyProfile(
      UserProfile.fromJSON(myProfileJSON.user_profile),
      myProfileJSON.impersonator_profile
        ? UserProfile.fromJSON(myProfileJSON.impersonator_profile)
        : null
    );
  }
}

export type UserBaseProfile = {
  id: string;
  organization_id: string;
  email: string | undefined;
  email_verified: string | undefined;
  name: string | undefined;
  nickname: string | undefined;
  picture: string | undefined;
};

export class UserProfile {
  id: string;

  profile: Record<string, JSONValue>;

  authns: UserAuthnSerialized[];

  mfaChannels: MFAChannelSerialized[];

  constructor(
    userID: string,
    profile: Record<string, JSONValue>,
    authns: UserAuthnSerialized[],
    mfaChannels: MFAChannelSerialized[]
  ) {
    this.id = userID;
    this.profile = profile;
    this.authns = authns;
    this.mfaChannels = mfaChannels;
  }

  static fromJSON(json: JSONValue): UserProfile {
    const userJSON = json as UserProfileSerialized;
    return new UserProfile(
      userJSON.id,
      userJSON.profile,
      userJSON.authns ? [...userJSON.authns] : [],
      userJSON.mfa_channels ? [...userJSON.mfa_channels] : []
    );
  }

  // Called by JSON.stringify
  toJSON(): JSONValue {
    const profile: Record<string, string> = {};
    for (const key in this.profile) {
      if (!this.profile[key]) {
        profile[key] = '';
      } else if (typeof this.profile[key] === 'object') {
        profile[key] = JSON.stringify(this.profile[key]);
      } else {
        profile[key] = this.profile[key]!.toString();
      }
    }
    // this is returning an idp.UpdateUserRequest
    return {
      profile: profile,
    };
  }

  // Returns the user's picture URL or a suitable default if not set
  pictureURL(): string {
    return this.profile.picture?.toString() || '/mystery-person.webp';
  }

  name(): string {
    return this.profile.name?.toString() || '';
  }

  nickname(): string {
    return this.profile.nickname?.toString() || '';
  }

  email(): string {
    return this.profile.email?.toString() || '';
  }

  emailVerified(): boolean {
    return this.profile.email_verified?.toString() === 'true';
  }
}

export const prettyProfileColumnName = (name: string): string => {
  return name
    .replace(/_/g, ' ')
    .split(' ')
    .map((s) => s.charAt(0).toUpperCase() + s.substring(1))
    .join(' ');
};
