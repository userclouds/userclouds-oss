import React, { useState } from 'react';
import clsx from 'clsx';

import ToolTip from '../../components/Tooltip';
import Heading from '../../components/Heading';
import Text from '../../components/Text';
import IconButton from '../../components/IconButton';
import { IconArrowUp, IconLock2 } from '../../icons';

import styles from './index.module.scss';

export interface CardProps {
  /** Content of the Card */
  children?: JSX.Element | JSX.Element[];
  /** `className` for overrides if needed, best to avoid */
  className?: string;
  /** If the card is collapsible (default is true) */
  collapsible?: boolean;
  /** If it's in an error state */
  hasError?: boolean;
  /** If it's in a disabled state */
  isDirty?: boolean;
  /** If card should render in closed state */
  isClosed?: boolean;
  /** Title of card */
  title?: string;
  /** Short sentence describing purpose of card */
  description?: string | JSX.Element;
  /** If card should render in locked state */
  lockedMessage?: string;
  /** HTML ID attribute to be placed on <section> tag */
  id?: string;
  /** Nav menu card per new menu and list designs */
  listview?: boolean;
  /** detail card per new detail/create/edit page designs */
  detailview?: boolean;
}

/** A wrapper to hold content blocks. Spacing between instances of
 * `<CardRow>` is set at 24px to provide
 * consistent visual separation between content blocks.
 */

export const Card: React.FC<CardProps> = ({
  children,
  className,
  collapsible = true,
  hasError = false,
  isDirty = false,
  isClosed = false,
  title,
  description,
  lockedMessage,
  id,
  listview = false,
  detailview = false,
}) => {
  const [closed, setClosed] = useState(isClosed);

  const classes = clsx({
    [styles.card]: !listview && !detailview,
    [styles.listcard]: listview,
    [styles.detailcard]: detailview,
    [styles.isDirty]: isDirty && !listview && !detailview,
    [styles.hasError]: hasError,
    [styles.isClosed]: closed,
    [styles.collapseHeaderSpacing]: !description && title,
    [className]: className,
  });

  const mainClasses = clsx({
    [styles.content]: true,
  });

  return (
    <section id={id}>
      {lockedMessage && (
        <div className={styles.isLocked}>
          <IconLock2 size="small" />
          <div className={styles.isLockedText}>{lockedMessage}</div>
        </div>
      )}
      <div className={classes}>
        {title && (
          <header className={styles.header}>
            {/* we need this wrapper div to ensure the block-level elements within it wrap */}
            <div>
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
            </div>
            {collapsible && (
              <IconButton
                icon={<IconArrowUp className={styles.icon} />}
                onClick={() => setClosed(!closed)}
                title="Set Closed"
                aria-label="Set Closed"
              />
            )}
          </header>
        )}
        {!closed && (
          <main className={mainClasses}>
            {React.Children.map(children, (child) => child)}
          </main>
        )}
      </div>
    </section>
  );
};

interface CardRowProps {
  /** Content of the Card */
  children?: JSX.Element | JSX.Element[];
  /** `className` for overrides if needed, best to avoid */
  className?: string;
  /** is the cardrow collapsible */
  collapsible?: boolean;
  /** If card should render in closed state */
  isClosed?: boolean;
  /** title of the card row */
  title?: string;
  /** Tool tip info */
  tooltip?: JSX.Element | JSX.Element[];
}

export const CardRow: React.FC<CardRowProps> = ({
  children,
  className,
  collapsible,
  isClosed = false,
  title,
  tooltip,
}) => {
  const [closed, setClosed] = useState(isClosed);
  const classes = clsx({
    [styles.row]: true,
    [className]: className,
    [styles.isClosed]: closed,
  });

  return collapsible ? (
    <div className={classes}>
      <div className={styles.cardrowtitle}>
        <h2>{title}</h2>
        {tooltip && <ToolTip direction="right">{tooltip}</ToolTip>}
        <IconButton
          icon={<IconArrowUp className={styles.icon} />}
          onClick={() => setClosed(!closed)}
          title="Set Closed"
          className={styles.iconButton}
          aria-label="Set Closed"
        />
      </div>
      {!closed && <div className={styles.detailrowcontent}>{children}</div>}
    </div>
  ) : (
    <div className={classes}>
      <h2 className={styles.cardrowtitle}>
        {title}
        {tooltip && <ToolTip direction="left">{tooltip}</ToolTip>}
      </h2>
      {children}
    </div>
  );
};

interface CardColumnsProps {
  /** Content of the Columns, a wrapper of Column */
  children?: React.ReactNode;
  /** `className` for overrides if needed, best to avoid */
  className?: string;
}

export const CardColumns: React.FC<CardColumnsProps> = ({
  children,
  className,
}) => {
  const classes = clsx({
    [styles.columns]: true,
    [className]: className,
  });

  return <div className={classes}>{children}</div>;
};

interface CardColumnProps {
  /** Content of the Column, commonly CardRow */
  children?: React.ReactNode;
  /** `className` for overrides if needed, best to avoid */
  className?: string;
}

export const CardColumn: React.FC<CardColumnProps> = ({
  children,
  className,
}) => {
  const classes = clsx({
    [styles.column]: true,
    [className]: className,
  });

  return <div className={classes}>{children}</div>;
};

interface CardFooterProps {
  /** Content of the Footer */
  children?: React.ReactNode;
  /** `className` for overrides if needed, best to avoid */
  className?: string;
}

export const CardFooter: React.FC<CardFooterProps> = ({
  children,
  className,
}) => {
  const classes = clsx({
    [styles.footer]: true,
    [className]: className,
  });

  return <footer className={classes}>{children}</footer>;
};
