import { useEffect } from 'react';
import { connect } from 'react-redux';

import { Card } from '@userclouds/ui-component-lib';
import { AppDispatch, RootState } from '../store';

import { ObjectType } from '../models/authz/ObjectType';
import { SelectedTenant } from '../models/Tenant';
import EdgeType from '../models/authz/EdgeType';
import { fetchAuthZObjectTypes, fetchAuthZEdgeTypes } from '../thunks/authz';
import ObjectTable from '../controls/ObjectTable';
import PaginatedResult from '../models/PaginatedResult';

const ObjectsPage = ({
  objectTypes,
  objectTypeError,
  fetchingObjectTypes,
  edgeTypes,
  fetchingEdgeTypes,
  selectedTenant,
  dispatch,
}: {
  objectTypes: PaginatedResult<ObjectType> | undefined;
  objectTypeError: string;
  fetchingObjectTypes: boolean;
  edgeTypes: PaginatedResult<EdgeType> | undefined;
  fetchingEdgeTypes: boolean;
  selectedTenant: SelectedTenant | undefined;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (selectedTenant) {
      if (!objectTypes && !fetchingObjectTypes) {
        dispatch(
          fetchAuthZObjectTypes(selectedTenant.id, new URLSearchParams())
        );
      }
      if (!edgeTypes && !fetchingEdgeTypes) {
        dispatch(fetchAuthZEdgeTypes(selectedTenant.id, new URLSearchParams()));
      }
    }
  }, [
    dispatch,
    objectTypes,
    fetchingObjectTypes,
    edgeTypes,
    fetchingEdgeTypes,
    selectedTenant,
  ]);

  return (
    <Card listview>
      <ObjectTable
        selectedTenant={selectedTenant}
        objectTypes={objectTypes}
        objectTypeError={objectTypeError}
        createButton={false}
      />
    </Card>
  );
};

const ConnectedObjectsPage = connect((state: RootState) => {
  return {
    objectTypes: state.objectTypes,
    objectTypeError: state.fetchObjectTypesError,
    fetchingObjectTypes: state.fetchingObjectTypes,
    edgeTypes: state.edgeTypes,
    fetchingEdgeTypes: state.fetchingEdgeTypes,
    selectedTenant: state.selectedTenant,
  };
})(ObjectsPage);

export default ConnectedObjectsPage;
