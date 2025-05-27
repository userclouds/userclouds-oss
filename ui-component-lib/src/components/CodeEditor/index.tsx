import clsx from 'clsx';
import CodeMirror, { ReactCodeMirrorProps } from '@uiw/react-codemirror';
import { javascript } from '@codemirror/lang-javascript';
import { json } from '@codemirror/lang-json';

import styles from './index.module.scss';

interface CodeEditorProps extends ReactCodeMirrorProps {
  hasError?: boolean;
  disabled?: boolean;
  jsonExt?: boolean;
  javascriptExt?: boolean;
}

/** Textarea with various states and a min-height. */

function CodeEditor({
  hasError = false,
  disabled = false,
  readOnly = false,
  jsonExt = false,
  javascriptExt = false,
  ...args
}: CodeEditorProps): JSX.Element {
  const extensions = [];
  if (jsonExt) {
    extensions.push(json());
  }
  if (javascriptExt) {
    extensions.push(javascript({ jsx: false }));
  }
  return (
    <CodeMirror
      {...args}
      extensions={extensions}
      className={clsx({
        [styles.root]: true,
        [styles.disabled]: disabled,
        [styles.error]: hasError,
      })}
      readOnly={readOnly}
    />
  );
}

export default CodeEditor;
