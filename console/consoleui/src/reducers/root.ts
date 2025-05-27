import { AnyAction } from 'redux';
import { RootState } from '../store';
import navigationReducer from './navigation';
import featureFlagsReducer from './features';
import serviceInfoReducer from './serviceinfo';
import companiesReducer from './companies';
import tenantsReducer from './tenants';
import orgsReducer from './orgs';
import authnReducer from './authn';
import authzReducer from './authz';
import datamappingReducer from './datamapping';
import userStoreReducer from './userstore';
import usersReducer from './users';
import teamReducer from './team';
import tokenizerReducer from './tokenizer';
import auditLogReducer from './auditlog';
import statusPageReducer from './statuspage';
import tenantKeysReducer from './tenantkeys';
import systemLogReducer from './systemlog';
import tagReducer from './tags';
import actions from '../actions';
import {
  POST_NEW_NOTIFICATION,
  REMOVE_POSTED_NOTIFICATION,
} from '../actions/notifications';
import Notification from '../models/Notification';

type SubReducer = (state: RootState, action: AnyAction) => RootState;

const subReducers: SubReducer[] = [
  navigationReducer,
  featureFlagsReducer,
  serviceInfoReducer,
  companiesReducer,
  tenantsReducer,
  orgsReducer,
  authnReducer,
  authzReducer,
  datamappingReducer,
  userStoreReducer,
  usersReducer,
  teamReducer,
  tokenizerReducer,
  auditLogReducer,
  statusPageReducer,
  tenantKeysReducer,
  systemLogReducer,
  tagReducer,
];
const rootReducer = (state: RootState | undefined, action: AnyAction) => {
  // NB:ksj: `combineReducers` is the canonical approach to splitting up
  // large reducers. A feature/bug of this approach is that it also splits up
  // the store itself: you have separate slices of the store, with their
  // own namespaces. If an action in one part of the store might have an effect
  // elsewhere (say, deleting an company means deleting its tenants, etc), you're
  // in an awkward position.
  // This little bit below just runs through switch statements in series.
  // The switches should be mutually exclusive (i.e. none should reference
  // an action referenced by another).
  // TODO: this function does nothing to check that sub-reducers are anatomically
  // correct or avoid these clashes.
  // We can check out https://github.com/redux-utilities/reduce-reducers,
  // but I'm not sure it gets us any further.
  // See also: https://stackoverflow.com/questions/38652789/correct-usage-of-reduce-reducers/44371190#44371190
  state = subReducers.reduce((newState: RootState, reducer: SubReducer) => {
    newState = reducer(newState, action);

    return newState;
  }, state as RootState);

  switch (action.type) {
    case actions.GET_MY_PROFILE_SUCCESS:
      state.myProfile = action.data;
      break;

    // Add global actions here.
    // e.g.: we move past React Router and fire a single
    // action on navigation/pageload/forward/back
    case POST_NEW_NOTIFICATION:
      state.notifications = [...state.notifications, action.data];
      break;
    case REMOVE_POSTED_NOTIFICATION:
      state.notifications = state.notifications.filter(
        (notification: Notification) => notification.id !== action.data
      );
      break;
    default:
      break;
  }
  return { ...state };
};

export default rootReducer;
