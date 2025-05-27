import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

export interface FormNoteProps {
  /** Text of note */
  children: React.ReactNode;
  /** If input it's associated with is in error */
  hasError?: boolean;
  /** Overriding classes */
  className?: string;
}

/** This goes below form inputs — checkbox, radio, select, text, textarea — to provide more context. */

const FormNote: React.FC<FormNoteProps> = ({
  children,
  className,
  hasError = false,
}) => {
  const classes = clsx({
    [styles.root]: true,
    [styles.hasError]: hasError,
    [className]: className,
  });

  return <div className={classes}>{children}</div>;
};

export default FormNote;
