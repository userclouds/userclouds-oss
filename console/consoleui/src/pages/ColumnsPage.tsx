import { useCallback, useEffect, useRef, useState } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  IconButton,
  IconDatabase2,
  IconFilter,
  IconLock2,
  IconUserReceived2,
  InputReadOnly,
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
  Dialog,
  DialogBody,
  DialogFooter,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import {
  Column,
  COLUMN_PREFIX,
  COLUMN_COLUMNS,
} from '../models/TenantUserStoreConfig';
import { SelectedTenant } from '../models/Tenant';
import PaginatedResult from '../models/PaginatedResult';
import { Filter } from '../models/authz/SearchFilters';
import {
  deleteColumn,
  fetchDataTypes,
  fetchUserStoreDisplayColumns,
  saveUserStore,
} from '../thunks/userstore';
import {
  changeColumnSearchFilter,
  deleteBulkUserStoreColumn,
} from '../actions/userstore';
import Link from '../controls/Link';
import Pagination from '../controls/Pagination';
import Search from '../controls/Search';
import {
  applySort,
  columnSortDirection,
  MAX_LIMIT,
} from '../controls/PaginationHelper';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';
import PageCommon from './PageCommon.module.css';
import styles from './ColumnsPage.module.css';

const SchemaColumnRow = ({
  selectedTenantID,
  col,
  columnsToDelete,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  col: Column;
  columnsToDelete: Record<string, Column>;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const queuedForDelete = !!columnsToDelete[col.id];
  const cleanQuery = makeCleanPageLink(query);

  return (
    <TableRow
      key={`column_row_${col.id}`}
      className={
        (queuedForDelete ? PageCommon.queuedfordelete : '') +
        ' ' +
        PageCommon.listviewtablerow
      }
    >
      <TableCell>
        <Checkbox
          id={'delete' + col.id}
          name="delete object"
          checked={columnsToDelete[col.id] ?? false}
          onChange={() => {
            dispatch(deleteBulkUserStoreColumn(col));
          }}
        />
      </TableCell>
      <TableCell className={PageCommon.tablePrimaryColumn}>
        <Link
          href={`/columns/${col.id}${cleanQuery}`}
          title="Column details page"
        >
          <Text>{col.name}</Text>
        </Link>
      </TableCell>
      <TableCell>
        <Text>{col.table}</Text>
      </TableCell>
      <TableCell>
        <InputReadOnly>{col.data_type.name}</InputReadOnly>
      </TableCell>
      <TableCell className={PageCommon.uuidtablecell}>
        <TextShortener text={col.id} length={6} />
      </TableCell>
      <TableCell>
        <InputReadOnly type="checkbox" isChecked={col.is_array === true} />
      </TableCell>
      <TableCell className={PageCommon.listviewtabledeletecell}>
        {col.is_system ? (
          <IconButton
            icon={<IconLock2 />}
            onClick={() => {}}
            title="Cannot delete system column"
            disabled
            aria-label="Cannot delete system column"
          />
        ) : (
          <DeleteWithConfirmationButton
            id="deleteColumnButton"
            message="Are you sure you want to delete this column? This action is irreversible."
            onConfirmDelete={() => {
              if (selectedTenantID) {
                dispatch(deleteColumn(selectedTenantID, col.id));
              }
            }}
            title="Delete Column"
          />
        )}
      </TableCell>
    </TableRow>
  );
};
const ConnectedSchemaRow = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  columnsToDelete: state.userStoreColumnsToDelete,
  query: state.query,
}))(SchemaColumnRow);

