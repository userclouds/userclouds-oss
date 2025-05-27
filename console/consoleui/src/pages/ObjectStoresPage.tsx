import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  IconButton,
  IconDatabase2,
  IconDeleteBin,
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
import { ObjectStore, OBJECT_STORE_PREFIX } from '../models/ObjectStore';
import { SelectedTenant } from '../models/Tenant';
import PaginatedResult from '../models/PaginatedResult';
import {
  bulkDeleteObjectStores,
  deleteObjectStore,
  fetchObjectStores,
} from '../thunks/userstore';
import { toggleObjectStoreForDelete } from '../actions/userstore';
import Link from '../controls/Link';
import Pagination from '../controls/Pagination';
import PageCommon from './PageCommon.module.css';
import styles from './ObjectStoresPage.module.css';

const ObjectStoreRow = ({
  selectedTenantID,
  objectStore,
  objectStoreDeleteQueue,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  objectStore: ObjectStore;
  objectStoreDeleteQueue: string[];
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);

  const queuedForDelete = objectStoreDeleteQueue.includes(objectStore.id);
  return (
    <TableRow
      key={`objectStore_row_${objectStore.id}`}
      className={
        (queuedForDelete ? PageCommon.queuedfordelete : '') +
        ' ' +
        PageCommon.listviewtablerow
      }
    >
      <TableCell>
        <Checkbox
          id={'delete' + objectStore.id}
          name={'delete' + objectStore.id}
          checked={queuedForDelete}
          onChange={() => {
            dispatch(toggleObjectStoreForDelete(objectStore));
          }}
        />
      </TableCell>
      <TableCell>
        <Link
          href={`/object_stores/${objectStore.id}${cleanQuery}`}
          title="Object store details page"
        >
          <Text>{objectStore.name}</Text>
        </Link>
      </TableCell>
      <TableCell>
        <Text>{objectStore.type}</Text>
      </TableCell>
      <TableCell>
        <Text>{objectStore.region}</Text>
      </TableCell>
      <TableCell className={PageCommon.listviewtabledeletecell}>
        <IconButton
          icon={<IconDeleteBin />}
          onClick={() => {
            const prompt = `This action will delete object store ${objectStore.name}. Proceed?`;

            const ok = window.confirm(prompt);
            if (!ok) {
              return;
            }
            if (selectedTenantID) {
              dispatch(deleteObjectStore(selectedTenantID, objectStore.id));
            }
          }}
          title="Delete Object Store"
          aria-label="Delete Object Store"
        />
      </TableCell>
    </TableRow>
  );
};
const ConnectedObjectStoreRow = connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  query: state.query,
  objectStoreDeleteQueue: state.objectStoreDeleteQueue,
}))(ObjectStoreRow);

const ObjectStoreTable = ({
  selectedTenant,
  objectStores,
  isFetching,
  error,
  query,
  deleteQueue,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  objectStores: PaginatedResult<ObjectStore> | undefined;
  isFetching: boolean;
  error: string;
  query: URLSearchParams;
  deleteQueue: string[];
  dispatch: AppDispatch;
}) => {
  return (
    <>
      {objectStores ? (
        objectStores.data && objectStores.data.length ? (
          <>
            <Table
              spacing="packed"
              id="objectStoresTable"
              className={styles.objectstorestable}
            >
              <TableHead floating>
                <TableRow>
                  <TableRowHead>
                    <Checkbox
                      checked={
                        Object.keys(deleteQueue).length ===
                        objectStores.data.length
                      }
                      onChange={() => {
                        const shouldMarkForDelete = !deleteQueue.includes(
                          objectStores.data[0].id
                        );
                        objectStores.data.forEach((o) => {
                          if (
                            shouldMarkForDelete &&
                            !deleteQueue.includes(o.id)
                          ) {
                            dispatch(toggleObjectStoreForDelete(o));
                          } else if (
                            !shouldMarkForDelete &&
                            deleteQueue.includes(o.id)
                          ) {
                            dispatch(toggleObjectStoreForDelete(o));
                          }
                        });
                      }}
                    />
                  </TableRowHead>
                  <TableRowHead key="object_store_name_header">
                    Object Store
                  </TableRowHead>
                  <TableRowHead key="object_store_type_header">
                    Type
                  </TableRowHead>
                  <TableRowHead key="object_store_region_header">
                    Region
                  </TableRowHead>
                  <TableRowHead key="delete_header" />
                </TableRow>
              </TableHead>
              <TableBody>
                {objectStores.data.length ? (
                  <>
                    {objectStores.data.map((objectStore: ObjectStore) => (
                      <ConnectedObjectStoreRow
                        objectStore={objectStore}
                        key={objectStore.id}
                      />
                    ))}
                  </>
                ) : (
                  <TableRow>
                    <TableCell colSpan={5}>No object stores.</TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </>
        ) : (
          <CardRow>
            <EmptyState
              title="No object store connections"
              image={<IconDatabase2 size="large" />}
            >
              {selectedTenant?.is_admin && (
                <Button theme="secondary">
                  <Link
                    href={`/object_stores/create` + makeCleanPageLink(query)}
                  >
                    Add Data Type
                  </Link>
                </Button>
              )}
            </EmptyState>
          </CardRow>
        )
      ) : isFetching ? (
        <Text>Fetching tenant object stores...</Text>
      ) : (
        <InlineNotification theme="alert">
          {error || 'Something went wrong'}
        </InlineNotification>
      )}
    </>
  );
};
const ConnectedObjectStoreTable = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  selectedTenantID: state.selectedTenantID,
  isFetching: state.fetchingDataType,
  error: state.fetchingDataTypeError,
  query: state.query,
  deleteQueue: state.objectStoreDeleteQueue,
}))(ObjectStoreTable);

