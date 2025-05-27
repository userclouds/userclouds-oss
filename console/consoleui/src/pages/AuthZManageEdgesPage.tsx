import { connect } from 'react-redux';

import { PageTitle } from '../mainlayout/PageWrap';
import { RootState } from '../store';

import { SelectedTenant } from '../models/Tenant';

import EdgeTable from '../controls/EdgeTable';
import Breadcrumbs from '../controls/Breadcrumbs';

const AuthZManageEdgesPage = ({
  selectedTenant,
}: {
  selectedTenant: SelectedTenant | undefined;
}) => {
  return (
    <>
      <Breadcrumbs />

      <PageTitle
        title="Manage Edges"
        description="Edges reflect real-world relationships between two objects in your authorization graph."
      />
      <EdgeTable selectedTenant={selectedTenant} createButton />
    </>
  );
};

const ConnectedManageAuthZEdgesPage = connect((state: RootState) => {
  return {
    edge: state.currentEdgeType,
    objectTypes: state.objectTypes,
    objectTypeError: state.fetchObjectTypesError,
    edgeTypes: state.edgeTypes,
    edgeTypeError: state.fetchEdgeTypesError,
    selectedTenant: state.selectedTenant,
    edgeTypeEditMode: state.edgeTypeEditMode,
    edgeTypeDeleteQueue: state.edgeTypeDeleteQueue,
    objectTypeEditMode: state.objectTypeEditMode,
    objectTypeDeleteQueue: state.objectTypeDeleteQueue,
  };
})(AuthZManageEdgesPage);

export default ConnectedManageAuthZEdgesPage;
