import React, { forwardRef, useRef } from 'react';
import clsx from 'clsx';

import Heading from '../Heading';
import Text from '../Text';
import IconButton from '../IconButton';
import { IconClose } from '../../icons';
import styles from './index.module.scss';
import cardStyles from '../../layouts/Card/index.module.scss';
import mergeRefs from '../../utils/mergeRefs';

export type DialogProps = React.ComponentProps<'dialog'> & {
  title: string;
  description: string;
  fullPage: boolean;
  isDismissable: boolean;
  onClose: () => void;
  open: boolean;
  children: React.ReactNode;
  className?: string;
};

export type DialogBodyProps = React.ComponentProps<'div'>;

export type DialogFooterProps = React.ComponentProps<'footer'>;

/**
 * Commonly used with the input components, like `<TextInput>` or `<Select>`. Clicking
 * on the label will place focus its input by using the `htmlFor` attr and matching that
 * to the `id` of the input.
 *
 * Note that its disabled and error states should be tied with the state of the input.
 */

const Dialog = forwardRef<HTMLDialogElement, DialogProps>(
  (
    {
      children,
      className,
      description = '',
      fullPage = false,
      isDismissable = true,
      onClose,
      open,
      title = '',
      ...args
    },
    ref
  ) => {
    const localRef = useRef<HTMLDialogElement>(null);

    const dialogRef = mergeRefs(localRef, ref as React.Ref<HTMLDialogElement>);

    const classes = clsx({
      [cardStyles.card]: true,
      [styles.root]: true,
      [styles.fullPage]: fullPage,
      [className]: className,
    });

    return (
      <dialog
        className={classes}
        open={open}
        {...args}
        ref={dialogRef}
        onClose={onClose}
      >
        {title && (
          <Heading size={2} headingLevel={1} className={styles.heading}>
            {title}
          </Heading>
        )}
        {description && (
          <Text size={1} className={styles.description}>
            {description}
          </Text>
        )}
        {isDismissable && (
          <IconButton
            icon={<IconClose size="small" />}
            title="Close dialog"
            className={styles.closeButton}
            aria-label="Close dialog"
            onClick={() => {
              if (localRef.current) {
                localRef.current.close();
              }
            }}
          />
        )}
        {children}
      </dialog>
    );
  }
);

export const DialogBody: React.FC<DialogBodyProps> = ({
  children,
  className,
}) => {
  const classes = clsx({
    [styles.dialogBody]: true,
    [className]: className,
  });

  return <div className={classes}>{children}</div>;
};

export const DialogFooter: React.FC<DialogFooterProps> = ({
  children,
  className,
}) => {
  const classes = clsx({
    [styles.footer]: true,
    [className]: className,
  });

  return <footer className={classes}>{children}</footer>;
};

export default Dialog;
