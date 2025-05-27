import Button from '../Button';
import InputReadOnly from '../InputReadOnly';
import Label from '../Label';
import Tag from '../Tag';

import styles from './index.module.scss';
import inputStyles from '../TextInput/index.module.scss';

type DataListOption = {
  label: string;
  value: string;
};
const DataListTagger = ({
  inputName,
  readOnly,
  label,
  menuItems,
  selectedItems,
  addHandler,
  removeHandler,
}: {
  inputName: string;
  readOnly: boolean;
  label: string;
  menuItems: DataListOption[];
  selectedItems: string[];
  addHandler: (val: string) => void;
  removeHandler: (val: string[]) => void;
}) => (
  <div className={styles.datalisttagger}>
    <Label>
      {label}
      {!readOnly && (
        <fieldset>
          <input
            id={inputName}
            name={inputName}
            type="text"
            list={`${inputName}-datalist`}
            className={inputStyles.input}
          />
          <Button
            theme="secondary"
            size="small"
            onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
              e.preventDefault();

              const textInput = document.getElementById(
                inputName
              ) as HTMLInputElement;

              addHandler(textInput.value);
              textInput.value = '';
            }}
          >
            Add
          </Button>
          <datalist id={`${inputName}-datalist`}>
            {menuItems
              .filter(({ label: l }) => !selectedItems.includes(l))
              .map(({ label: l, value }) => (
                <option value={value} key={value}>
                  {l}
                </option>
              ))}
          </datalist>
        </fieldset>
      )}
    </Label>
    <InputReadOnly className="ml-6">
      {selectedItems.length
        ? selectedItems.map((item: string) => (
            <>
              <Tag
                tag={item}
                key={item}
                isRemovable={!readOnly}
                onDismissClick={(e: React.MouseEvent) => {
                  e.preventDefault();

                  const newList = [...selectedItems].filter(
                    (si) => item !== si
                  );

                  removeHandler(newList);
                }}
              />{' '}
            </>
          ))
        : '-'}
    </InputReadOnly>
  </div>
);

export default DataListTagger;
