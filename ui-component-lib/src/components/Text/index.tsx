import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

export interface TextProps extends React.ComponentProps<'p'> {
  /** Text to render */
  children?: React.ReactNode;
  /** Custom classes */
  className?: string;
  /** Size level of the text */
  size?: 1 | 2;
  /** Element type to use: p, span, etc */
  elementName?: keyof React.ReactHTML;
  /** A selector to hook into the React component */
  id?: string; // eslint-disable-line react/no-unused-prop-types
  /** Render text in monospace font */
  monospace?: boolean;
  /** Prevent line break, useful in tables */
  noWrap?: boolean;
}

export const ErrorText: React.FC<TextProps> = ({
  children,
  className,
  size = 1,
  elementName = 'p',
  monospace = false,
  noWrap = false,
}) => {
  const props = {
    className: clsx(
      styles.root,
      styles.error,
      styles[`text${size}`],
      { [styles.monospace]: monospace },
      { [styles.noWrap]: noWrap },
      className
    ),
  };

  return React.createElement(elementName, props, children);
};

export const SuccessText: React.FC<TextProps> = ({
  children,
  className,
  size = 1,
  elementName = 'p',
  monospace = false,
  noWrap = false,
}) => {
  const props = {
    className: clsx(
      styles.root,
      styles.success,
      styles[`text${size}`],
      { [styles.monospace]: monospace },
      { [styles.noWrap]: noWrap },
      className
    ),
  };

  return React.createElement(elementName, props, children);
};

/** Text styles. Defaults to a `<p>` but accepts any valid html tag, often `<span>` or `<b>`. */
const Text: React.FC<TextProps> = ({
  children,
  className,
  size = 1,
  elementName = 'p',
  monospace = false,
  noWrap = false,
}) => {
  const props = {
    className: clsx(
      styles.root,
      styles[`text${size}`],
      { [styles.monospace]: monospace },
      { [styles.noWrap]: noWrap },
      className
    ),
  };

  return React.createElement(elementName, props, children);
};

export default Text;