const SchemaTable = ({
  selectedTenant,
  columns,
  isFetching,
  error,
  query,
  deleteQueue,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  columns: PaginatedResult<Column> | undefined;
  isFetching: boolean;
  error: string;
  query: URLSearchParams;
  deleteQueue: Record<string, Column>;
  dispatch: AppDispatch;
}) => {
  return (
    <>
      {columns ? (
        columns.data && columns.data.length ? (
          <Table
            spacing="packed"
            id="columnsTable"
            className={styles.columnstable}
          >
            <TableHead floating>
              <TableRow>
                <TableRowHead>
                  <Checkbox
                    checked={
                      Object.keys(deleteQueue).length === columns.data.length
                    }
                    onChange={() => {
                      const shouldMarkForDelete =
                        !deleteQueue[columns.data[0].id];
                      columns.data.forEach((o) => {
                        if (shouldMarkForDelete && !deleteQueue[o.id]) {
                          dispatch(deleteBulkUserStoreColumn(o));
                        } else if (!shouldMarkForDelete && deleteQueue[o.id]) {
                          dispatch(deleteBulkUserStoreColumn(o));
                        }
                      });
                    }}
                  />
                </TableRowHead>
                <TableRowHead
                  key="name_header"
                  sort={columnSortDirection(COLUMN_PREFIX, query, 'name')}
                >
                  <Link
                    href={'?' + applySort(COLUMN_PREFIX, query, 'name')}
                    applyStyles={false}
                  >
                    Name
                  </Link>
                </TableRowHead>
                <TableRowHead key="table_header">Table</TableRowHead>
                <TableRowHead key="type_header">Column Type</TableRowHead>
                <TableRowHead
                  key="id_header"
                  sort={columnSortDirection(COLUMN_PREFIX, query, 'id')}
                >
                  <Link
                    href={'?' + applySort(COLUMN_PREFIX, query, 'id')}
                    applyStyles={false}
                  >
                    ID
                  </Link>
                </TableRowHead>
                <TableRowHead key="array_header">Array</TableRowHead>
                <TableRowHead key="delete_header" />
              </TableRow>
            </TableHead>
            <TableBody>
              {columns.data.length ? (
                <>
                  {columns.data.map((col: Column) => (
                    <ConnectedSchemaRow col={col} key={col.id} />
                  ))}
                </>
              ) : (
                <TableRow>
                  <TableCell colSpan={6}>No columns.</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        ) : (
          <CardRow>
            <EmptyState
              title="No columns"
              image={<IconUserReceived2 size="large" />}
            >
              {selectedTenant?.is_admin && (
                <Button theme="secondary">
                  <Link
                    href={'/columns/create' + makeCleanPageLink(query)}
                    applyStyles={false}
                  >
                    Create Column
                  </Link>
                </Button>
              )}
            </EmptyState>
          </CardRow>
        )
      ) : isFetching ? (
        <Text>Fetching tenant columns…</Text>
      ) : (
        <InlineNotification theme="alert">
          {error || 'Something went wrong'}
        </InlineNotification>
      )}
    </>
  );
};
const ConnectedSchemaTable = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  selectedTenantID: state.selectedTenantID,
  columnsToAdd: state.userStoreColumnsToAdd,
  isFetching: state.fetchingColumn,
  error: state.fetchingColumnError,
  query: state.query,
  deleteQueue: state.userStoreColumnsToDelete,
}))(SchemaTable);

