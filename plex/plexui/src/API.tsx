import {
  APIError,
  tryValidate,
  tryGetJSON,
  makeAPIError,
  HTTPError,
  extractErrorMessage,
} from '@userclouds/sharedui';
import { MFAChannelsResponse } from './models/MFAChannelsResponse';
import { MFASubmitSettings } from './models/MFASubmitSettings';
import { PageParametersResponse } from './models/PageParametersResponse';

// makePlexURL constructs URLs to Plex endpoints for AJAX requests
// and makes it easy to centrally fix up URLs if things change in the future.
// path: URL path string
// query: javascript object/dict, query string, or URLSearchParams object
export function makePlexURL(
  path: string,
  query?: string | Record<string, string>
): string {
  if (!query) {
    return path;
  }
  // Since this is running on the same protocol & host, use relative paths.
  return `${path}?${new URLSearchParams(query).toString()}`;
}

// TODO: figure out best way to keep path URLs in sync with server as well as the
// "basename" for the UI (/plexui/...).
class InternalClient {
  navigateToLogin(sessionID: string) {
    window.location.href = makePlexURL('/plexui/login', {
      session_id: sessionID,
    });
  }

  navigateToCreateAccount(sessionID: string) {
    window.location.href = makePlexURL('/plexui/createuser', {
      session_id: sessionID,
    });
  }

  navigateToPasswordReset(sessionID: string) {
    window.location.href = makePlexURL('/plexui/startresetpassword', {
      session_id: sessionID,
    });
  }

  navigateToConfigureMFA(sessionID: string) {
    window.location.href = makePlexURL('/plexui/mfachannel/configure', {
      session_id: sessionID,
    });
  }

  navigateToConfigureMFAChannel(sessionID: string, channelType: string) {
    window.location.href = makePlexURL('/plexui/mfachannel/configurechannel', {
      session_id: sessionID,
      channelType: channelType,
    });
  }

  navigateToChooseMFAChannel(sessionID: string) {
    window.location.href = makePlexURL('/plexui/mfachannel/choose', {
      session_id: sessionID,
    });
  }

  navigateToMFARecoverycode(sessionID: string) {
    window.location.href = makePlexURL('/plexui/mfashowrecoverycode', {
      session_id: sessionID,
    });
  }

  navigateToMFASubmit(sessionID: string, channelType: string) {
    window.location.href = makePlexURL(`/plexui/mfasubmit`, {
      session_id: sessionID,
      channelType: channelType,
    });
  }

  startSocialLogin(sessionID: string, provider: string, email: string = '') {
    window.location.href = makePlexURL('/social/login', {
      session_id: sessionID,
      oidc_provider: provider,
      email: email,
    });
  }

  async grantOrDenyAuthnAddPermission(
    sessionID: string,
    permission: boolean
  ): Promise<void> {
    return new Promise(async (resolve, reject) => {
      const url = makePlexURL('/create/grantordenyauthnaddpermission');
      const req = {
        session_id: sessionID,
        permission: permission,
      };
      try {
        const rawResponse = await fetch(url, {
          method: 'POST',
          body: JSON.stringify(req),
        });
        await tryValidate(rawResponse);
        resolve();
      } catch (e) {
        reject(makeAPIError(e, 'api error'));
      }
    });
  }

