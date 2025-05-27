import React from 'react';

import styles from './index.module.scss';

export interface CodeBlockProps {
  /** Code to display */
  children: React.ReactNode;
}

/**
 * Blocks of code.
 */

const CodeBlock: React.FC<CodeBlockProps> = ({ children }) => (
  <div className={styles.root}>{children}</div>
);

export default CodeBlock;