const UserColumns = ({
  selectedTenant,
  userStoreDisplayColumns,
  fetching,
  fetchError,
  saveSuccess,
  saveErrors,
  query,
  columnSearchFilter,
  deleteQueue,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  userStoreDisplayColumns: PaginatedResult<Column> | undefined;
  fetching: boolean;
  fetchError: string;
  saveSuccess: string;
  saveErrors: string[];
  query: URLSearchParams;
  columnSearchFilter: Filter;
  deleteQueue: Record<string, Column>;
  dispatch: AppDispatch;
}) => {
  const [errorDialogOpen, setErrorDialogOpen] = useState(false);
  const [manuallyClosedDialog, setManuallyClosedDialog] = useState(false);

  const errorDialogRef = useRef<HTMLDialogElement>(null);
  const errorDialogId = 'saveErrorsDialog';

  const handleOpenDialog = useCallback(() => {
    errorDialogRef.current?.showModal();
    setErrorDialogOpen(true);
  }, [errorDialogRef]);

  const handleCloseDialog = useCallback(() => {
    errorDialogRef.current?.close();
    setErrorDialogOpen(false);
    setManuallyClosedDialog(true);
  }, [errorDialogRef]);

  useEffect(() => {
    // If there are errors and the dialog was not manually closed,
    // automatically open the dialog
    if (!!saveErrors.length && !manuallyClosedDialog) {
      handleOpenDialog();
    }
  }, [saveErrors, handleOpenDialog, manuallyClosedDialog]);

  const deletePrompt = `Are you sure you want to delete ${
    Object.keys(deleteQueue).length
  } column${Object.keys(deleteQueue).length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <Dialog
        open={errorDialogOpen}
        title="Delete Errors"
        className={PageCommon.dialog}
        id={errorDialogId}
        isDismissable
        onClose={handleCloseDialog}
        ref={errorDialogRef}
      >
        <DialogBody>
          <Text>The following errors occurred while deleting the columns:</Text>
          <ul className={styles.saveErrorsDialogList}>
            {saveErrors.map((error) => (
              <li key={error}>{error}</li>
            ))}
          </ul>
        </DialogBody>
        <DialogFooter>
          <Button onClick={handleCloseDialog}>Close</Button>
        </DialogFooter>
      </Dialog>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="columns"
          columns={COLUMN_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeColumnSearchFilter(filter));
          }}
          prefix={COLUMN_PREFIX}
          searchFilter={columnSearchFilter}
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip direction="left">
            <>
              {'Manage '}
              <a
                href="https://docs.userclouds.com/docs/key-concepts"
                title="UserClouds documentation for User Store key concepts"
                target="new"
                className={PageCommon.link}
              >
                user store data configuration
              </a>
              {' and settings for your tenant.'}
            </>
          </ToolTip>
        </div>

        {selectedTenant?.is_admin && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
          >
            <Link
              href={'/columns/create' + makeCleanPageLink(query)}
              applyStyles={false}
            >
              Create Column
            </Link>
          </Button>
        )}
      </div>
      <Card
        id="userstoreColumns"
        lockedMessage={
          !selectedTenant?.is_admin ? 'You do not have edit access' : ''
        }
        listview
      >
        {!!saveErrors.length && (
          <div className={PageCommon.tableNotification}>
            <InlineNotification theme="alert">
              {`${saveErrors.length} errors occurred while saving your edits.`}{' '}
              {!errorDialogOpen && (
                <button
                  onClick={handleOpenDialog}
                  className={PageCommon.tableNotificationButton}
                >
                  View {saveErrors.length === 1 ? ' Error' : 'Errors'}
                </button>
              )}
            </InlineNotification>
          </div>
        )}

        {saveSuccess && (
          <InlineNotification theme="success">{saveSuccess}</InlineNotification>
        )}

        {(fetching ||
          (!fetching && !userStoreDisplayColumns && !fetchError)) && (
          <Text>Fetching tenant User Store config…</Text>
        )}

        {!fetching && !userStoreDisplayColumns && (
          <CardRow className={PageCommon.tableNotification}>
            <InlineNotification theme="alert">
              {fetchError || 'Error fetching user store configuration'}
            </InlineNotification>
          </CardRow>
        )}

        {!fetching &&
          userStoreDisplayColumns &&
          !userStoreDisplayColumns?.data?.length && (
            <CardRow className={PageCommon.emptyState}>
              <EmptyState
                title="Nothing to display"
                subTitle="No user store columns have been specified yet for this tenant."
                image={<IconDatabase2 size="large" />}
              >
                {selectedTenant?.is_admin && (
                  <Link href={'/columns/create' + makeCleanPageLink(query)}>
                    <Button theme="secondary">Add Column</Button>
                  </Link>
                )}
              </EmptyState>
            </CardRow>
          )}

        {userStoreDisplayColumns?.data &&
          !!userStoreDisplayColumns.data?.length && (
            <>
              <div className={PageCommon.listviewpaginationcontrols}>
                <div className={PageCommon.listviewpaginationcontrolsdelete}>
                  <DeleteWithConfirmationButton
                    disabled={Object.keys(deleteQueue).length < 1}
                    id="deleteColumnsButton"
                    message={deletePrompt}
                    onConfirmDelete={() => {
                      dispatch(saveUserStore());
                      setManuallyClosedDialog(false);
                    }}
                    title="Delete Columns"
                  />
                </div>
                <Pagination
                  prev={userStoreDisplayColumns?.prev}
                  next={userStoreDisplayColumns?.next}
                  isLoading={fetching}
                  prefix={COLUMN_PREFIX}
                />
              </div>
              <ConnectedSchemaTable columns={userStoreDisplayColumns} />
            </>
          )}
      </Card>
    </>
  );
};

const ConnectedUserColumns = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  userStoreDisplayColumns: state.userStoreDisplayColumns,
  fetching: state.fetchingColumn,
  fetchError: state.fetchingColumnError,
  saving: state.savingUserStoreConfig,
  saveSuccess: state.saveUserStoreConfigSuccess,
  saveErrors: state.saveUserStoreConfigErrors,
  query: state.query,
  columnSearchFilter: state.columnSearchFilter,
  deleteQueue: state.userStoreColumnsToDelete,
}))(UserColumns);

const ColumnsPage = ({
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
      dispatch(fetchUserStoreDisplayColumns(selectedTenantID, query));
    }
  }, [selectedTenantID, query, dispatch]);

  useEffect(() => {
    if (selectedTenantID) {
      const params = new URLSearchParams();
      params.set('data_types_limit', String(MAX_LIMIT));
      dispatch(fetchDataTypes(selectedTenantID, params));
    }
  }, [selectedTenantID, dispatch]);

  return <ConnectedUserColumns />;
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  location: state.location,
  query: state.query,
}))(ColumnsPage);
