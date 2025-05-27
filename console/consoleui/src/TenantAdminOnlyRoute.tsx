import { connect } from 'react-redux';

import { Navigate } from './routing';
import { AppDispatch, RootState } from './store';
import { SelectedTenant } from './models/Tenant';
import { postAlertToast } from './thunks/notifications';

const TenantAdminOnlyRoute = ({
  children,
  selectedTenant,
  fetchingTenant,
  tenantFetchError,
  dispatch,
}: {
  children: JSX.Element;
  selectedTenant: SelectedTenant | undefined;
  fetchingTenant: boolean;
  tenantFetchError: string;
  dispatch: AppDispatch;
}) => {
  if (fetchingTenant || (!tenantFetchError && !selectedTenant)) {
    return <>Fetching tenant...</>;
  }
  if (selectedTenant && !selectedTenant.is_admin) {
    dispatch(postAlertToast('Admin only page. Renavigating to home.'));
    return <Navigate to="/" replace />;
  }
  return children;
};

export default connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    fetchingTenant: state.fetchingSelectedTenant,
    tenantFetchError: state.tenantFetchError,
  };
})(TenantAdminOnlyRoute);
