import { connect } from 'react-redux';
import { Navigate } from './routing';
import { RootState } from './store';
import ServiceInfo from './ServiceInfo';

const UCAdminOnlyRoute = ({
  serviceInfo,
  children,
}: {
  serviceInfo: ServiceInfo | undefined;
  children: JSX.Element;
}) => {
  if (serviceInfo && !serviceInfo.uc_admin) {
    return <Navigate to="/" replace />;
  }
  return children;
};
export default connect((state: RootState) => ({
  serviceInfo: state.serviceInfo,
}))(UCAdminOnlyRoute);
