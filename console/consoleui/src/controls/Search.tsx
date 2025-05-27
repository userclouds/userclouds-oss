import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  AppliedFilter,
  Select,
  TextInput,
} from '@userclouds/ui-component-lib';

import { redirect } from '../routing';
import { RootState, AppDispatch } from '../store';
import { Filter } from '../models/authz/SearchFilters';
import {
  clearSearchFilter,
  addFilterToSearchParams,
  getSearchParamsArray,
  getHumanReadableColumnName,
  getFormattedValue,
  mergeFilter,
  getPatternFromColumnName,
  DATE_COLUMNS,
} from './SearchHelper';
import Link from './Link';
import styles from './Search.module.css';

type SearchProps = {
  columns: string[];
  prefix: string;
  searchFilter: Filter;
  changeSearchFilter: Function;
  updateURL?: boolean;
  location: URL;
  searchParams: URLSearchParams;
  id: string;
  dialog?: boolean;
  dispatch: AppDispatch;
};

const Search = ({
  columns,
  prefix,
  searchFilter,
  changeSearchFilter,
  updateURL = true,
  location,
  searchParams,
  id,
  dialog = false,
  dispatch,
}: SearchProps) => {
  const hashParams = new URLSearchParams(location.hash.substring(1));
  const searchFilters = getSearchParamsArray(
    (updateURL ? searchParams : hashParams).get(prefix + 'filter')
  );
  const clearFiltersParams = new URLSearchParams(searchParams);
  clearFiltersParams.delete(prefix + 'filter');
  const clearFiltersURL = updateURL
    ? `${location.pathname}?${clearFiltersParams.toString()}`
    : `${location.pathname}?${searchParams.toString()}#`;

  useEffect(() => {
    if (!searchFilter || !searchFilter.columnName) {
      dispatch(() =>
        changeSearchFilter({
          columnName: columns[0],
          operator: 'EQ',
          value: '',
          operator2: 'LE',
          value2: '',
        })
      );
    }
  }, [
    columns,
    searchFilter,
    changeSearchFilter,
    searchParams,
    location.hash,
    dispatch,
  ]);

  return (
    searchFilter && (
      <form
        id={id + 'searchBar'}
        className={styles.root}
        onSubmit={(e: React.FormEvent) => {
          e.preventDefault();
          if (updateURL) {
            const newSearchParams = addFilterToSearchParams(
              prefix,
              searchParams,
              searchFilter
            );
            redirect(`${location.pathname}?${newSearchParams.toString()}`);
          } else {
            const newFilterString =
              prefix +
              'filter=' +
              encodeURIComponent(
                mergeFilter(searchFilter, hashParams.get(prefix + 'filter'))
              );
            redirect(
              `${
                location.pathname
              }?${searchParams.toString()}#${newFilterString}`
            );
          }
        }}
      >
        <Select
          name="columns"
          id={id + 'columns'}
          items={columns}
          value={searchFilter.columnName}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            dispatch(() => {
              changeSearchFilter({ columnName: e.currentTarget.value });
            });
          }}
          themeSize="small"
          className={styles.select}
        >
          {columns.map((t) => (
            <option key={t} value={t}>
              {'Filter by ' + getHumanReadableColumnName(t)}
            </option>
          ))}
        </Select>
        <TextInput
          name="search_value_1"
          id="search_value_1"
          placeholder={
            DATE_COLUMNS.includes(searchFilter.columnName)
              ? 'start YYYY/MM/DD HH:MM:SS.SSSS'
              : 'Enter ' + getHumanReadableColumnName(searchFilter.columnName)
          }
          value={searchFilter.value?.replaceAll('%', '')}
          pattern={getPatternFromColumnName(searchFilter.columnName)}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            dispatch(() => {
              changeSearchFilter({ value: e.currentTarget.value });
            });
          }}
          size="small"
        />
        {DATE_COLUMNS.includes(searchFilter.columnName) && (
          <TextInput
            name="search_value_2"
            id="search_value_2"
            placeholder="end YYYY/MM/DD HH:MM:SS.SSSS"
            pattern={getPatternFromColumnName(searchFilter.columnName)}
            value={searchFilter.value2}
            onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
              dispatch(() => {
                changeSearchFilter({ value2: e.currentTarget.value });
              });
            }}
            size="small"
          />
        )}
        <ButtonGroup className={styles.buttonGroup}>
          <Button
            type="submit"
            disabled={
              !searchFilter || (!searchFilter.value && !searchFilter.value2)
            }
            theme="secondary"
            size="small"
          >
            Add Filter
          </Button>

          <Button
            size="small"
            theme="secondary"
            disabled={!searchFilters || !(searchFilters.length > 0)}
          >
            <Link
              href={clearFiltersURL}
              title="Remove currently applied filters"
              applyStyles={false}
            >
              Clear Filters
            </Link>
          </Button>
        </ButtonGroup>
        {searchFilters && searchFilters.length > 0 && (
          <ul
            id="searchBarFilters"
            className={
              !dialog ? styles.appliedFiltersNavMenu : styles.appliedFilters
            }
          >
            {searchFilters.map((f) => (
              <li key={f.columnName + f.operator + f.value}>
                <AppliedFilter
                  isDismissable
                  source={f.columnName}
                  text={
                    f.operator + ' ' + getFormattedValue(f.columnName, f.value)
                  }
                  key={f.columnName + f.operator + f.value}
                  onDismissClick={() => {
                    const newSearchParams = clearSearchFilter(
                      f.columnName,
                      f.operator,
                      f.value,
                      prefix,
                      updateURL ? searchParams : hashParams
                    );
                    if (updateURL) {
                      if (newSearchParams) {
                        redirect(
                          `${location.pathname}?${newSearchParams.toString()}`
                        );
                      }
                    } else {
                      redirect(
                        `${
                          location.pathname
                        }?${searchParams.toString()}#${newSearchParams}`
                      );
                    }
                  }}
                />
              </li>
            ))}
          </ul>
        )}
      </form>
    )
  );
};

export default connect((state: RootState) => ({
  location: state.location,
  searchParams: state.query,
}))(Search);
