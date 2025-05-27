import React, { useEffect } from 'react';
import { AnyAction } from 'redux';
import { connect } from 'react-redux';

import { APIError } from '@userclouds/sharedui';
import {
  Button,
  Card,
  CardRow,
  Checkbox,
  EmptyState,
  IconFilter,
  IconLock2,
  InlineNotification,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableRowHead,
  TableCell,
  Tag,
  Text,
  ToolTip,
} from '@userclouds/ui-component-lib';

import { makeCleanPageLink } from '../AppNavigation';
import { AppDispatch, RootState } from '../store';
import { SelectedTenant } from '../models/Tenant';
import Link from '../controls/Link';
import { Filter } from '../models/authz/SearchFilters';
import Search from '../controls/Search';
import PaginatedResult from '../models/PaginatedResult';
import DataSource, {
  dataSourceColumns,
  dataSourcesPrefix,
  DataClassifications,
  DataFormats,
  DataStorageOptions,
} from '../models/DataSource';
import {
  toggleDataSourceForDelete,
  bulkDeleteDataSourcesError,
  bulkDeleteDataSourcesRequest,
  bulkDeleteDataSourcesSuccess,
  deleteDataSourceSuccess,
  deleteDataSourceError,
  changeDataSourcesSearchFilter,
} from '../actions/datamapping';
import { deleteTenantDataSource } from '../API/datamapping';
import { fetchDataSources } from '../thunks/datamapping';
import { postAlertToast } from '../thunks/notifications';

import PageCommon from './PageCommon.module.css';
import DeleteWithConfirmationButton from '../controls/DeleteWithConfirmationButton';

const onDeleteDataSources =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedTenantID, dataSourcesDeleteQueue, query } = getState();
    if (!selectedTenantID || !dataSourcesDeleteQueue) {
      return;
    }
    let promises: Array<Promise<AnyAction>> = [];
    const totalToDelete = dataSourcesDeleteQueue.length;
    if (totalToDelete) {
      dispatch(bulkDeleteDataSourcesRequest());
      promises = dataSourcesDeleteQueue.map((id) =>
        deleteTenantDataSource(selectedTenantID, id).then(
          () => {
            return dispatch(deleteDataSourceSuccess(id));
          },
          (err: APIError) => {
            dispatch(deleteDataSourceError(err));
            throw err;
          }
        )
      );
    }
    Promise.all(promises as Array<Promise<AnyAction>>).then(
      () => {
        dispatch(bulkDeleteDataSourcesSuccess());
        dispatch(fetchDataSources(selectedTenantID, query));
      },
      () => {
        dispatch(bulkDeleteDataSourcesError());
        const { bulkDeleteDataSourcesErrors } = getState();
        dispatch(
          postAlertToast(
            `Errors deleting ${bulkDeleteDataSourcesErrors.length} of ${totalToDelete} data sources`
          )
        );
      }
    );
  };

