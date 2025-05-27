import React from 'react';
import clsx from 'clsx';
import styles from './index.module.scss';

export interface InputGroupProps {
  /** Input components to render in a vertical space. */
  children?: React.ReactNode;
}

/** Provides consistent spacing between any group of inputs: checkbox, text, or radio. */

export default function InputGroup({
  children = null,
}: InputGroupProps): JSX.Element {
  return (
    <div
      className={clsx({
        [styles.root]: true,
      })}
    >
      {React.Children.map(children, (child) => child)}
    </div>
  );
}
