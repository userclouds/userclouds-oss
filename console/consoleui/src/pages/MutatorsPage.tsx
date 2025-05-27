import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  IconFilter,
  IconUserReceived2,
  InlineNotification,
  Table,
  TableHead,
  TableBody,
  TableCell,
  TableRow,
  TableRowHead,
  Text,
  TextShortener,
  ToolTip,
  Tag,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import Mutator, { MUTATOR_COLUMNS, MUTATOR_PREFIX } from '../models/Mutator';
import { SelectedTenant } from '../models/Tenant';
import PaginatedResult from '../models/PaginatedResult';
import { Filter } from '../models/authz/SearchFilters';
import {
  bulkDeleteMutatorsOrAccessors,
  fetchMutators,
} from '../thunks/userstore';

import {
  changeMutatorSearchFilter,
  toggleMutatorForDelete,
} from '../actions/mutators';
import Link from '../controls/Link';
import { applySort, columnSortDirection } from '../controls/PaginationHelper';
import Pagination from '../controls/Pagination';
import Search from '../controls/Search';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';
import PageCommon from './PageCommon.module.css';
import Styles from './MutatorsPage.module.css';

const MutatorList = ({
  selectedTenant,
  mutators,
  isFetching,
  fetchError,
  deleteQueue,
  saveSuccess,
  saveErrors,
  query,
  mutatorSearchFilter,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  mutators: PaginatedResult<Mutator> | undefined;
  isFetching: boolean;
  fetchError: string;
  deleteQueue: Record<string, Mutator>;
  saveSuccess: string;
  saveErrors: string[];
  query: URLSearchParams;
  mutatorSearchFilter: Filter;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const deleteQueueLength = Object.keys(deleteQueue).length;

  const deletePrompt = `Are you sure you want to delete ${
    deleteQueueLength
  } mutator${deleteQueueLength === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="mutators"
          columns={MUTATOR_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeMutatorSearchFilter(filter));
          }}
          prefix={MUTATOR_PREFIX}
          searchFilter={mutatorSearchFilter}
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              {
                'Mutators are configurable APIs that allow a client to write data to the User Store. '
              }
              <a
                href="https://docs.userclouds.com/docs/mutators-write-apis"
                title="UserClouds documentation for mutators (write APIs)"
                target="new"
                className={PageCommon.link}
              >
                Learn more here.
              </a>
            </>
          </ToolTip>
        </div>

        {selectedTenant?.is_admin && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
          >
            <Link href={`/mutators/create${cleanQuery}`} applyStyles={false}>
              Create Mutator
            </Link>
          </Button>
        )}
      </div>

      <Card
        id="userstoreMutators"
        lockedMessage={
          !selectedTenant?.is_admin ? 'You do not have edit access' : ''
        }
        listview
      >
        {mutators ? (
          mutators.data && mutators.data.length ? (
            <>
              {saveSuccess && (
                <div className={PageCommon.tableNotification}>
                  <InlineNotification theme="success">
                    {saveSuccess}
                  </InlineNotification>
                </div>
              )}
              {!!saveErrors.length && (
                <div className={PageCommon.tableNotification}>
                  <InlineNotification theme="alert">
                    {saveErrors.length > 1
                      ? `Error deleting ${saveErrors.length} mutators`
                      : saveErrors[0]}
                  </InlineNotification>
                </div>
              )}
              <div className={PageCommon.listviewpaginationcontrols}>
                <div className={PageCommon.listviewpaginationcontrolsdelete}>
                  <DeleteWithConfirmationButton
                    id="deleteMutatorsButton"
                    message={deletePrompt}
                    onConfirmDelete={() => {
                      dispatch(
                        bulkDeleteMutatorsOrAccessors(
                          selectedTenant?.id || '',
                          deleteQueue,
                          'mutator'
                        )
                      );
                    }}
                    title="Delete Mutators"
                    disabled={deleteQueueLength < 1}
                  />
                </div>
                <Pagination
                  prev={mutators?.prev}
                  next={mutators?.next}
                  isLoading={isFetching}
                  prefix={MUTATOR_PREFIX}
                />
              </div>
              <Table
                spacing="nowrap"
                id="mutators"
                className={Styles.mutatorlisttable}
              >
                <TableHead floating>
                  <TableRow>
                    <TableRowHead className={Styles.checkboxCol}>
                      <Checkbox
                        checked={
                          Object.keys(deleteQueue).length ===
                          mutators.data.length
                        }
                        onChange={() => {
                          const shouldMarkForDelete =
                            !deleteQueue[mutators.data[0].id];
                          mutators.data.forEach((o) => {
                            if (shouldMarkForDelete && !deleteQueue[o.id]) {
                              dispatch(toggleMutatorForDelete(o));
                            } else if (
                              !shouldMarkForDelete &&
                              deleteQueue[o.id]
                            ) {
                              dispatch(toggleMutatorForDelete(o));
                            }
                          });
                        }}
                      />
                    </TableRowHead>

                    <TableRowHead
                      key="mutator_name"
                      sort={columnSortDirection(MUTATOR_PREFIX, query, 'name')}
                    >
                      <Link
                        href={'?' + applySort(MUTATOR_PREFIX, query, 'name')}
                        applyStyles={false}
                      >
                        Mutator Name
                      </Link>
                    </TableRowHead>
                    <TableRowHead key="mutator_table">Table</TableRowHead>
                    <TableRowHead key="mutator_columns">Columns</TableRowHead>
                    <TableRowHead
                      sort={columnSortDirection(MUTATOR_PREFIX, query, 'id')}
                      key="mutator_id"
                    >
                      <Link
                        href={'?' + applySort(MUTATOR_PREFIX, query, 'id')}
                        applyStyles={false}
                      >
                        ID
                      </Link>
                    </TableRowHead>
                    <TableRowHead key="mutator_version">Version</TableRowHead>
                    <TableRowHead key="delete_mutator" />
                    <TableRowHead className={Styles.chevronCol} />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {mutators.data.map((mutator) => {
                    const mutatorColumns = mutator.columns
                      .map((col) => col.name)
                      .join(', ');
                    return (
                      <TableRow
                        key={mutator.id}
                        isExtensible
                        className={
                          (deleteQueue[mutator.id]
                            ? PageCommon.queuedfordelete
                            : '') +
                          ' ' +
                          PageCommon.listviewtablerow
                        }
                      >
                        <TableCell>
                          <Checkbox
                            id={'delete' + mutator.id}
                            name="delete mutator"
                            checked={deleteQueue[mutator.id] ?? false}
                            onChange={() => {
                              dispatch(toggleMutatorForDelete(mutator));
                            }}
                          />
                        </TableCell>
                        <TableCell className={Styles.primaryCol}>
                          <Link
                            href={`/mutators/${mutator.id}/${mutator.version}${cleanQuery}`}
                            title="View details for this mutator"
                          >
                            {mutator.name}
                          </Link>
                        </TableCell>
                        <TableCell>
                          {mutator.columns
                            .map((col) => col.table)
                            .filter((value, index, self) => {
                              return self.indexOf(value) === index;
                            })
                            .join(', ')}
                        </TableCell>
                        <TableCell title={mutatorColumns}>
                          <strong>
                            {mutator.columns.length}{' '}
                            <span className={Styles.columnsCountText}>
                              Columns
                            </span>
                          </strong>
                          <span className={Styles.mutatorList}>
                            ({mutatorColumns})
                          </span>
                          <div className={Styles.mutatorTagsContainer}>
                            {mutator.columns.map((col, index) => (
                              <Tag
                                // eslint-disable-next-line react/no-array-index-key
                                key={`${col.name}-${index}`}
                                tag={col.name}
                                theme="primary"
                              />
                            ))}
                          </div>
                        </TableCell>
                        <TableCell>
                          <TextShortener text={mutator.id} length={6} />
                        </TableCell>
                        <TableCell>{mutator.version}</TableCell>
                      </TableRow>
                    );
                  })}
                </TableBody>
              </Table>
            </>
          ) : (
            <CardRow>
              <EmptyState
                title="No mutators"
                image={<IconUserReceived2 size="large" />}
              >
                {selectedTenant?.is_admin && (
                  <Button theme="secondary">
                    <Link
                      href={`/mutators/create${cleanQuery}`}
                      applyStyles={false}
                    >
                      Add Mutator
                    </Link>
                  </Button>
                )}
              </EmptyState>
            </CardRow>
          )
        ) : isFetching ? (
          <Text>Fetching tenant mutators...</Text>
        ) : (
          <InlineNotification theme="alert">
            {fetchError || 'Something went wrong'}
          </InlineNotification>
        )}
      </Card>
    </>
  );
};
const ConnectedMutatorList = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  mutators: state.mutators,
  isFetching: state.fetchingMutators,
  fetchError: state.fetchMutatorsError,
  deleteQueue: state.mutatorsToDelete,
  saveSuccess: state.bulkUpdateMutatorsSuccess,
  saveErrors: state.bulkUpdateMutatorsErrors,
  query: state.query,
  mutatorSearchFilter: state.mutatorSearchFilter,
}))(MutatorList);

const MutatorsPage = ({
  selectedTenantID,

  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchMutators(selectedTenantID, query));
    }
  }, [selectedTenantID, query, dispatch]);

  return <ConnectedMutatorList />;
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  query: state.query,
}))(MutatorsPage);
