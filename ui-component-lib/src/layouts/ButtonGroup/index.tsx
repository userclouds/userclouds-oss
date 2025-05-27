import React from 'react';
import clsx from 'clsx';
import styles from './index.module.scss';

export interface ButtonGroupProps {
  /** `Button` components to render. */
  children?: React.ReactNode;
  /** Controls the horizontal alignment of buttons within the container. */
  justify?: 'center' | 'left' | 'right' | 'between';
  className?: string;
}

export default function ButtonGroup({
  children = null,
  justify = 'left',
  className = '',
}: ButtonGroupProps): JSX.Element {
  return (
    <div
      className={`${clsx({
        [styles.root]: true,
        [styles[justify]]: justify,
      })} ${className}`}
    >
      {React.Children.map(children, (child) => child)}
    </div>
  );
}
