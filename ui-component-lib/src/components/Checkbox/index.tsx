import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

/**
 * Standard checkbox with label surrounding text and input element.
 */

const Checkbox: React.FC<React.ComponentProps<'input'>> = ({
  children,
  checked,
  disabled,
  ...args
}) => {
  const labelClasses = clsx({
    [styles.label]: true,
    [styles.disabled]: disabled,
    [styles.isChecked]: checked,
  });

  return (
    <label className={labelClasses}>
      <input
        className={styles.input}
        type="checkbox"
        disabled={disabled}
        checked={checked}
        {...args}
      />
      <div className={styles.inputImage}>
        {checked && (
          <svg
            width="16"
            height="16"
            strokeWidth="2"
            viewBox="0 0 24 24"
            fill="none"
            xmlns="http://www.w3.org/2000/svg"
            fr="0"
          >
            <path
              d="M5 13l4 4L19 7"
              stroke="currentColor"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        )}
      </div>
      <div className={styles.text}>{children}</div>
    </label>
  );
};

export default Checkbox;
