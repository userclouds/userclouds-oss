import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

export interface LabelProps extends React.ComponentProps<'label'> {
  /** `className` for overrides if needed, best to avoid */
  hasError?: boolean;
  /** If it's in a disabled state */
  disabled?: boolean;
}

/**
 * Commonly used with the input components, like `<TextInput>` or `<Select>`. Clicking
 * on the label will place focus its input by using the `htmlFor` attr and maching that
 * to the `id` of the input.
 *
 * Note that its disabled and error states should be tied with the state of the input.
 */

const Label: React.FC<LabelProps> = ({
  children,
  className,
  hasError = false,
  disabled = false,
  ...args
}) => {
  const classes = clsx({
    [styles.root]: true,
    [styles.disabled]: disabled,
    [styles.hasError]: hasError,
    [className]: className,
  });

  return (
    <label className={classes} htmlFor={args.htmlFor} {...args}>
      {children}
    </label>
  );
};

export default Label;
