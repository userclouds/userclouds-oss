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
import { saveTenantEmailMessageElements } from './API/authn';
import {
  updateEmailMessageElementsRequest,
  updateEmailMessageElementsSuccess,
  updateEmailMessageElementsError,
  changeSelectedEmailMessageType,
  modifyEmailMessageElements,
  resetEmailMessageElements,
} from './actions/authn';
import {
  MessageTypeMessageElements,
  EmailMessageElementsSavePayload,
  messageTypesForApp,
  savedValueForElement,
  friendlyEmailMessageTypes,
  MessageType,
  AppMessageElement,
} from './models/MessageElements';
import {
  ParamChooser,
  MessageTypeSelector,
} from './PlexAppMessageSettingsHelpers';

// TODO: unit-test this
const serializeEmailFormData = (
  formData: FormData,
  elements: MessageTypeMessageElements
): EmailMessageElementsSavePayload => {
  const tenantId = formData.get('tenant_id')! as string;
  const appId = formData.get('app_id')! as string;
  const messageType = formData.get('email_message_type')! as string;
  const sender = (formData.get('sender')! as string).trim();
  const sender_name = (formData.get('sender_name')! as string).trim();
  const subject = (formData.get('subject_template')! as string).trim();
  const html = (formData.get('html_template')! as string).trim();
  const text = (formData.get('text_template')! as string).trim();

  // NOTE:ksj: we could DRY the message_elements part, but it doesn't seem that useful to me
  return {
    modified_message_type_message_elements: {
      tenant_id: tenantId,
      app_id: appId,
      message_type: messageType,
      message_elements: {
        sender:
          sender !== savedValueForElement(elements, 'sender')
            ? sender
            : elements.message_elements.sender.custom_value,
        sender_name:
          sender_name !== savedValueForElement(elements, 'sender_name')
            ? sender_name
            : elements.message_elements.sender_name.custom_value,
        subject_template:
          subject !== savedValueForElement(elements, 'subject_template')
            ? subject
            : elements.message_elements.subject_template.custom_value,
        html_template:
          html !== savedValueForElement(elements, 'html_template')
            ? html
            : elements.message_elements.html_template.custom_value,
        text_template:
          text !== savedValueForElement(elements, 'text_template')
            ? text
            : elements.message_elements.text_template.custom_value,
      },
    },
  };
};

const saveEmailElements =
  (formData: FormData, elements: MessageTypeMessageElements) =>
  (dispatch: AppDispatch): void => {
    if (!elements) {
      return;
    }

    const dataToSubmit = serializeEmailFormData(formData, elements);
    dispatch(updateEmailMessageElementsRequest());
    saveTenantEmailMessageElements(dataToSubmit).then(
      (resp) => {
        dispatch(
          updateEmailMessageElementsSuccess(
            resp.tenant_app_message_elements.app_message_elements
          )
        );
      },
      (error: APIError) => {
        dispatch(updateEmailMessageElementsError(error));
      }
    );
  };

