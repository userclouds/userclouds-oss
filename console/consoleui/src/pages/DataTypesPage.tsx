import clsx from 'clsx';
import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  IconDatabase2,
  IconFilter,
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
  ToolTip,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { RootState, AppDispatch } from '../store';
import {
  DataType,
  DATA_TYPE_COLUMNS,
  DATA_TYPE_PREFIX,
} from '../models/DataType';
import { SelectedTenant } from '../models/Tenant';
import PaginatedResult from '../models/PaginatedResult';
import {
  bulkDeleteDataTypes,
  deleteDataType,
  fetchDataTypes,
} from '../thunks/userstore';
import {
  changeDataTypeSearchFilter,
  toggleDataTypeForDelete,
} from '../actions/userstore';
import Link from '../controls/Link';
import Pagination from '../controls/Pagination';
import PageCommon from './PageCommon.module.css';
import styles from './DataTypesPage.module.css';
import Search from '../controls/Search';
import { Filter } from '../models/authz/SearchFilters';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';

const SchemaDataTypeRow = ({
  selectedTenantID,
  dataType,
  dataTypeDeleteQueue,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  dataType: DataType;
  dataTypeDeleteQueue: string[];
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);
  const queuedForDelete = dataTypeDeleteQueue.includes(dataType.id);
  return (
    <TableRow
      key={`dataType_row_${dataType.id}`}
      className={clsx(
        queuedForDelete ? PageCommon.queuedfordelete : '',
        PageCommon.listviewtablerow
      )}
    >
      <TableCell>
        <Checkbox
          id={'delete' + dataType.id}
          name="delete object"
          checked={queuedForDelete}
          onChange={() => {
            dispatch(toggleDataTypeForDelete(dataType));
          }}
        />
      </TableCell>
      <TableCell>
        <Link
          href={`/datatypes/${dataType.id}${cleanQuery}`}
          title="DataType details page"
        >
          <Text>{dataType.name}</Text>
        </Link>
      </TableCell>
      <TableCell>
        <InputReadOnly>
          {dataType.is_native ? 'UserClouds Default' : 'Custom'}
        </InputReadOnly>
      </TableCell>
      <TableCell className={PageCommon.uuidtablecell}>
        <InputReadOnly>{dataType.description}</InputReadOnly>
      </TableCell>

      <TableCell className={PageCommon.listviewtabledeletecell}>
        <DeleteWithConfirmationButton
          id="deleteDataTypeButton"
          message="Are you sure you want to delete this data type? This action is irreversible."
          onConfirmDelete={() => {
            if (selectedTenantID) {
              dispatch(deleteDataType(selectedTenantID, dataType.id));
            }
          }}
          title="Delete Data Type"
        />
      </TableCell>
    </TableRow>
  );
};
const ConnectedSchemaRow = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  query: state.query,
  dataTypeDeleteQueue: state.dataTypesDeleteQueue,
}))(SchemaDataTypeRow);

