interface MessageElement {
  type: string;
  default_value: string;
  custom_value: string;
}
interface MessageParameter {
  name: string;
  default_value: string;
}

export interface MessageTypeMessageElements {
  type: string;
  message_elements: Record<string, MessageElement>;
  message_parameters: MessageParameter[];
}

export interface AppMessageElement {
  app_id: string;
  message_type_message_elements: Record<string, MessageTypeMessageElements>;
}

export interface TenantAppMessageElements {
  tenant_app_message_elements: {
    tenant_id: string;
    app_message_elements: AppMessageElement[];
  };
}

export interface EmailMessageElementsSavePayload {
  modified_message_type_message_elements: {
    tenant_id: string;
    app_id: string;
    // TODO: should we make this a union type/enum?
    message_type: string;
    message_elements: {
      sender: string;
      sender_name: string;
      subject_template: string;
      html_template: string;
      text_template: string;
    };
  };
}

export interface SMSMessageElementsSavePayload {
  modified_message_type_message_elements: {
    tenant_id: string;
    app_id: string;
    message_type: string;
    message_elements: {
      sms_body_template: string;
      sms_sender: string;
    };
  };
}

export enum MessageType {
  EmailInviteNew = 'invite_new',
  EmailInviteExisting = 'invite_existing',
  EmailMFAChallenge = 'mfa_email_challenge',
  EmailMFAVerify = 'mfa_email_verify',
  EmailPasswordlessLogin = 'passwordless_login',
  EmailResetPassword = 'reset_password',
  EmailVerifyEmail = 'verify_email',
  SMSMFAChallenge = 'sms_mfa_challenge',
  SMSMFAVerify = 'sms_mfa_verify',
}

export const friendlyEmailMessageTypes: Record<string, string> = {
  [MessageType.EmailInviteNew]: 'Invite new user',
  [MessageType.EmailInviteExisting]: 'Invite existing user',
  [MessageType.EmailMFAChallenge]: 'MFA email challenge',
  [MessageType.EmailMFAVerify]: 'MFA email verification',
  [MessageType.EmailPasswordlessLogin]: 'Passwordless login',
  [MessageType.EmailResetPassword]: 'Reset password',
  [MessageType.EmailVerifyEmail]: 'Verify email',
};

export const friendlySMSMessageTypes: Record<string, string> = {
  [MessageType.SMSMFAChallenge]: 'MFA SMS challenge',
  [MessageType.SMSMFAVerify]: 'MFA SMS verification',
};

export function messageTypesForApp(
  messageElements: AppMessageElement[],
  appId: string
): Record<string, MessageTypeMessageElements> | undefined {
  const matchingApp: AppMessageElement | undefined = messageElements.find(
    (app) => app.app_id === appId
  );
  return matchingApp && 'message_type_message_elements' in matchingApp
    ? matchingApp.message_type_message_elements
    : undefined;
}

export function savedValueForElement(
  elements: MessageTypeMessageElements,
  elementType: string
): string {
  return (
    elements.message_elements[elementType].custom_value ||
    elements.message_elements[elementType].default_value
  );
}

export const modifyMessageElement = (
  messageElements: AppMessageElement[],
  appID: string,
  messageType: string,
  elementName: string,
  value: string
): AppMessageElement[] => {
  let matchingApp: AppMessageElement | undefined;
  const rest = messageElements.filter((app: AppMessageElement) => {
    if (app.app_id === appID) {
      matchingApp = app;
      return false;
    }
    return true;
  });
  if (matchingApp) {
    const { [messageType]: matchingType, ...otherTypes } =
      matchingApp.message_type_message_elements;
    const modifiedApp = {
      app_id: appID,
      message_type_message_elements: {
        ...otherTypes,
        [messageType]: {
          message_elements: {
            ...matchingType.message_elements,
            [elementName]: {
              custom_value: value,
              default_value:
                matchingType.message_elements[elementName].default_value,
              type: elementName,
            },
          },
          message_parameters: matchingType.message_parameters,
          type: messageType,
        },
      },
    };
    return [...rest, modifiedApp];
  }
  return messageElements;
};

export const cloneMessageSettings = (
  targetAppID: string,
  sourceAppID: string,
  messageElements: AppMessageElement[]
): AppMessageElement[] => {
  const sourceApp = messageElements.find(
    (app: AppMessageElement) => app.app_id === sourceAppID
  );
  if (sourceApp) {
    return messageElements.reduce(
      (acc: AppMessageElement[], app: AppMessageElement) => {
        if (app.app_id === targetAppID) {
          acc.push({
            app_id: targetAppID,
            message_type_message_elements: JSON.parse(
              JSON.stringify(sourceApp.message_type_message_elements)
            ),
          });
        } else {
          acc.push(app);
        }
        return acc;
      },
      []
    );
  }
  return messageElements;
};