const EmailTemplateEditor = ({
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
      title="Email settings"
      description="Customize your end-user-facing emails for this application."
    >
      {elements ? (
        <form
          id="emailSettings"
          onSubmit={(e: React.FormEvent<HTMLFormElement>) => {
            e.preventDefault();

            const formData: FormData = new FormData(e.currentTarget);
            dispatch(saveEmailElements(formData, elements));
          }}
        >
          <input type="hidden" name="tenant_id" value={tenantID} />
          <input type="hidden" name="app_id" value={plexAppID} />
          <MessageTypeSelector
            types={messageTypes}
            selectedMessageType={selectedMessageType}
            isDirty={isDirty}
            dispatch={dispatch}
            templateType="email"
            changeSelectedMessageType={changeSelectedEmailMessageType}
            friendlyMessageTypes={friendlyEmailMessageTypes}
          />
          <Label className={GlobalStyles['mt-6']}>
            Sender email
            <TextInput
              id="sender"
              name="sender"
              type="email"
              value={
                elements?.message_elements.sender.custom_value ||
                elements?.message_elements.sender.default_value
              }
              onChange={(e: React.ChangeEvent) => {
                dispatch(
                  modifyEmailMessageElements(
                    plexAppID,
                    selectedMessageType,
                    'sender',
                    (e.target as HTMLInputElement).value
                  )
                );
              }}
            />
          </Label>

          <Label className={GlobalStyles['mt-3']}>
            Sender name
            <TextInput
              id="sender_name"
              name="sender_name"
              value={
                elements?.message_elements.sender_name.custom_value ||
                elements?.message_elements.sender_name.default_value
              }
              onChange={(e: React.ChangeEvent) => {
                dispatch(
                  modifyEmailMessageElements(
                    plexAppID,
                    selectedMessageType,
                    'sender_name',
                    (e.target as HTMLInputElement).value
                  )
                );
              }}
            />
          </Label>

          <fieldset>
            <Label className={GlobalStyles['mt-3']}>
              Subject
              <TextInput
                id="subject_template"
                name="subject_template"
                value={
                  elements?.message_elements.subject_template.custom_value ||
                  elements?.message_elements.subject_template.default_value
                }
                onChange={(e: React.ChangeEvent) => {
                  dispatch(
                    modifyEmailMessageElements(
                      plexAppID,
                      selectedMessageType,
                      'subject_template',
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
              fieldId="subject_template"
              dispatch={dispatch}
              modifyMessageElements={modifyEmailMessageElements}
            />
          </fieldset>

          <fieldset>
            <Label className={GlobalStyles['mt-3']}>
              HTML body
              <TextArea
                id="html_template"
                name="html_template"
                value={
                  elements?.message_elements.html_template.custom_value ||
                  elements?.message_elements.html_template.default_value
                }
                rows={12}
                cols={80}
                onChange={(e: React.ChangeEvent) => {
                  dispatch(
                    modifyEmailMessageElements(
                      plexAppID,
                      selectedMessageType,
                      'html_template',
                      (e.target as HTMLTextAreaElement).value
                    )
                  );
                }}
              />
            </Label>
            <ParamChooser
              appID={plexAppID}
              messageType={selectedMessageType}
              elements={elements}
              fieldId="html_template"
              dispatch={dispatch}
              modifyMessageElements={modifyEmailMessageElements}
            />
          </fieldset>

          <fieldset>
            <Label className={GlobalStyles['mt-3']}>
              Text body
              <TextArea
                id="text_template"
                name="text_template"
                value={
                  elements?.message_elements.text_template.custom_value ||
                  elements?.message_elements.text_template.default_value
                }
                rows={12}
                cols={80}
                onChange={(e: React.ChangeEvent) => {
                  dispatch(
                    modifyEmailMessageElements(
                      plexAppID,
                      selectedMessageType,
                      'text_template',
                      (e.target as HTMLTextAreaElement).value
                    )
                  );
                }}
              />
            </Label>
            <ParamChooser
              appID={plexAppID}
              messageType={selectedMessageType}
              elements={elements}
              fieldId="text_template"
              dispatch={dispatch}
              modifyMessageElements={modifyEmailMessageElements}
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
              Save Email Configuration
            </Button>
            <Button
              theme="secondary"
              isLoading={isSaving}
              onClick={() => {
                dispatch(resetEmailMessageElements());
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
  modifiedMessageElements: state.modifiedEmailMessageElements,
  isDirty: state.emailMessageElementsAreDirty,
  isFetching: state.fetchingEmailMessageElements,
  isSaving: state.savingEmailMessageElements,
  fetchError: state.emailMessageElementsFetchError,
  saveSuccess: state.emailMessageElementsSaveSuccess,
  saveError: state.emailMessageElementsSaveError,
  selectedMessageType: state.selectedEmailMessageType,
}))(EmailTemplateEditor);
