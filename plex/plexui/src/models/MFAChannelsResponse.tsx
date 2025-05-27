export interface MFAChannelType {
  mfa_channel_type: string;
  can_create: boolean;
}

export interface MFAChannel {
  mfa_channel_id: string;
  mfa_channel_type: string;
  mfa_channel_description: string;
  primary: boolean;
  can_challenge: boolean;
  challenge_block_expiration: string;
  can_delete: boolean;
  can_make_primary: boolean;
  can_reissue: boolean;
}

export interface MFAAuthenticatorType {
  authenticator_type: string;
  description: string;
}

export interface MFAChannelsResponse {
  mfa_purpose: string;
  mfa_channel_types: MFAChannelType[];
  mfa_channels: MFAChannel[];
  mfa_authenticator_types: MFAAuthenticatorType[];
  can_disable: boolean;
  can_dismiss: boolean;
  can_enable: boolean;
  max_mfa_channels: number;
  description: string;
}