const ObjectStores = ({
  selectedTenant,
  objectStores,
  fetching,
  saveSuccess,
  saveErrors,
  query,
  deleteQueue,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  objectStores: PaginatedResult<ObjectStore> | undefined;
  fetching: boolean;
  saveSuccess: string;
  saveErrors: string[];
  query: URLSearchParams;
  deleteQueue: string[];
  dispatch: AppDispatch;
}) => {
  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Create object store shims to manage access to your object stores
              using UserClouds access policies.
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
              href={'/object_stores/create' + makeCleanPageLink(query)}
              applyStyles={false}
            >
              Create Object Store
            </Link>
          </Button>
        )}
      </div>

      <Card
        id="userstoreObjectStores"
        lockedMessage={
          !selectedTenant?.is_admin ? 'You do not have edit access' : ''
        }
        listview
      >
        {!!saveErrors.length && (
          <InlineNotification theme="alert">
            {saveErrors.length === 1
              ? saveErrors[0]
              : `${saveErrors.length} errors occurred while saving your edits`}
          </InlineNotification>
        )}
        {saveSuccess && (
          <InlineNotification theme="success">{saveSuccess}</InlineNotification>
        )}
        {objectStores && objectStores.data ? (
          objectStores.data.length ? (
            <>
              <div className={PageCommon.listviewpaginationcontrols}>
                <IconButton
                  icon={<IconDeleteBin />}
                  onClick={() => {
                    const prompt = `This action will delete ${deleteQueue.length} object stores. Proceed?`;

                    const ok = window.confirm(prompt);
                    if (!ok) {
                      return;
                    }
                    if (selectedTenant) {
                      dispatch(
                        bulkDeleteObjectStores(selectedTenant.id, deleteQueue)
                      );
                    }
                  }}
                  size="small"
                  disabled={deleteQueue.length < 1}
                  className={PageCommon.listviewpaginationcontrolsdelete}
                  id="deleteButton"
                  aria-label="Delete Object Stores"
                />
                <Pagination
                  prev={objectStores?.prev}
                  next={objectStores?.next}
                  isLoading={fetching}
                  prefix={OBJECT_STORE_PREFIX}
                />
              </div>
              <ConnectedObjectStoreTable objectStores={objectStores} />
            </>
          ) : (
            <CardRow className={PageCommon.emptyState}>
              <EmptyState
                title="Nothing to display"
                subTitle="No object stores have been created for this tenant."
                image={<IconDatabase2 size="large" />}
              >
                {selectedTenant?.is_admin && (
                  <Button theme="secondary">
                    <Link
                      href={`/object_stores/create` + makeCleanPageLink(query)}
                      applyStyles={false}
                    >
                      Add Object Store
                    </Link>
                  </Button>
                )}
              </EmptyState>
            </CardRow>
          )
        ) : fetching ? (
          <Text>Fetching tenant User Store config...</Text>
        ) : (
          <CardRow className={PageCommon.tableNotification}>
            <InlineNotification theme="alert">
              Error fetching object stores
            </InlineNotification>
          </CardRow>
        )}
      </Card>
    </>
  );
};
const ConnectedObjectStores = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  objectStores: state.objectStores,
  fetching: state.fetchingObjectStore,
  saveSuccess: state.saveUserStoreConfigSuccess,
  saveErrors: state.saveUserStoreConfigErrors,
  query: state.query,
  deleteQueue: state.objectStoreDeleteQueue,
}))(ObjectStores);

const ObjectStoresPage = ({
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
      dispatch(fetchObjectStores(selectedTenantID, query));
    }
  }, [selectedTenantID, query, dispatch]);

  return (
    <>
      <ConnectedObjectStores />
    </>
  );
};

export default connect((state: RootState) => ({
  selectedTenantID: state.selectedTenantID,
  query: state.query,
}))(ObjectStoresPage);
