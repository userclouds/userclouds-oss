import React from 'react';
import { connect } from 'react-redux';
import {
  ButtonGroup,
  Button,
  IconArrowLeft,
  IconArrowRight,
} from '@userclouds/ui-component-lib';
import { RootState } from '../store';
import Link from './Link';
import styles from './Pagination.module.css';

type PaginationProps = {
  location: URL;
  next: string | undefined;
  prev: string | undefined;
  isLoading: boolean;
  updateURL?: boolean;
  // use prefix when there are multiple paginated lists on the same page
  prefix?: string;
  onNextClick?: React.MouseEventHandler<HTMLAnchorElement>;
  onPrevClick?: React.MouseEventHandler<HTMLAnchorElement>;
};

const Pagination = ({
  location,
  next,
  prev,
  isLoading,
  updateURL = true,
  prefix = '',
  onNextClick,
  onPrevClick,
}: PaginationProps) => {
  const { pathname, search } = location;
  const searchParams = new URLSearchParams(search);
  const hashParams = new URLSearchParams(location.hash.substring(1));
  if (!updateURL) {
    searchParams.delete(`${prefix}starting_after`);
    searchParams.delete(`${prefix}ending_before`);
  }

  const prevSearchParams = new URLSearchParams(updateURL ? search : hashParams);
  if (prev) {
    prevSearchParams.set(`${prefix}ending_before`, prev);
    prevSearchParams.delete(`${prefix}starting_after`);
  } else {
    prevSearchParams.delete(`${prefix}ending_before`);
  }

  const nextSearchParams = new URLSearchParams(updateURL ? search : hashParams);
  if (next) {
    nextSearchParams.set(`${prefix}starting_after`, next);
    nextSearchParams.delete(`${prefix}ending_before`);
  } else {
    nextSearchParams.delete(`${prefix}starting_after`);
  }
  return (
    <ButtonGroup justify="right" className={styles.root}>
      <Button
        disabled={!prev}
        isLoading={isLoading}
        theme="ghost"
        size="pagination"
        id="prev"
        title="View previous page"
      >
        {prev ? (
          <Link
            href={`${pathname}${
              (updateURL ? '?' : search + '#') + prevSearchParams
            }`}
            title="View previous page"
            applyStyles={false}
            onClick={onNextClick}
          >
            <IconArrowLeft />
          </Link>
        ) : (
          <IconArrowLeft />
        )}
      </Button>
      <Button
        disabled={!next || next === 'end'}
        isLoading={isLoading}
        theme="ghost"
        size="pagination"
        id="next"
        title="View next page"
      >
        {next && next !== 'end' ? (
          <Link
            href={`${pathname}${
              (updateURL ? '?' : search + '#') + nextSearchParams
            }`}
            title="View next page"
            applyStyles={false}
            onClick={onPrevClick}
          >
            <IconArrowRight />
          </Link>
        ) : (
          <IconArrowRight />
        )}
      </Button>
    </ButtonGroup>
  );
};

export default connect((state: RootState) => ({
  location: state.location,
}))(Pagination);
