import clsx from 'clsx';
import React, { useState } from 'react';
import IconButton from '../IconButton';
import { IconArrowDown, IconArrowUp } from '../../icons';

import styles from './index.module.scss';

/** Table */

export interface TableProps extends React.ComponentProps<'table'> {
  /** Spacing inside table cells */
  spacing?: 'default' | 'tight' | 'packed' | 'nowrap';
  /** Disables lines between table rows, useful for console output */
  hasLines?: boolean;
  /** If the table layout should change for mobile */
  isResponsive?: boolean;
}

/**
 * Basic table layout. Each component will pass through any valid table attributes.
 * For example, `align` and `width` can be used on table heads and cells.
 *
 * Tables are responsive by default. Use a `data-title` value for each cell that matches
 * its column title. Responsiveness can be disabled by setting `isResponsive={false}`
 */

const Table: React.FC<TableProps> = ({
  children,
  className,
  spacing = 'default',
  hasLines = true,
  isResponsive = true,
  ...attrs
}) => {
  const classes = clsx({
    [styles.table]: true,
    [styles.hasLines]: hasLines,
    [styles.isResponsive]: isResponsive,
    [styles[spacing]]: spacing,
    [className]: className,
  });

  return (
    <table className={classes} {...attrs}>
      {children}
    </table>
  );
};

/** Table head */

export interface TableHeadProps extends React.ComponentProps<'thead'> {
  /** floating head */
  floating?: boolean;
}

const TableHead: React.FC<TableHeadProps> = ({
  children,
  className,
  floating = false,
  ...attrs
}) => {
  const classes = clsx({
    [styles.thead]: true,
    [className]: className,
    [styles.theadfloating]: floating,
  });

  return (
    <thead className={classes} {...attrs}>
      {children}
    </thead>
  );
};

const TableFoot = ({
  children,
  className,
  ...attrs
}: React.ComponentProps<'tfoot'>) => {
  const classes = clsx({
    [styles.thead]: true,
    [className]: className,
  });

  return (
    <tfoot className={classes} {...attrs}>
      {children}
    </tfoot>
  );
};

/** Table row head */
interface TableRowHeadProps extends React.ComponentProps<'th'> {
  sort?: string;
}

const TableRowHead: React.FC<TableRowHeadProps> = ({
  sort,
  children,
  className,
  ...attrs
}) => {
  const classes = clsx({
    [styles.th]: true,
    [className]: className,
    [styles[sort]]: true,
  });

  return (
    <th className={classes} {...attrs}>
      {children}
    </th>
  );
};

/** Table body */

type TableBodyProps = React.ComponentProps<'tbody'>;

const TableBody: React.FC<TableBodyProps> = ({ children, className }) => {
  const classes = clsx({
    [styles.tbody]: true,
    [className]: className,
  });

  return <tbody className={classes}>{children}</tbody>;
};

const TableCell = ({
  children,
  className,
  ...attrs
}: React.TdHTMLAttributes<HTMLTableCellElement> & { className?: string }) => {
  const classes = clsx({
    [styles.td]: true,
    [className]: className,
  });

  return (
    <td className={classes} {...attrs}>
      {children}
    </td>
  );
};

/** Table row */

interface TableRowProps extends React.ComponentProps<'tr'> {
  /** If the table row has content that it can show on expansion. */
  isExtensible?: boolean;
  /** The number of columns the row should occupy when expanded. */
  columns?: number;
  /** The number of empty lead columns that should  */
  leadColumns?: number;
  /** The content that the row shows by placing in TableCells when expanded. */
  expandedContent?: JSX.Element | JSX.Element[];
}

const TableRow: React.FC<TableRowProps> = ({
  children,
  className,
  isExtensible = false,
  columns,
  leadColumns,
  expandedContent,
  ...attrs
}) => {
  const [closed, setClosed] = useState(true);

  const classes = clsx({
    [styles.tr]: closed,
    [styles.trextended]: !closed,
    [styles.trextendedline]: !closed && !expandedContent,
    [className]: className,
  });

  return (
    <>
      <tr className={classes} {...attrs} data-extended={!closed}>
        {children}
        {isExtensible && (
          <TableCell className={styles.expansionIndicator}>
            <IconButton
              size="tiny"
              icon={
                closed ? (
                  <IconArrowDown className={styles.icon} />
                ) : (
                  <IconArrowUp className={styles.icon} />
                )
              }
              onClick={() => setClosed(!closed)}
              title="Set Closed"
              aria-label="Set Closed"
            />
          </TableCell>
        )}
      </tr>
      {isExtensible && !closed && expandedContent && (
        <tr className={styles.trextendedcontent}>
          {leadColumns && <TableCell colSpan={leadColumns} />}
          <TableCell colSpan={columns}>{expandedContent}</TableCell>
        </tr>
      )}
    </>
  );
};

interface TableTitleProps extends React.ComponentProps<'h3'> {
  mainText: string;
  subText?: string;
}

const TableTitle = ({
  mainText,
  subText = '',
  className = '',
  ...attrs
}: TableTitleProps) => {
  return (
    <h3 {...attrs} className={clsx(className, styles.tableTitle)}>
      {mainText}
      {subText && <span className={styles.subText}> {subText}</span>}
    </h3>
  );
};

export {
  Table,
  TableHead,
  TableBody,
  TableFoot,
  TableRowHead,
  TableRow,
  TableCell,
  TableTitle,
};
