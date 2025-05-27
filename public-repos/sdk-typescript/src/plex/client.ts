import BaseClient from '../uc/baseclient';

class Client extends BaseClient {
  async inviteUser(
    invitee_email: string,
    inviter_user_id: string,
    inviter_name: string,
    inviter_email: string,
    client_id: string,
    state: string,
    redirect_url: string,
    invite_text: string,
    expires: string
  ): Promise<void> {
    return this.makeRequest<void>(
      `/invite/send`,
      'POST',
      undefined,
      JSON.stringify({
        invitee_email,
        inviter_user_id,
        inviter_name,
        inviter_email,
        client_id,
        state,
        redirect_url,
        invite_text,
        expires,
      })
    );
  }
}

export default Client;