const SchemaTable = ({
  selectedTenant,
  dataTypes,
  isFetching,
  error,
  query,
  deleteQueue,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  dataTypes: PaginatedResult<DataType> | undefined;
  isFetching: boolean;
  error: string;
  query: URLSearchParams;
  deleteQueue: string[];
  dispatch: AppDispatch;
}) => {
  return (
    <>
      {dataTypes ? (
        dataTypes.data && dataTypes.data.length ? (
          <Table
            spacing="packed"
            id="dataTypesTable"
            className={styles.datatypestable}
          >
            <TableHead floating>
              <TableRow>
                <TableRowHead>
                  <Checkbox
                    checked={
                      Object.keys(deleteQueue).length === dataTypes.data.length
                    }
                    onChange={() => {
                      const shouldMarkForDelete = !deleteQueue.includes(
                        dataTypes.data[0].id
                      );
                      dataTypes.data.forEach((o) => {
                        if (
                          shouldMarkForDelete &&
                          !deleteQueue.includes(o.id)
                        ) {
                          dispatch(toggleDataTypeForDelete(o));
                        } else if (
                          !shouldMarkForDelete &&
                          deleteQueue.includes(o.id)
                        ) {
                          dispatch(toggleDataTypeForDelete(o));
                        }
                      });
                    }}
                  />
                </TableRowHead>
                <TableRowHead key="data_type_header">Data Type</TableRowHead>
                <TableRowHead key="specification_header">
                  Specification
                </TableRowHead>
                <TableRowHead key="description_header">
                  Description
                </TableRowHead>
                <TableRowHead key="delete_header" />
              </TableRow>
            </TableHead>
            <TableBody>
              {dataTypes.data.length ? (
                <>
                  {dataTypes.data.map((datatype: DataType) => (
                    <ConnectedSchemaRow dataType={datatype} key={datatype.id} />
                  ))}
                </>
              ) : (
                <TableRow>
                  <TableCell colSpan={5}>No data types.</TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        ) : (
          <CardRow>
            <EmptyState
              title="No dataTypes"
              image={<IconUserReceived2 size="large" />}
            >
              {selectedTenant?.is_admin && (
                <Button theme="secondary">
                  <Link href={`/datatypes/create` + makeCleanPageLink(query)}>
                    Add Data Type
                  </Link>
                </Button>
              )}
            </EmptyState>
          </CardRow>
        )
      ) : isFetching ? (
        <Text>Fetching tenant Data Types…</Text>
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
  isFetching: state.fetchingDataType,
  error: state.fetchingDataTypeError,
  query: state.query,
  deleteQueue: state.dataTypesDeleteQueue,
}))(SchemaTable);

const UserDataTypes = ({
  selectedTenant,
  dataTypes,
  fetching,
  fetchError,
  saveSuccess,
  saveErrors,
  query,
  dataTypeSearchFilter,
  deleteQueue,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  dataTypes: PaginatedResult<DataType> | undefined;
  fetching: boolean;
  fetchError: string;
  saveSuccess: string;
  saveErrors: string[];
  query: URLSearchParams;
  dataTypeSearchFilter: Filter;
  deleteQueue: string[];
  dispatch: AppDispatch;
}) => {
  const onConfirmDelete = () => {
    if (selectedTenant) {
      dispatch(bulkDeleteDataTypes(selectedTenant.id, deleteQueue));
    }
  };

  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } data type${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <IconFilter />
        <Search
          id="dataTypes"
          columns={DATA_TYPE_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeDataTypeSearchFilter(filter));
          }}
          prefix={DATA_TYPE_PREFIX}
          searchFilter={dataTypeSearchFilter}
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
              href={'/datatypes/create' + makeCleanPageLink(query)}
              applyStyles={false}
            >
              Create Data Type
            </Link>
          </Button>
        )}
      </div>

      <Card
        id="userstoreDataTypes"
        lockedMessage={
          !selectedTenant?.is_admin ? 'You do not have edit access' : ''
        }
        listview
      >
        {!!saveErrors.length && (
          <div className={PageCommon.tableNotification}>
            <InlineNotification theme="alert">
              {saveErrors.length === 1
                ? saveErrors[0]
                : `${saveErrors.length} errors occurred while saving your edits`}
            </InlineNotification>
          </div>
        )}

        {saveSuccess && (
          <div className={PageCommon.tableNotification}>
            <InlineNotification theme="success">
              {saveSuccess}
            </InlineNotification>
          </div>
        )}

        {(fetching || (!fetching && !dataTypes && !fetchError)) && (
          <Text>Fetching tenant Data Types…</Text>
        )}

        {!fetching && !dataTypes?.data && (
          <CardRow className={PageCommon.tableNotification}>
            <InlineNotification theme="alert">
              {fetchError || 'Error fetching Data Types'}
            </InlineNotification>
          </CardRow>
        )}

        {!fetching && dataTypes?.data && !dataTypes.data.length && (
          <CardRow className={PageCommon.emptyState}>
            <EmptyState
              title="Nothing to display"
              subTitle="No user store dataTypes have been specified yet for this tenant."
              image={<IconDatabase2 size="large" />}
            >
              {selectedTenant?.is_admin && (
                <Button theme="secondary">
                  <Link
                    href={`/datatypes/create` + makeCleanPageLink(query)}
                    applyStyles={false}
                  >
                    Add Data Type
                  </Link>
                </Button>
              )}
            </EmptyState>
          </CardRow>
        )}

        {dataTypes?.data && dataTypes.data.length > 0 && (
          <>
            <div className={PageCommon.listviewpaginationcontrols}>
              <div className={PageCommon.listviewpaginationcontrolsdelete}>
                <DeleteWithConfirmationButton
                  disabled={Object.keys(deleteQueue).length < 1}
                  id="deleteDataTypesButton"
                  message={deletePrompt}
                  onConfirmDelete={onConfirmDelete}
                  title="Delete Data Types"
                />
              </div>
              <Pagination
                prev={dataTypes?.prev}
                next={dataTypes?.next}
                isLoading={fetching}
                prefix={DATA_TYPE_PREFIX}
              />
            </div>
            <ConnectedSchemaTable dataTypes={dataTypes} />
          </>
        )}
      </Card>
    </>
  );
};
const ConnectedUserDataTypes = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  dataTypes: state.dataTypes,
  fetching: state.fetchingDataTypes,
  fetchError: state.fetchingDataTypeError,
  saving: state.savingUserStoreConfig,
  saveSuccess: state.saveUserStoreConfigSuccess,
  saveErrors: state.saveUserStoreConfigErrors,
  query: state.query,
  dataTypeSearchFilter: state.dataTypeSearchFilter,
  deleteQueue: state.dataTypesDeleteQueue,
}))(UserDataTypes);

const DataTypesPage = ({
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
      dispatch(fetchDataTypes(selectedTenantID, query));
    }
  }, [selectedTenantID, dispatch, query]);

  return <ConnectedUserDataTypes />;
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  query: state.query,
}))(DataTypesPage);
