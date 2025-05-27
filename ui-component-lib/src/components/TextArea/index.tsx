import React from 'react';
import clsx from 'clsx';
import styles from './index.module.scss';

interface TextAreaProps extends React.ComponentProps<'textarea'> {
  /** If TextArea has an error */
  hasError?: boolean;
  /** If TextArea should have monospace text */
  monospace?: boolean;
}

/** Textarea with various states and a min-height. */

function TextArea({
  hasError = false,
  monospace = false,
  ...args
}: TextAreaProps): JSX.Element {
  return (
    <textarea
      {...args}
      className={clsx({
        [styles.root]: true,
        [styles.disabled]: args.disabled,
        [styles.error]: hasError,
        [styles.monospace]: monospace,
      })}
    />
  );
}

export default TextArea;
