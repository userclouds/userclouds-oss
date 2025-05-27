export interface MFASubmitSettings {
  channel_type: string;
  channel_id: string;
  challenge_description: string;
  challenge_status: string;
  challenge_block_expiration: string;
  can_change_channel: boolean;
  can_reissue_challenge: boolean;
  can_submit_code: boolean;
  registration_link: string;
  registration_qr_code: string;
  customer_service_link: string;
  mfa_purpose: string;
}
