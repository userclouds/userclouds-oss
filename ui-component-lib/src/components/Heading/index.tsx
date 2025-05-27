import React from 'react';
import clsx from 'clsx';

import styles from './index.module.scss';

export interface HeadingProps {
  /** Content to render */
  children?: React.ReactNode;
  /** Custom styling, avoid if possible */
  className?: string;
  /** Size level of the text */
  size: 1 | 2 | 3 | 'new';
  /** Heading element level (h1 to h6) to render. If omitted, a div is used. */
  headingLevel?: 1 | 2 | 3 | 4 | 5 | 6;
  /** A selector to hook into the React component */
  id?: string;
  /** Custom styling, avoid if possible */
  style?: React.CSSProperties;
}

/**
 * Set large titles, usually `h` elements. Will use `<div>` if not heading level is set.
 */
const Heading: React.FC<HeadingProps> = ({
  children,
  size,
  className,
  headingLevel,
  id,
  style,
}) => {
  const elementName = headingLevel ? `h${headingLevel}` : 'div';
  const props = {
    className: clsx(styles.root, styles[`heading${size}`], className),
    id,
    style,
  };

  return React.createElement(elementName, props, children);
};

export default Heading;
