import { connect } from 'react-redux';

import { RootState } from '../store';

import { SelectedTenant } from '../models/Tenant';

import EdgeTable from '../controls/EdgeTable';

const EdgesPage = ({
  selectedTenant,
}: {
  selectedTenant: SelectedTenant | undefined;
}) => {
  return <EdgeTable selectedTenant={selectedTenant} createButton />;
};

const ConnectedEdgesPage = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
  };
})(EdgesPage);

export default ConnectedEdgesPage;
