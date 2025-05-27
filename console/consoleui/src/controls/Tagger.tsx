import { useEffect, useState } from 'react';
import { connect } from 'react-redux';

import { v4 as uuidv4 } from 'uuid';

import {
  Dropdown,
  DropdownSection,
  DropdownButton,
  InlineNotification,
  Tag,
  Text,
  TextInput,
} from '@userclouds/ui-component-lib';

import {
  getFilteredTags,
  getFilteredTagsAndSearch,
  getTagNameFromID,
  TagModel,
} from '../models/Tag';
import { AppDispatch, RootState } from '../store';
import { fetchTagsSuccess } from '../actions/tags';

const handleTextChange =
  (
    objectTags: string[],
    systemTags: TagModel[],
    setInputValue: Function,
    setFilteredTags: Function
  ) =>
  (e: React.ChangeEvent<HTMLInputElement>) => {
    setInputValue(e.target.value);
    let tags: Array<TagModel> = [];
    if (objectTags) {
      tags = getFilteredTags(objectTags, systemTags);
    }
    setFilteredTags(tags.filter((tag) => tag.name.includes(e.target.value)));
  };

const handleOnCreateInputTag =
  (
    objectTags: string[],
    systemTags: TagModel[],
    inputValue: string,
    setFilteredTags: Function,
    setObjectTags: Function
  ) =>
  (dispatch: AppDispatch) => {
    const tag = { id: uuidv4(), name: inputValue };
    // TODO on BE create
    // createTag(selectedID, {id:uuidv4(), name: inputValue})
    dispatch(fetchTagsSuccess([...systemTags, tag]));
    setFilteredTags(getFilteredTags(objectTags, systemTags));
    dispatch(setObjectTags({ tag_ids: [...objectTags, tag.id] }));
  };

const handleAddInputTag =
  (
    objectTags: string[],
    systemTags: TagModel[],
    tag: TagModel,
    setFilteredTags: Function,
    setObjectTags: Function
  ) =>
  (dispatch: AppDispatch) => {
    dispatch(setObjectTags({ tag_ids: [...objectTags, tag.id] }));
    setFilteredTags(getFilteredTags(objectTags, systemTags));
  };

const handleDismissTag =
  (
    objectTags: string[],
    systemTags: TagModel[],
    dismissedTagID: string,
    setFilteredTags: Function,
    setObjectTags: Function
  ) =>
  (dispatch: AppDispatch) => {
    const updatedTags = objectTags.filter((tag) => tag !== dismissedTagID);
    setFilteredTags(getFilteredTags(updatedTags, systemTags));
    dispatch(setObjectTags({ tag_ids: updatedTags }));
  };

const OptionText = ({ name, bold }: { name: string; bold: string }) => {
  if (!bold) {
    return <Text>{name}</Text>;
  }
  if (!name.includes(bold)) {
    return <></>;
  }
  const fields = name.split(bold);
  return (
    <span>
      {fields.map((item, index) => (
        <>
          {item}
          {index !== fields.length - 1 && <b>{bold}</b>}
        </>
      ))}
    </span>
  );
};

type TaggerComponentProps = {
  selectedTenantID: string | undefined;
  systemTags: TagModel[] | undefined;
  objectTags: string[];
  setObjectTags: Function;
  dispatch: AppDispatch;
};

const TaggerComponent = ({
  selectedTenantID,
  systemTags,
  objectTags,
  setObjectTags,
  dispatch,
}: TaggerComponentProps) => {
  useEffect(() => {
    if (selectedTenantID) {
      if (!systemTags) {
        // TODO once back end has fetchSystemTags(selectedTenantID);
      }
    }
  }, [selectedTenantID, systemTags, dispatch]);

  const [inputValue, setInputValue] = useState('');
  const [showDropdown, setShowDropdown] = useState(false);
  const [filteredTags, setFilteredTags] = useState<TagModel[]>([]);

  if (!systemTags) {
    return (
      <InlineNotification theme="alert">
        Unable to fetch tags.
      </InlineNotification>
    );
  }

  return (
    <div className="bg-white" style={{ width: '100%', height: '50px' }}>
      <TextInput
        style={{ width: '100%' }}
        size="large"
        onBlur={() => setShowDropdown(false)}
        onFocus={() => {
          setShowDropdown(true);
          setFilteredTags(systemTags);
        }}
        innerLeft={
          objectTags.length ? (
            <div className="flex gap-2">
              {objectTags.map((tag) => (
                <Tag
                  style={{ margin: '5px' }}
                  tag={getTagNameFromID(systemTags, tag)}
                  isRemovable
                  onClick={() =>
                    dispatch(
                      handleDismissTag(
                        objectTags,
                        systemTags,
                        tag,
                        setFilteredTags,
                        setObjectTags
                      )
                    )
                  }
                  key={tag}
                />
              ))}
            </div>
          ) : null
        }
        value={inputValue}
        onChange={handleTextChange(
          objectTags,
          systemTags,
          setInputValue,
          setFilteredTags
        )}
      />

      {showDropdown && (
        <Dropdown>
          <DropdownSection>
            {filteredTags.length > 0 &&
              getFilteredTagsAndSearch(objectTags, systemTags, inputValue).map(
                (tag) => (
                  <DropdownButton
                    onClick={() => {
                      dispatch(
                        handleAddInputTag(
                          objectTags,
                          systemTags,
                          tag,
                          setFilteredTags,
                          setObjectTags
                        )
                      );
                      setInputValue('');
                    }}
                    key={tag.id}
                  >
                    <OptionText name={tag.name} bold={inputValue} />
                  </DropdownButton>
                )
              )}
            <DropdownButton
              disabled={
                inputValue.length === 0 ||
                filteredTags.find((tag) => tag.name === inputValue) ||
                systemTags.find((tag) => tag.name === inputValue)
              }
              onMouseDown={(e: React.MouseEvent) => {
                e.preventDefault();
              }}
              onClick={() => {
                dispatch(
                  handleOnCreateInputTag(
                    objectTags,
                    systemTags,
                    inputValue,
                    setFilteredTags,
                    setObjectTags
                  )
                );
                setInputValue('');
              }}
            >
              Add <b>{inputValue}</b>
            </DropdownButton>
          </DropdownSection>
        </Dropdown>
      )}
    </div>
  );
};

const Tagger = connect((state: RootState) => {
  return {
    systemTags: state.tags,
    selectedTenantID: state.selectedTenantID,
  };
})(TaggerComponent);

export default Tagger;
