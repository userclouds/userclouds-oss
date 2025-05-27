import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

/** Radio control for a collection of mutually exclusive items. */
const Radio: React.FC<React.ComponentProps<'input'>> = ({
  children,
  checked = false,
  disabled = false,
  ...args
}) => {
  const labelClasses = clsx({
    [styles.label]: true,
    [styles.disabled]: disabled,
    [styles.isChecked]: checked,
  });

  return (
    <label className={labelClasses}>
      <input className={styles.input} type="radio" {...args} />
      <div className={styles.inputImage} />
      <div className={styles.text}>{children}</div>
    </label>
  );
};

export default Radio;
