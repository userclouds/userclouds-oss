import { useEffect } from 'react';
import { connect } from 'react-redux';

import { APIError } from '@userclouds/sharedui';
import { LoaderDots, InlineNotification } from '@userclouds/ui-component-lib';

import store, { AppDispatch, RootState } from './store';
import { Navigate } from './routing';
import actions from './actions';
import {
  GET_SELECTED_TENANT_REQUEST,
  GET_SELECTED_TENANT_SUCCESS,
  GET_SELECTED_TENANT_ERROR,
} from './actions/tenants';
import API from './API';
import { fetchTenant } from './API/tenants';
import { MyProfile } from './models/UserProfile';
import Company from './models/Company';
import Tenant, {
  SelectedTenant,
  TenantState,
  TenantStateType,
} from './models/Tenant';
import { featureFlagsAreEnabled } from './models/FeatureFlag';
import redirectToLoginOn401 from './RedirectToLogin';
import ServiceInfo from './ServiceInfo';
import { fetchFeatureFlags } from './thunks/featureflags';
import fetchServiceInfo from './thunks/FetchServiceInfo';
import fetchCompanies from './thunks/FetchCompanies';
import { fetchTenants } from './thunks/tenants';
import PageWrap from './mainlayout/PageWrap';
import WelcomePage from './pages/WelcomePage';
import OnboardingPage from './pages/OnboardingPage';

const AppRoutes = ({
  location,
  handler,
}: {
  location: URL;
  handler: Function | undefined;
}) => {
  useEffect(() => {
    setTimeout(() => {
      // TODO: this doesn't work if we go between two history
      // entries via back/forward. Hit back: restore scroll,
      // Hit forward after hitting back: lose scroll.
      // Solution is a map in redux of all page URLs with scroll positions
      const scrollRestore = window.history.state?.scrollY;
      document.getElementById('pageContent')?.scrollTo({
        top: scrollRestore || 0,
        left: 0,
        behavior: 'smooth',
      });
    }, 1);
  }, [location.pathname]);

  if (handler) {
    return handler();
  }
  return <Navigate to="/" replace />;
};

const ConnectedAppRoutes = connect((state: RootState) => ({
  location: state.location,
  handler: state.routeHandler,
}))(AppRoutes);

const fetchMyProfile = (location: URL) => (dispatch: AppDispatch) => {
  dispatch({
    type: actions.GET_MY_PROFILE_REQUEST,
  });
  API.fetchMyProfile().then(
    (myProfile: MyProfile) => {
      dispatch({
        type: actions.GET_MY_PROFILE_SUCCESS,
        data: myProfile,
      });
    },
    (error: APIError) => {
      redirectToLoginOn401(error.message, error.code, location.pathname);
    }
  );
};

// auth-session-id is the cookie set by console API to identify the user
const getAuthSessionId = () => {
  return document.cookie
    .split('; ')
    .filter((row) => row.startsWith('auth-session-id='))
    .map((c) => c.split('=')[1])[0];
};

const fetchProfileWhenSessionChanges =
  (loc: URL) => (dispatch: AppDispatch) => {
    let sessionID = getAuthSessionId();
    dispatch(fetchMyProfile(loc));

    return setInterval(() => {
      const newID = getAuthSessionId();
      if (newID !== sessionID) {
        sessionID = newID;
        dispatch(fetchMyProfile(loc));
      }
    }, 30000);
  };

const startupDataIsLoaded = () => {
  const {
    companies,
    myProfile,
    serviceInfo,
    selectedCompanyID,
    featureFlags,
    featureFlagsFetchError,
  } = store.getState();
  return (
    companies &&
    myProfile &&
    serviceInfo &&
    selectedCompanyID &&
    (featureFlags || featureFlagsFetchError)
  );
};

