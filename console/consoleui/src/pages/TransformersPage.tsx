import { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  Checkbox,
  EmptyState,
  GlobalStyles,
  IconFileList2,
  IconFilter,
  InlineNotification,
  InputReadOnly,
  Table,
  TableHead,
  TableRow,
  TableRowHead,
  TableBody,
  TableCell,
  Text,
  TextShortener,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import {
  bulkDeleteTransformers,
  deleteSingleTransformer,
  fetchTransformers,
  fetchUserPolicyPermissions,
} from '../thunks/tokenizer';
import {
  toggleTransformerForDelete,
  changeTransformerSearchFilter,
} from '../actions/tokenizer';
import { AppDispatch, RootState } from '../store';
import PaginatedResult from '../models/PaginatedResult';
import Transformer, {
  TRANSFORMERS_PREFIX,
  TRANSFORMER_COLUMNS,
  TransformTypeFriendly,
} from '../models/Transformer';
import { Filter } from '../models/authz/SearchFilters';
import PermissionsOnObject from '../models/authz/Permissions';
import Link from '../controls/Link';
import Pagination from '../controls/Pagination';
import Search from '../controls/Search';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';
import PageCommon from './PageCommon.module.css';
import styles from './TransformersPage.module.css';

type TransformersProps = {
  selectedTenantID: string | undefined;
  transformers: PaginatedResult<Transformer> | undefined;
  fetchError: string | undefined;
  isFetching: boolean;
  deleteQueue: string[];
  deleteSuccess: string;
  deleteErrors: string[];
  permissions: PermissionsOnObject | undefined;
  fetchingPermissions: boolean;
  permissionsFetchError: string;
  transformerSearchFilter: Filter;
  query: URLSearchParams;
  dispatch: AppDispatch;
};

const TransformerList = ({
  selectedTenantID,
  transformers,
  fetchError,
  isFetching,
  deleteQueue,
  deleteSuccess,
  deleteErrors,
  permissions,
  fetchingPermissions,
  permissionsFetchError,
  transformerSearchFilter,
  query,
  dispatch,
}: TransformersProps) => {
  const cleanQuery = makeCleanPageLink(query);
  const fetchingInProgress =
    isFetching ||
    fetchingPermissions ||
    (!permissions && !permissionsFetchError) ||
    (permissions?.read && !transformers && !fetchError);

  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } transformer${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="transformers"
          columns={TRANSFORMER_COLUMNS}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeTransformerSearchFilter(filter));
          }}
          prefix={TRANSFORMERS_PREFIX}
          searchFilter={transformerSearchFilter}
        />

        <ToolTip>
          <>
            {
              'Transformers are re-usable functions that manipulate data in UserClouds. '
            }
            <a
              href="https://docs.userclouds.com/docs/transformers-1"
              title="UserClouds documentation for transformers"
              target="new"
              className={PageCommon.link}
            >
              Learn more here.
            </a>
          </>
        </ToolTip>
        {permissions?.create && (
          <Button
            theme="primary"
            size="small"
            className={PageCommon.listviewtablecontrolsButton}
          >
            <Link
              href={`/transformers/create` + makeCleanPageLink(query)}
              applyStyles={false}
            >
              Create Transformer
            </Link>
          </Button>
        )}
      </div>

      <Card listview>
        {transformers ? (
          transformers.data && transformers.data.length ? (
            <>
              {!!deleteErrors.length && (
                <InlineNotification
                  theme="alert"
                  elementName="div"
                  className={GlobalStyles['mb-3']}
                >
                  <ul>
                    {deleteErrors.map((error: string) => (
                      <li key={`transformers_error_${error}`}>{error}</li>
                    ))}
                  </ul>
                </InlineNotification>
              )}
              {deleteSuccess && (
                <InlineNotification
                  theme="success"
                  className={GlobalStyles['mb-3']}
                >
                  {deleteSuccess}
                </InlineNotification>
              )}
              <div className={PageCommon.listviewpaginationcontrols}>
                <div className={PageCommon.listviewpaginationcontrolsdelete}>
                  <DeleteWithConfirmationButton
                    id="deleteTransformersButton"
                    message={deletePrompt}
                    onConfirmDelete={() => {
                      if (selectedTenantID) {
                        dispatch(
                          bulkDeleteTransformers(selectedTenantID, deleteQueue)
                        );
                      }
                    }}
                    title="Delete Transformers"
                    disabled={deleteQueue.length < 1}
                  />
                </div>
                <Pagination
                  prev={transformers?.prev}
                  next={transformers?.next}
                  prefix={TRANSFORMERS_PREFIX}
                  isLoading={isFetching}
                />
              </div>

              <Table
                spacing="nowrap"
                id="transformers"
                className={styles.transformerstable}
              >
                <TableHead floating>
                  <TableRow>
                    <TableRowHead>
                      <Checkbox
                        checked={
                          Object.keys(deleteQueue).length ===
                          transformers.data.length
                        }
                        onChange={() => {
                          const shouldMarkForDelete = !deleteQueue.includes(
                            transformers.data[0].id
                          );
                          transformers.data.forEach((o) => {
                            if (
                              shouldMarkForDelete &&
                              !deleteQueue.includes(o.id)
                            ) {
                              dispatch(toggleTransformerForDelete(o.id));
                            } else if (
                              !shouldMarkForDelete &&
                              deleteQueue.includes(o.id)
                            ) {
                              dispatch(toggleTransformerForDelete(o.id));
                            }
                          });
                        }}
                      />
                    </TableRowHead>
                    <TableRowHead>Name</TableRowHead>
                    <TableRowHead>Transform Type</TableRowHead>
                    <TableRowHead>ID</TableRowHead>
                    <TableRowHead key="purpose_delete" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {transformers.data.map((entry) => (
                    <TableRow
                      key={entry.id}
                      className={
                        (deleteQueue.includes(entry.id)
                          ? PageCommon.queuedfordelete
                          : '') +
                        ' ' +
                        PageCommon.listviewtablerow
                      }
                    >
                      <TableCell>
                        <Checkbox
                          id={'delete' + entry.id}
                          name="delete object"
                          checked={deleteQueue.includes(entry.id)}
                          onChange={() => {
                            dispatch(toggleTransformerForDelete(entry.id));
                          }}
                        />
                      </TableCell>
                      <TableCell>
                        <Link
                          href={`/transformers/${entry.id}/${entry.version}${cleanQuery}`}
                        >
                          <Text>{entry.name}</Text>
                        </Link>
                      </TableCell>
                      <TableCell>
                        <InputReadOnly>
                          {TransformTypeFriendly[entry.transform_type]}
                        </InputReadOnly>
                      </TableCell>
                      <TableCell>
                        <TextShortener text={entry.id} length={6} />
                      </TableCell>
                      <TableCell
                        align="right"
                        className={PageCommon.listviewtabledeletecell}
                      >
                        <DeleteWithConfirmationButton
                          id="deleteTransformerButton"
                          message="Are you sure you want to delete this transformer? This action is irreversible."
                          onConfirmDelete={() => {
                            if (selectedTenantID) {
                              dispatch(
                                deleteSingleTransformer(
                                  selectedTenantID,
                                  entry,
                                  query
                                )
                              );
                            }
                          }}
                          title="Delete Transformer"
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </>
          ) : (
            <EmptyState
              title="No transformers"
              image={<IconFileList2 size="large" />}
            >
              <Button theme="secondary">
                <Link
                  href={`/transformers/create${cleanQuery}`}
                  applyStyles={false}
                >
                  Add Transformer
                </Link>
              </Button>
            </EmptyState>
          )
        ) : fetchingInProgress ? (
          <Text>Loading transformers...</Text>
        ) : permissions && !permissions.read ? (
          <Card
            title="Request access"
            description="You do not have permission to view any transformers. Please contact your administrator to request access."
          />
        ) : (
          <InlineNotification theme="alert">
            {fetchError || permissionsFetchError || 'Something went wrong'}
          </InlineNotification>
        )}
      </Card>
    </>
  );
};
const ConnectedTransformerList = connect((state: RootState) => {
  return {
    selectedTenantID: state.selectedTenantID,
    transformers: state.transformers,
    fetchError: state.transformerFetchError,
    isFetching: state.fetchingTransformers,
    deleteQueue: state.transformersDeleteQueue,
    deleteSuccess: state.deleteTransformersSuccess,
    deleteErrors: state.deleteTransformersErrors,
    transformerSearchFilter: state.transformerSearchFilter,
    permissions: state.userPolicyPermissions,
    fetchingPermissions: state.fetchingUserPolicyPermissions,
    permissionsFetchError: state.userPolicyPermissionsFetchError,
    query: state.query,
  };
})(TransformerList);

const TransformersPage = ({
  selectedTenantID,
  userPolicyPermissions,
  query,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  userPolicyPermissions: PermissionsOnObject | undefined;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenantID) {
      dispatch(fetchUserPolicyPermissions(selectedTenantID));
    }
  }, [dispatch, selectedTenantID]);

  useEffect(() => {
    if (selectedTenantID && userPolicyPermissions?.read) {
      dispatch(fetchTransformers(selectedTenantID, query));
    }
  }, [dispatch, userPolicyPermissions, selectedTenantID, query]);

  return <ConnectedTransformerList />;
};

export default connect((state: RootState) => {
  return {
    selectedTenantID: state.selectedTenantID,
    userPolicyPermissions: state.userPolicyPermissions,
    query: state.query,
  };
})(TransformersPage);
