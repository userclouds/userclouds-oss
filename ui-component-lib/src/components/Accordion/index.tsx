import React from 'react';
import styles from './index.module.scss';
import { IconArrowDown } from '../../icons';

export interface AccordionProps {
  /** Content of the Accordion */
  children: React.ReactNode;
}

/** This component styles the `<details>` disclosure element.  */

export const Accordion: React.FC<AccordionProps> = ({ children }) => (
  <div className={styles.accordion}>
    {React.Children.map(children, (child) => child)}
  </div>
);

interface AccordionItemProps {
  /** Content of the AccordionItem */
  children: React.ReactNode;
  /** Title of the AccordionItem */
  title: string;
  /** If AccordionItem should start open */
  isOpen?: boolean;
}

export const AccordionItem: React.FC<AccordionItemProps> = ({
  title,
  children,
  isOpen,
}) => (
  <details className={styles.accordionItem} open={isOpen}>
    <summary className={styles.summary}>
      <div className={styles.title}>{title}</div>
      <IconArrowDown className={styles.icon} />
    </summary>
    <div className={styles.content}>{children}</div>
  </details>
);
