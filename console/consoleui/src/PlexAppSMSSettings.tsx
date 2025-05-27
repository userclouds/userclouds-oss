import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  GlobalStyles,
  InlineNotification,
  Label,
  Text,
  TextArea,
  TextInput,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { AppDispatch, RootState } from './store';
import { saveTenantSMSMessageElements } from './API/authn';
import {
  updateSMSMessageElementsRequest,
  updateSMSMessageElementsSuccess,
  updateSMSMessageElementsError,
  changeSelectedSMSMessageType,
  modifySMSMessageElements,
  resetSMSMessageElements,
} from './actions/authn';
import {
  MessageTypeMessageElements,
  SMSMessageElementsSavePayload,
  messageTypesForApp,
  savedValueForElement,
  friendlySMSMessageTypes,
  MessageType,
  AppMessageElement,
} from './models/MessageElements';
import {
  ParamChooser,
  MessageTypeSelector,
} from './PlexAppMessageSettingsHelpers';

// TODO: unit-test this
const serializeSMSFormData = (
  formData: FormData,
  elements: MessageTypeMessageElements
): SMSMessageElementsSavePayload => {
  const tenantId = formData.get('tenant_id')! as string;
  const appId = formData.get('app_id')! as string;
  const messageType = formData.get('sms_message_type')! as string;
  const body = (formData.get('sms_body_template')! as string).trim();
  const sender = (formData.get('sms_sender')! as string).trim();

  // NOTE:ksj: we could DRY the message_elements part, but it doesn't seem that useful to me
  return {
    modified_message_type_message_elements: {
      tenant_id: tenantId,
      app_id: appId,
      message_type: messageType,
      message_elements: {
        sms_body_template:
          body !== savedValueForElement(elements, 'sms_body_template')
            ? body
            : elements.message_elements.sms_body_template.custom_value,
        sms_sender:
          sender !== savedValueForElement(elements, 'sms_sender')
            ? sender
            : elements.message_elements.sms_sender.custom_value,
      },
    },
  };
};

const saveSMSElements =
  (formData: FormData, elements: MessageTypeMessageElements) =>
  (dispatch: AppDispatch): void => {
    if (!elements) {
      return;
    }

    const dataToSubmit = serializeSMSFormData(formData, elements);
    dispatch(updateSMSMessageElementsRequest());
    saveTenantSMSMessageElements(dataToSubmit).then(
      (resp) => {
        dispatch(
          updateSMSMessageElementsSuccess(
            resp.tenant_app_message_elements.app_message_elements
          )
        );
      },
      (error: APIError) => {
        dispatch(updateSMSMessageElementsError(error));
      }
    );
  };

const SMSTemplateEditor = ({
  plexAppID,
  tenantID,
  modifiedMessageElements,
  isDirty,
  isFetching,
  isSaving,
  fetchError,
  saveSuccess,
  saveError,
  selectedMessageType,
  dispatch,
}: {
  plexAppID: string;
  tenantID: string;
  modifiedMessageElements: AppMessageElement[] | undefined;
  isDirty: boolean;
  isFetching: boolean;
  isSaving: boolean;
  fetchError: string;
  saveSuccess: boolean;
  saveError: string;
  selectedMessageType: MessageType;
  dispatch: AppDispatch;
}) => {
  const elements: MessageTypeMessageElements | undefined =
    modifiedMessageElements?.find(
      (app: AppMessageElement) => app.app_id === plexAppID
    )?.message_type_message_elements[selectedMessageType];
  const messageTypes = messageTypesForApp(
    modifiedMessageElements || [],
    plexAppID
  );
  return (
    <Card
      title="SMS settings"
      description="Customize your end-user-facing SMS messages for this application."
    >
      {elements ? (
        <form
          id="smsSettings"
          onSubmit={(e: React.FormEvent<HTMLFormElement>) => {
            e.preventDefault();

            const formData: FormData = new FormData(e.currentTarget);
            dispatch(saveSMSElements(formData, elements));
          }}
        >
          <input type="hidden" name="tenant_id" value={tenantID} />
          <input type="hidden" name="app_id" value={plexAppID} />
          <MessageTypeSelector
            types={messageTypes}
            selectedMessageType={selectedMessageType}
            isDirty={isDirty}
            dispatch={dispatch}
            templateType="SMS"
            changeSelectedMessageType={changeSelectedSMSMessageType}
            friendlyMessageTypes={friendlySMSMessageTypes}
          />
          <Label className={GlobalStyles['mt-6']}>
            Sender phone number
            <TextInput
              id="sms_sender"
              name="sms_sender"
              value={
                elements?.message_elements.sms_sender.custom_value ||
                elements?.message_elements.sms_sender.default_value
              }
              onChange={(e: React.ChangeEvent) => {
                dispatch(
                  modifySMSMessageElements(
                    plexAppID,
                    selectedMessageType,
                    'sms_sender',
                    (e.target as HTMLInputElement).value
                  )
                );
              }}
            />
          </Label>

          <fieldset>
            <Label className={GlobalStyles['mt-3']}>
              Message body
              <TextArea
                id="sms_body_template"
                name="sms_body_template"
                value={
                  elements?.message_elements.sms_body_template.custom_value ||
                  elements?.message_elements.sms_body_template.default_value
                }
                rows={12}
                cols={80}
                onChange={(e: React.ChangeEvent) => {
                  dispatch(
                    modifySMSMessageElements(
                      plexAppID,
                      selectedMessageType,
                      'sms_body_template',
                      (e.target as HTMLInputElement).value
                    )
                  );
                }}
              />
            </Label>
            <ParamChooser
              appID={plexAppID}
              messageType={selectedMessageType}
              elements={elements}
              fieldId="sms_body_template"
              dispatch={dispatch}
              modifyMessageElements={modifySMSMessageElements}
            />
          </fieldset>

          {saveError && (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          )}
          {saveSuccess && (
            <InlineNotification theme="success">
              Changes successfully saved!
            </InlineNotification>
          )}
          <br />
          <ButtonGroup>
            <Button
              type="submit"
              theme="primary"
              isLoading={isSaving}
              disabled={!isDirty || isSaving}
            >
              Save SMS Configuration
            </Button>
            <Button
              theme="secondary"
              isLoading={isSaving}
              onClick={() => {
                dispatch(resetSMSMessageElements());
              }}
              disabled={!isDirty || isSaving}
            >
              Cancel
            </Button>
          </ButtonGroup>
        </form>
      ) : fetchError ? (
        <InlineNotification theme="alert">{fetchError}</InlineNotification>
      ) : isFetching ? (
        <Text>Loading ...</Text>
      ) : (
        <InlineNotification theme="alert">
          Something went wrong
        </InlineNotification>
      )}
    </Card>
  );
};

export default connect((state: RootState) => ({
  modifiedMessageElements: state.modifiedSMSMessageElements,
  isDirty: state.smsMessageElementsAreDirty,
  isFetching: state.fetchingSMSMessageElements,
  isSaving: state.savingSMSMessageElements,
  fetchError: state.smsMessageElementsFetchError,
  saveSuccess: state.smsMessageElementsSaveSuccess,
  saveError: state.smsMessageElementsSaveError,
  selectedMessageType: state.selectedSMSMessageType,
}))(SMSTemplateEditor);
