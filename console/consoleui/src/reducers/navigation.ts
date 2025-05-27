import { AnyAction } from 'redux';
import {
  RootState,
  initialCompanyPageState,
  initialTenantPageState,
} from '../store';
import { NAVIGATE } from '../actions/routing';

const navigationReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case NAVIGATE: {
      const { location, handler, pattern, params } = action.data;
      // reset ephemeral state on every navigation
      // if query string doesn't change, treat it like a refresh
      // UNLESS hash DOES change
      if (
        !(
          location.search === state.location.search &&
          location.hash !== state.location.hash
        )
      ) {
        Object.assign(state, initialCompanyPageState, initialTenantPageState);
        // React will treat this as a change, even if querystring has not changed
        state.query = new URLSearchParams(location.search);
      }
      state.routeParams = params;
      state.routeHandler = handler;
      state.routePattern = pattern;

      state.location = location;
      break;
    }
    default:
      break;
  }

  return state;
};

export default navigationReducer;
