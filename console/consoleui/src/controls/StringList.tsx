import clsx from 'clsx';
import React from 'react';

import {
  IconButton,
  IconDeleteBin,
  TextInput,
} from '@userclouds/ui-component-lib';
import Styles from './StringList.module.css';

type StringRowProps = {
  value: string;
  index: number;
  listID: string;
  inputType: string;
  onValueChange: (index: number, value: string) => void;
  onDelete: (val: string) => void;
};

const StringRow = ({
  value,
  index,
  listID,
  inputType,
  onValueChange,
  onDelete,
}: StringRowProps) => {
  return (
    <li className={Styles.row}>
      <TextInput
        id={`${listID}_${index}`}
        value={value}
        type={inputType}
        onChange={(e: React.ChangeEvent) => {
          onValueChange(index, (e.target as HTMLInputElement).value);
        }}
      />
      <div className={Styles.delete}>
        <IconButton
          icon={<IconDeleteBin />}
          onClick={() => onDelete(value)}
          title="Delete Element"
        >
          x
        </IconButton>
      </div>
    </li>
  );
};

type StringListProps = {
  strings: string[];
  id: string;
  inputType?: string;
  onValueChange: (index: number, value: string) => void;
  onDeleteRow: (val: string) => void;
};

const StringList = ({
  strings,
  id,
  inputType = 'text',
  onValueChange,
  onDeleteRow,
}: StringListProps) => {
  return (
    <ul id={id} className={clsx(Styles.stringtable, 'editableStringList')}>
      {strings.map((str, i) => (
        <StringRow
          // Disabling lint warning here because there is no innate unique key for each URL,
          // and even using the string as a key raises errors if the user has a duplicate.
          // eslint-disable-next-line react/no-array-index-key
          key={i}
          index={i}
          listID={id}
          value={str}
          inputType={inputType}
          onValueChange={onValueChange}
          onDelete={onDeleteRow}
        />
      ))}
    </ul>
  );
};

export default StringList;
