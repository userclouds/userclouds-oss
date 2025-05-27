import React from 'react';
import Styles from './DropDown.module.css';

enum DropDownOverflow {
  Left,
  Right,
}
type DropDownProps = React.HTMLAttributes<HTMLDivElement> & {
  active?: boolean;
  overflow?: DropDownOverflow;
  children?: React.ReactNode;
};

const DropDown = ({
  active,
  overflow = DropDownOverflow.Right,
  children,
  ...rest
}: DropDownProps) => {
  let dropDownClasses = Styles.dropdown;
  if (rest?.className) {
    dropDownClasses += ` ${rest.className}`;
  }
  if (!active) {
    dropDownClasses += ` ${Styles.hidden}`;
  }

  if (overflow === DropDownOverflow.Left) {
    dropDownClasses += ` ${Styles.overflowleft}`;
  } else {
    dropDownClasses += ` ${Styles.overflowright}`;
  }

  return <div className={dropDownClasses}>{children}</div>;
};

export default DropDown;
export { DropDownOverflow };
