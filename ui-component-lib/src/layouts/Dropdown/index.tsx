import React from 'react';
import clsx from 'clsx';
import styles from './index.module.scss';

export interface DropdownProps extends React.ComponentProps<'div'> {
  /** `OverlaySection` components to render. */
  children: React.ReactNode | React.ReactNode[];
  /** direction to render */
  direction?: 'left' | 'right';
}

interface DropdownSectionProps extends React.ComponentProps<'div'> {
  /** `Button` components to render. */
  children: React.ReactNode | React.ReactNode[];
}

const DropdownSection = ({ children }: DropdownSectionProps) => (
  <div className={styles.section}>
    {React.Children.map(children, (child) => child)}
  </div>
);

const Dropdown = ({ children, direction = 'left' }: DropdownProps) => {
  const classes = clsx({
    [styles.root]: true,
    [styles.sectionleft]: direction === 'left',
    [styles.sectionrightc]: direction === 'right',
  });
  return (
    <div className={classes}>
      <DropdownSection>{children}</DropdownSection>
    </div>
  );
};

interface DropdownButtonProps extends React.ComponentProps<'button'> {
  /** Text of button. */
  children: React.ReactNode;
}

const DropdownButton = ({ children, ...rest }: DropdownButtonProps) => (
  <button
    className={styles.button}
    {...rest}
    onMouseDown={(e: React.MouseEvent) => {
      e.preventDefault();
    }}
  >
    {children}
  </button>
);

export { Dropdown, DropdownSection, DropdownButton };
