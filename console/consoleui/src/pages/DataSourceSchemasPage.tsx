import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  CardRow,
  EmptyState,
  IconFilter,
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
import { Filter } from '../models/authz/SearchFilters';
import { SelectedTenant } from '../models/Tenant';
import DataSource, {
  DataSourceElement,
  UserDataTypes,
  Regulations,
  dataSourceElementColumns,
  dataSourceElementsPrefix,
} from '../models/DataSource';
import PaginatedResult from '../models/PaginatedResult';
import { changeDataSourceElementsSearchFilter } from '../actions/datamapping';
import {
  fetchDataSources,
  fetchDataSourceElements,
} from '../thunks/datamapping';
import Search from '../controls/Search';
import Link from '../controls/Link';
import PageCommon from './PageCommon.module.css';

const DataSourceSchemasPage = ({
  selectedTenant,
  dataSources,
  fetchingDataSources,
  elements,
  fetchingDataSourceElements,
  elementsFilter,
  query,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  dataSources: PaginatedResult<DataSource> | undefined;
  fetchingDataSources: boolean;
  elements: PaginatedResult<DataSourceElement> | undefined;
  fetchingDataSourceElements: boolean;
  elementsFilter: Filter;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenant) {
      if (!elements) {
        dispatch(fetchDataSourceElements(selectedTenant.id, query));
      }
      if (!dataSources) {
        dispatch(fetchDataSources(selectedTenant.id, query));
      }
    }
  }, [selectedTenant, query, elements, dispatch, dataSources]);

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <div>
          <IconFilter />
        </div>
        <Search
          id="dataSourceSchemas"
          columns={dataSourceElementColumns}
          changeSearchFilter={(filter: Filter) => {
            dispatch(changeDataSourceElementsSearchFilter(filter));
          }}
          prefix={dataSourceElementsPrefix}
          searchFilter={elementsFilter}
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
      </div>
      <Card listview>
        {dataSources && elements && elements.data && elements.data.length ? (
          <CardRow>
            <Table spacing="packed" id="objectTypes">
              <TableHead>
                <TableRow>
                  <TableRowHead key="path">Schema Path</TableRowHead>
                  <TableRowHead key="pii">PII</TableRowHead>
                  <TableRowHead key="contents">Contents</TableRowHead>
                  <TableRowHead key="regulations">Regulations</TableRowHead>
                  <TableRowHead key="tags">Tags</TableRowHead>
                  <TableRowHead key="source">Source</TableRowHead>
                  <TableRowHead key="type">Type</TableRowHead>
                  <TableRowHead key="owner">Owner</TableRowHead>
                </TableRow>
              </TableHead>
              <TableBody>
                {elements.data.map((el) => (
                  <TableRow key={el.id} className={PageCommon.listviewtablerow}>
                    <TableCell title={el.id}>
                      <Link
                        key={el.id}
                        href={
                          '/datasourceschemas/' +
                          el.id +
                          makeCleanPageLink(query)
                        }
                      >
                        {el.path}
                      </Link>
                    </TableCell>
                    <TableCell>
                      {el.metadata.contains_pii ? 'Yes' : 'No'}
                    </TableCell>
                    <TableCell>
                      {(el.metadata.contents &&
                        el.metadata.contents.length &&
                        el.metadata.contents.map(
                          (t: keyof typeof UserDataTypes) => (
                            <React.Fragment key={t}>
                              <Tag tag={UserDataTypes[t] || t} />{' '}
                            </React.Fragment>
                          )
                        )) ||
                        '-'}
                    </TableCell>
                    <TableCell>
                      {(el.metadata.regulations &&
                        el.metadata.regulations.length &&
                        el.metadata.regulations.map(
                          (t: keyof typeof Regulations) => (
                            <React.Fragment key={t}>
                              <Tag tag={Regulations[t] || t} />{' '}
                            </React.Fragment>
                          )
                        )) ||
                        '-'}
                    </TableCell>
                    <TableCell>
                      {(el.metadata.tags &&
                        el.metadata.tags.length &&
                        el.metadata.tags.map((t: string) => (
                          <React.Fragment key={t}>
                            <Tag tag={t} />{' '}
                          </React.Fragment>
                        ))) ||
                        '-'}
                    </TableCell>
                    <TableCell>
                      {dataSources.data.find(
                        (source: DataSource) => source.id === el.data_source_id
                      )?.name || el.data_source_id}
                    </TableCell>
                    <TableCell>{el.type}</TableCell>
                    <TableCell>{el.metadata.owner || '-'}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </CardRow>
        ) : fetchingDataSourceElements || fetchingDataSources ? (
          <Text element="h4">Loading...</Text>
        ) : (
          <CardRow>
            <EmptyState title="No schema" />
          </CardRow>
        )}
      </Card>
    </>
  );
};

const ConnectedDataSourceSchemasPage = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    dataSources: state.dataSources,
    fetchingDataSources: state.fetchingDataSources,
    elements: state.dataSourceElements,
    fetchingDataSourceElements: state.fetchingDataSourceElements,
    elementsFilter: state.dataSourceElementsSearchFilter,
    deleteQueue: state.objectTypeDeleteQueue,
    query: state.query,
  };
})(DataSourceSchemasPage);

export default ConnectedDataSourceSchemasPage;