const App = ({
  myProfile,
  selectedCompanyID,
  selectedCompany,
  serviceInfo,
  serviceInfoError,
  companies,
  fetchingCompanyError,
  selectedTenantID,
  fetchingSelectedTenant,
  fetchingTenantError,
  tenants,
  location,
  query,
  dispatch,
  tenantCreationState,
}: {
  myProfile: MyProfile | undefined;
  selectedCompanyID: string | undefined;
  selectedCompany: Company | undefined;
  serviceInfo: ServiceInfo | undefined;
  serviceInfoError: string | undefined;
  companies: Company[] | undefined;
  fetchingCompanyError: string | undefined;
  selectedTenantID: string | undefined;
  fetchingSelectedTenant: boolean;
  fetchingTenantError: string | undefined;
  tenants: Tenant[] | undefined;
  location: URL;
  query: URLSearchParams;
  dispatch: AppDispatch;
  tenantCreationState: TenantStateType | null;
}) => {
  useEffect(() => {
    if (
      companies !== undefined &&
      (!selectedCompanyID ||
        (query.get('company_id') &&
          selectedCompanyID !== query.get('company_id')))
    ) {
      const matchingCompany = companies?.find(
        (c: Company) => c.id === query.get('company_id')
      );
      if (matchingCompany) {
        dispatch({
          type: actions.CHANGE_SELECTED_COMPANY,
          data: query.get('company_id'),
        });
      } else if (companies?.length && !selectedCompanyID) {
        dispatch({
          type: actions.CHANGE_SELECTED_COMPANY,
          data: companies[0].id,
        });
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCompanyID, companies, query]);

  useEffect(() => {
    if (
      query.get('company_id') &&
      query.get('tenant_id') &&
      selectedTenantID !== query.get('tenant_id') &&
      tenants !== undefined &&
      !fetchingSelectedTenant
    ) {
      if (tenants?.length) {
        const matchingTenant = tenants.find(
          (t: Tenant) => t.id === query.get('tenant_id')
        );
        if (matchingTenant) {
          dispatch({
            type: GET_SELECTED_TENANT_REQUEST,
          });
          fetchTenant(query.get('company_id')!, query.get('tenant_id')!)
            .then((selectedTenant: SelectedTenant) => {
              dispatch({
                type: GET_SELECTED_TENANT_SUCCESS,
                data: selectedTenant,
              });
            })
            .catch((error: APIError) => {
              dispatch({
                type: GET_SELECTED_TENANT_ERROR,
                error: error,
              });
            });
        }
      }
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedTenantID, tenants, query]);

  useEffect(() => {
    dispatch(fetchProfileWhenSessionChanges(location));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (
      !fetchingSelectedTenant &&
      selectedCompanyID &&
      myProfile &&
      (selectedTenantID || tenants !== undefined)
    ) {
      if (featureFlagsAreEnabled()) {
        dispatch(
          fetchFeatureFlags(
            myProfile.userProfile.id,
            selectedCompanyID,
            selectedTenantID || '' // we re-fetch feature flags when our tenant changes but we still want flags for no tenant selected
          )
        );
      }
    }
  }, [
    fetchingSelectedTenant,
    selectedCompanyID,
    selectedTenantID,
    tenants,
    myProfile,
    dispatch,
  ]);

  // Set universe in document title once at start
  useEffect(() => {
    if (!serviceInfo) {
      dispatch(fetchServiceInfo());
    } else if (!serviceInfo.is_production) {
      document.title = `[${serviceInfo.environment}] UserClouds Console`;
    }
  }, [serviceInfo, dispatch]);
  useEffect(() => {
    if (!selectedCompany) {
      dispatch(fetchCompanies());
    } else {
      dispatch(
        fetchTenants(
          selectedCompany.id,
          selectedTenantID,
          location.pathname,
          query
        )
      );
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedCompany, dispatch]);

  if (tenants?.length === 0 && tenantCreationState === TenantState.CREATING) {
    return <OnboardingPage />;
  }

  if (myProfile && companies?.length === 0) {
    return (
      <WelcomePage
        name={myProfile.userProfile.name()}
        userID={myProfile.userProfile.id}
      />
    );
  }

  if (startupDataIsLoaded()) {
    return (
      <PageWrap>
        <ConnectedAppRoutes />
      </PageWrap>
    );
  }

  return (
    <PageWrap>
      {fetchingTenantError || fetchingCompanyError || serviceInfoError ? (
        <>
          {fetchingCompanyError && (
            <InlineNotification theme="alert">
              {fetchingCompanyError}
            </InlineNotification>
          )}
          {fetchingTenantError && (
            <InlineNotification theme="alert">
              {fetchingTenantError}
            </InlineNotification>
          )}
          {serviceInfoError && (
            <InlineNotification theme="alert">
              {serviceInfoError}
            </InlineNotification>
          )}
        </>
      ) : (
        <LoaderDots assistiveText="Loading ..." size="small" theme="brand" />
      )}
    </PageWrap>
  );
};

export default connect((state: RootState) => {
  return {
    myProfile: state.myProfile,
    selectedCompanyID: state.selectedCompanyID,
    selectedCompany: state.selectedCompany,
    serviceInfo: state.serviceInfo,
    serviceInfoError: state.serviceInfoError,
    companies: state.companies,
    fetchingCompanies: state.fetchingCompanies,
    fetchingCompanyError: state.companiesFetchError,
    selectedTenantID: state.selectedTenantID,
    fetchingSelectedTenant: state.fetchingSelectedTenant,
    tenants: state.tenants,
    fetchingTenants: state.fetchingTenants,
    fetchingTenantError: state.tenantFetchError,
    tenantCreationState: state.tenantCreationState,
    // TODO: for some reason, the app fails to load if this isn't
    // imported, even though we don't actually use it in the component
    featureFlags: state.featureFlags,
    location: state.location,
    query: state.query,
  };
})(App);
