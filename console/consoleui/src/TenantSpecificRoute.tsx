import { connect } from 'react-redux';

import { RootState } from './store';
import { SelectedTenant } from './models/Tenant';

const TenantSpecificRoute = ({
  children,
  selectedTenant,
  fetchingTenant,
  tenantFetchError,
}: {
  children: JSX.Element;
  selectedTenant: SelectedTenant | undefined;
  fetchingTenant: boolean;
  tenantFetchError: string;
}) => {
  if (fetchingTenant || (!tenantFetchError && !selectedTenant)) {
    return <>Fetching tenant...</>;
  }
  if (!selectedTenant || !selectedTenant.is_member) {
    return <>Access denied </>;
  }
  return children;
};

export default connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    fetchingTenant: state.fetchingSelectedTenant,
    tenantFetchError: state.tenantFetchError,
  };
})(TenantSpecificRoute);
