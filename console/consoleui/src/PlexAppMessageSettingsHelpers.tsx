import {
  Button,
  InlineNotification,
  Label,
  Select,
  Text,
} from '@userclouds/ui-component-lib';

import { AppDispatch } from './store';
import {
  MessageTypeMessageElements,
  MessageType,
} from './models/MessageElements';
import Styles from './pages/PlexAppPage.module.css';

const insertTemplateParameter =
  (
    appID: string,
    messageType: MessageType,
    param: string,
    targetElId: string,
    modifyMessageElements: (
      appID: string,
      messagetype: MessageType,
      element_name: string,
      value: string
    ) => any
  ) =>
  (dispatch: AppDispatch): void => {
    const targetEl: HTMLElement | null = document.getElementById(targetElId);

    if (targetEl) {
      let val;
      let selectionStart = 0;
      let selectionEnd = 0;
      if (targetEl instanceof HTMLTextAreaElement) {
        val = (targetEl as HTMLTextAreaElement).textContent;
        selectionStart = (targetEl as HTMLTextAreaElement).selectionStart || 0;
        selectionEnd =
          (targetEl as HTMLTextAreaElement).selectionEnd || selectionStart;
      } else {
        val = (targetEl as HTMLInputElement).value;
        selectionStart = (targetEl as HTMLInputElement).selectionStart || 0;
        selectionEnd =
          (targetEl as HTMLInputElement).selectionEnd || selectionStart;
      }
      const paramText = `{{.${param}}}`;
      const newValue = val
        ? `${val.substring(0, selectionStart)}${paramText}${val.substring(
            selectionEnd
          )}`
        : paramText;
      dispatch(
        modifyMessageElements(
          appID,
          messageType,
          targetEl.getAttribute('name') as string,
          newValue
        )
      );

      // Set focus and cursor position after the parameter
      setTimeout(() => {
        targetEl.focus();
        const newCursorPosition = selectionStart + paramText.length;
        if (targetEl instanceof HTMLTextAreaElement) {
          (targetEl as HTMLTextAreaElement).selectionStart = newCursorPosition;
          (targetEl as HTMLTextAreaElement).selectionEnd = newCursorPosition;
        } else {
          (targetEl as HTMLInputElement).selectionStart = newCursorPosition;
          (targetEl as HTMLInputElement).selectionEnd = newCursorPosition;
        }
      }, 0);
    }
  };

export const ParamChooser = ({
  appID,
  messageType,
  elements,
  fieldId,
  dispatch,
  modifyMessageElements,
}: {
  appID: string;
  messageType: MessageType;
  elements: MessageTypeMessageElements;
  fieldId: string;
  dispatch: AppDispatch;
  modifyMessageElements: (
    appID: string,
    messagetype: MessageType,
    element_name: string,
    value: string
  ) => any;
}): JSX.Element => {
  return (
    <Text className={Styles.paramChooser} elementName="div">
      <strong>Insert parameter:</strong>
      <ul>
        {elements.message_parameters.map((param) => (
          <li key={param.name}>
            <Button
              value={param.name}
              onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                const clickedParam = e.currentTarget.value;
                dispatch(
                  insertTemplateParameter(
                    appID,
                    messageType,
                    clickedParam,
                    fieldId,
                    modifyMessageElements
                  )
                );
              }}
              theme="outline"
            >
              {param.name}
            </Button>
          </li>
        ))}
      </ul>
    </Text>
  );
};

export const MessageTypeSelector = ({
  types,
  selectedMessageType,
  isDirty,
  dispatch,
  templateType,
  changeSelectedMessageType,
  friendlyMessageTypes,
}: {
  types: Record<string, MessageTypeMessageElements> | undefined;
  selectedMessageType: MessageType;
  isDirty: boolean;
  dispatch: AppDispatch;
  templateType: string;
  changeSelectedMessageType: (type: MessageType) => any;
  friendlyMessageTypes: Record<string, string>;
}) => {
  return types ? (
    <>
      <Label htmlFor="messageType">Choose {templateType} template</Label>
      <Select
        defaultValue={selectedMessageType}
        name={`${templateType}_message_type`}
        id={'messageType' + selectedMessageType.toString()}
        onChange={(e: React.ChangeEvent<HTMLSelectElement>) => {
          e.preventDefault();
          const selectedType = e.currentTarget.value;

          if (
            isDirty &&
            !window.confirm(
              'You have unsaved changes. Are you sure you want to switch to a different ' +
                templateType +
                ' template?'
            )
          ) {
            e.preventDefault();
            e.currentTarget.value = selectedMessageType;
            return;
          }
          dispatch(changeSelectedMessageType(selectedType as MessageType));
        }}
      >
        {Object.keys(types).map((type) => (
          <option value={type} key={type}>
            {friendlyMessageTypes[type]}
          </option>
        ))}
      </Select>
    </>
  ) : (
    <InlineNotification theme="alert">
      Issue fetching message types
    </InlineNotification>
  );
};