  async usernamePasswordLogin(
    sessionID: string,
    username: string,
    password: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/login');
    const req = {
      session_id: sessionID,
      username,
      password,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.href = typedResponse.redirect_to;
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaChallenge(
    sessionID: string,
    channelID: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/mfa/challenge');
    const req = {
      session_id: sessionID,
      mfa_channel_id: channelID,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async getNewChallenge(
    sessionID: string,
    channelID: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/mfa/challenge');
    const req = {
      session_id: sessionID,
      mfa_channel_id: channelID,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaCreateChannel(
    sessionID: string,
    channelType: string,
    channelTypeID: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/mfa/createchannel');
    const req = {
      session_id: sessionID,
      mfa_channel_type: channelType,
      mfa_channel_type_id: channelTypeID,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaDeleteChannel(
    sessionID: string,
    channelID: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/mfa/deletechannel');
    const req = {
      session_id: sessionID,
      mfa_channel_id: channelID,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaEndConfiguration(sessionID: string): Promise<void | APIError> {
    const url = makePlexURL('/mfa/endconfiguration', {
      session_id: sessionID,
    });
    try {
      const rawResponse = await fetch(url, {
        method: 'GET',
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaClearPrimaryChannel(sessionID: string): Promise<void | APIError> {
    const url = makePlexURL('/mfa/clearprimarychannel');
    const req = {
      session_id: sessionID,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaMakePrimaryChannel(
    sessionID: string,
    channelID: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/mfa/makeprimarychannel');
    const req = {
      session_id: sessionID,
      mfa_channel_id: channelID,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaReissueRecoveryCode(
    sessionID: string,
    channelID: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/mfa/reissuerecoverycode');
    const req = {
      session_id: sessionID,
      mfa_channel_id: channelID,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaSubmit(
    sessionID: string,
    mfaCode: string,
    evaluateSupportedChannels: boolean
  ): Promise<void | APIError> {
    const url = makePlexURL('/mfa/submit');
    const req = {
      session_id: sessionID,
      mfa_code: mfaCode,
      evaluate_supported_channels: evaluateSupportedChannels,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async mfaConfirmRecoveryCode(sessionID: string): Promise<void | APIError> {
    const url = makePlexURL('/mfa/confirmrecoverycode');
    const req = {
      session_id: sessionID,
    };
    try {
      const rawResponse = await fetch(url + `?session_id=${sessionID}`, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.replace(typedResponse.redirect_to);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  // returns true if successful
  // if email is already associated with an account, this function returns an array of login providers
  // associated with the email address
  async createUser(
    sessionID: string,
    clientID: string,
    email: string,
    username: string,
    password: string,
    name: string
  ): Promise<true | Array<string>> {
    return new Promise((resolve, reject) => {
      const url = makePlexURL('/create/submit');
      const req = {
        session_id: sessionID,
        client_id: clientID,
        email,
        username,
        password,
        name,
      };
      fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      }).then(
        (response) => {
          if (response.status === 202) {
            // Status code 202 means that the email is associated with other account(s),
            // and the response body contains a list of login providers associated with the other account(s)
            resolve(response.json());
          } else if (response.ok) {
            resolve(true);
          } else {
            extractErrorMessage(response).then((message: string) => {
              reject(makeAPIError(new HTTPError(message, response.status)));
            });
          }
        },
        (e) => {
          reject(makeAPIError(e));
        }
      );
    });
  }

  navigateToPasswordlessLogin(sessionID: string) {
    window.location.href = makePlexURL('/plexui/passwordlesslogin', {
      session_id: sessionID,
    });
  }

  async startPasswordlessLogin(
    sessionID: string,
    email: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/passwordless/start');
    const req = {
      session_id: sessionID,
      email,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      await tryValidate(rawResponse);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async finishPasswordlessLogin(
    sessionID: string,
    email: string,
    otpCode: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/otp/submit');
    const req = {
      session_id: sessionID,
      email,
      otp_code: otpCode,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      const jsonResponse = await tryGetJSON(rawResponse);
      const typedResponse = jsonResponse as {
        redirect_to: string;
      };
      window.location.href = typedResponse.redirect_to;
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async startResetPassword(
    sessionID: string,
    email: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/resetpassword/startsubmit');
    const req = {
      session_id: sessionID,
      email,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      await tryValidate(rawResponse);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async finishResetPassword(
    sessionID: string,
    otpCode: string,
    password: string
  ): Promise<void | APIError> {
    const url = makePlexURL('/resetpassword/resetsubmit');
    const req = {
      session_id: sessionID,
      otp_code: otpCode,
      password,
    };
    try {
      const rawResponse = await fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      });
      await tryValidate(rawResponse);
      this.navigateToLogin(sessionID);
      return undefined;
    } catch (e) {
      return makeAPIError(e);
    }
  }

  async verifyOTPEmail(
    sessionID: string,
    email: string,
    otpCode: string
  ): Promise<string> {
    const url = makePlexURL('/otp/submit');
    const req = {
      session_id: sessionID,
      email,
      otp_code: otpCode,
    };
    return new Promise((resolve, reject) => {
      return fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      })
        .then(
          (response) => {
            if (!response.ok) {
              reject(
                makeAPIError(
                  new Error(response.statusText),
                  `Failed to validate email ${email}`
                )
              );
            }
            return response.json();
          },
          (e) => {
            reject(makeAPIError(e, `Failed to validate email ${email}`));
          }
        )
        .then((response) => {
          if (!response.ok) {
            reject(
              makeAPIError(
                new Error(response.statusText),
                `Failed to validate email ${email}`
              )
            );
          }
          resolve(response.json());
        })
        .catch((e) => {
          reject(makeAPIError(e, `Failed to validate email ${email}`));
        });
    });
  }

  async fetchMFAChannels(sessionID: string): Promise<MFAChannelsResponse> {
    const url = `/mfa/getchannels?session_id=${sessionID}`;
    return new Promise((resolve, reject) => {
      return fetch(url, {
        method: 'GET',
      }).then(
        (response) => {
          if (!response.ok) {
            reject(
              makeAPIError(
                new Error(response.statusText),
                'failed to fetch MFA channels'
              )
            );
          }
          resolve(response.json());
        },
        (e) => {
          reject(makeAPIError(e, 'failed to fetch MFA channels'));
        }
      );
    });
  }

  async fetchMFASubmitSettings(sessionID: string): Promise<MFASubmitSettings> {
    const url = `/mfa/getsubmitsettings?session_id=${sessionID}`;
    return new Promise((resolve, reject) => {
      return fetch(url, {
        method: 'GET',
      }).then(
        (response) => {
          if (!response.ok) {
            reject(
              makeAPIError(
                new Error(response.statusText),
                'failed to fetch MFA submit settings'
              )
            );
          }
          resolve(response.json());
        },
        (e) => {
          reject(makeAPIError(e, 'failed to fetch MFA submit settings'));
        }
      );
    });
  }

  async fetchPageParameters(
    sessionID: string,
    pageType: string,
    parameterNames: string[]
  ): Promise<PageParametersResponse> {
    const url = makePlexURL('/login/pageparameters');
    const req = {
      session_id: sessionID,
      page_type: pageType,
      parameter_names: parameterNames,
    };
    return new Promise((resolve, reject) => {
      return fetch(url, {
        method: 'POST',
        body: JSON.stringify(req),
      })
        .then(
          (response) => {
            if (!response.ok) {
              reject(
                makeAPIError(
                  new Error(response.statusText),
                  'failed to fetch page parameters'
                )
              );
            }
            return response.json();
          },
          (e) => {
            reject(makeAPIError(e, 'failed to fetch page parameters'));
          }
        )
        .then(resolve, (e) => {
          reject(makeAPIError(e, 'failed to fetch page parameters'));
        });
    });
  }
}

const API = new InternalClient();

export default API;
