/* eslint jsx-a11y/click-events-have-key-events: 0 */
/* eslint jsx-a11y/no-static-element-interactions: 0 */
import React, { useImperativeHandle, forwardRef, useRef } from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

interface TextInputPropsBase extends React.ComponentProps<'input'> {
  /** If input has an error */
  hasError?: boolean;
  /** Static `<Icon...>` that should be at the left of the input */
  innerLeft?: React.ReactNode;
  /** Clickable `<IconButton>` that should be at the right of the input */
  innerRight?: React.ReactNode;
  /** Monospace */
  monospace?: boolean;
}

export interface TextInputProps extends Omit<TextInputPropsBase, 'size'> {
  /** Size theme */
  size?: 'large' | 'medium' | 'small' | 'auto';
}

/**
 * Text input with optional `<Icon...>` on left and optional `<IconButton>` on the right.
 */

const TextInput = forwardRef<HTMLInputElement, TextInputProps>(
  (
    {
      hasError = false,
      innerLeft,
      innerRight,
      size = 'large',
      monospace = false,
      type = 'text',
      ...args
    },
    _forwardRef
  ) => {
    const inputRef = useRef(null);
    useImperativeHandle(_forwardRef, () => inputRef.current);

    // This passes through any clicks on the left icon to the input
    const focusInput = () => {
      // `current` points to the mounted text input element
      inputRef.current.focus();
    };

    const rootStyles = clsx({
      [styles.root]: true,
      [styles.innerLeftRoot]: innerLeft,
      [styles.innerRightRoot]: innerRight,
      [styles.hasError]: hasError,
      // we should probably style entirely from `[disabled]`
      // but some of the styles apply to the wrapper
      // and :has() isn't supported in all browsers
      [styles.disabled]: args.disabled,
      [styles.monospace]: monospace,
      [styles[size]]: true,
    });

    return (
      <div className={rootStyles}>
        {innerLeft && (
          <div className={styles.innerLeft} onClick={focusInput}>
            {innerLeft}
          </div>
        )}
        <input type={type} className={styles.input} ref={inputRef} {...args} />
        {innerRight && <div className={styles.innerRight}>{innerRight}</div>}
        <div className={styles.wrapper} />
      </div>
    );
  }
);

TextInput.displayName = 'TextInput';

export default TextInput;