const DataSourcesPage = ({
  selectedTenant,
  dataSources,
  fetchingDataSources,
  dataSourcesFilter,
  deleteQueue,
  error,
  query,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  dataSources: PaginatedResult<DataSource> | undefined;
  fetchingDataSources: boolean;
  dataSourcesFilter: Filter;
  deleteQueue: string[];
  error: string;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenant) {
      dispatch(fetchDataSources(selectedTenant.id, query));
    }
  }, [selectedTenant, query, dispatch]);

  const deletePrompt = `Are you sure you want to delete ${
    deleteQueue.length
  } data source${deleteQueue.length === 1 ? '' : 's'}? This action is irreversible.`;

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="dataSources"
          columns={dataSourceColumns}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeDataSourcesSearchFilter(filter));
          }}
          prefix={dataSourcesPrefix}
          searchFilter={dataSourcesFilter}
        />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              Data sources...
              <a
                href="https://docs.userclouds.com/docs/key-concepts-1#datasources"
                title="UserClouds documentation for key concepts in data sources"
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
            <Link
              href={'/datasources/create' + makeCleanPageLink(query)}
              applyStyles={false}
            >
              {' '}
              Add data source
            </Link>
          </Button>
        )}
      </div>

      <Card
        lockedMessage={
          !selectedTenant?.is_admin ? 'You do not have edit access' : ''
        }
        listview
      >
        {dataSources ? (
          dataSources.data && dataSources.data.length ? (
            <>
              <div className={PageCommon.listviewpaginationcontrols}>
                <div className={PageCommon.listviewpaginationcontrolsdelete}>
                  <DeleteWithConfirmationButton
                    disabled={Object.keys(deleteQueue).length < 1}
                    id="deleteDataSourcesButton"
                    message={deletePrompt}
                    onConfirmDelete={() => {
                      dispatch(onDeleteDataSources());
                    }}
                    title="Delete Columns"
                  />
                </div>
              </div>
              <Table spacing="packed" id="objectTypes">
                <TableHead>
                  <TableRow>
                    <TableRowHead key="data_source_bulk_select">
                      <Checkbox
                        checked={
                          Object.keys(deleteQueue).length ===
                          dataSources.data.length
                        }
                        onChange={() => {
                          const shouldMarkForDelete = !deleteQueue.includes(
                            dataSources.data[0].id
                          );
                          dataSources.data.forEach((o) => {
                            if (
                              shouldMarkForDelete &&
                              !deleteQueue.includes(o.id)
                            ) {
                              dispatch(toggleDataSourceForDelete(o));
                            } else if (
                              !shouldMarkForDelete &&
                              deleteQueue.includes(o.id)
                            ) {
                              dispatch(toggleDataSourceForDelete(o));
                            }
                          });
                        }}
                      />
                    </TableRowHead>
                    <TableRowHead key="source_name">Name</TableRowHead>
                    <TableRowHead key="contains_pii">PII</TableRowHead>
                    <TableRowHead key="source_format">Format</TableRowHead>
                    <TableRowHead key="source_classifications">
                      Classifications
                    </TableRowHead>
                    <TableRowHead key="type_name">Type</TableRowHead>
                    <TableRowHead key="source_storage">Storage</TableRowHead>
                    <TableRowHead key="3p_hosted">3P-Hosted</TableRowHead>
                    <TableRowHead key="3p_managed">3P-Managed</TableRowHead>
                    <TableRowHead key="delete_header" />
                  </TableRow>
                </TableHead>
                <TableBody>
                  {dataSources.data.map((ds) => (
                    <TableRow
                      key={ds.id}
                      className={
                        (deleteQueue.includes(ds.id)
                          ? PageCommon.queuedfordelete
                          : '') +
                        (' ' + PageCommon.listviewtablerow)
                      }
                    >
                      <TableCell>
                        <Checkbox
                          id={'delete' + ds.id}
                          name="delete data source"
                          checked={deleteQueue.includes(ds.id)}
                          onChange={() => {
                            dispatch(toggleDataSourceForDelete(ds));
                          }}
                        />
                      </TableCell>
                      <TableCell title={ds.id}>
                        {selectedTenant?.is_admin ? (
                          <Link
                            key={ds.id}
                            href={
                              '/datasources/' + ds.id + makeCleanPageLink(query)
                            }
                          >
                            {ds.name}
                          </Link>
                        ) : (
                          ds.name
                        )}
                      </TableCell>
                      <TableCell>
                        {ds.metadata?.contains_pii ? 'Yes' : 'No'}
                      </TableCell>
                      <TableCell>
                        {(ds.metadata.format &&
                          ds.metadata.format.length &&
                          ds.metadata.format.map(
                            (f: keyof typeof DataFormats) => (
                              <React.Fragment key={f}>
                                <Tag tag={DataFormats[f] || f} />{' '}
                              </React.Fragment>
                            )
                          )) ||
                          '-'}
                      </TableCell>
                      <TableCell>
                        {(ds.metadata.classifications &&
                          ds.metadata.classifications.length &&
                          ds.metadata.classifications.map(
                            (c: keyof typeof DataClassifications) => (
                              <React.Fragment key={c}>
                                <Tag tag={DataClassifications[c] || c} />{' '}
                              </React.Fragment>
                            )
                          )) ||
                          '-'}
                      </TableCell>
                      <TableCell>{ds.type}</TableCell>
                      <TableCell>
                        {DataStorageOptions[
                          ds.metadata
                            ?.storage as keyof typeof DataStorageOptions
                        ] || '-'}
                      </TableCell>
                      <TableCell>
                        {ds.metadata && ds.metadata['3p_hosted'] ? 'Yes' : 'No'}
                      </TableCell>
                      <TableCell>
                        {ds.metadata && ds.metadata['3p_managed']
                          ? 'Yes'
                          : 'No'}
                      </TableCell>
                      <TableCell
                        align="right"
                        className={PageCommon.listviewtabledeletecell}
                      >
                        <DeleteWithConfirmationButton
                          id="deleteDataSourceButton"
                          message="Are you sure you want to delete this data source? This action is irreversible."
                          onConfirmDelete={() => {
                            dispatch(onDeleteDataSources());
                          }}
                          title="Delete data source"
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </>
          ) : (
            <CardRow>
              <EmptyState
                title="No data sources"
                image={<IconLock2 size="large" />}
              >
                <Button theme="secondary">
                  <Link
                    href={'/datasources/create' + makeCleanPageLink(query)}
                    applyStyles={false}
                  >
                    Add Data Source
                  </Link>
                </Button>
              </EmptyState>
            </CardRow>
          )
        ) : fetchingDataSources ? (
          <Text element="h4">Loading...</Text>
        ) : (
          <InlineNotification theme="alert">
            {error || 'Something went wrong'}
          </InlineNotification>
        )}
      </Card>
    </>
  );
};
const ConnectedDataSourcesPage = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    dataSources: state.dataSources,
    fetchingDataSources: state.fetchingDataSources,
    dataSourcesFilter: state.dataSourcesSearchFilter,
    deleteQueue: state.dataSourcesDeleteQueue,
    error: state.fetchDataSourceError,
    query: state.query,
  };
})(DataSourcesPage);

export default ConnectedDataSourcesPage;
