import { AnyAction } from 'redux';
import { RootState } from '../store';
import {
  GET_FEATURE_FLAGS_REQUEST,
  GET_FEATURE_FLAGS_SUCCESS,
  GET_FEATURE_FLAGS_ERROR,
} from '../actions/featureflags';

const featureFlagsReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case GET_FEATURE_FLAGS_REQUEST:
      state.fetchingFeatureFlags = true;
      state.featureFlagsFetchError = '';
      break;
    case GET_FEATURE_FLAGS_SUCCESS: {
      const { feature_gates } = action.data;
      state.featureFlags = feature_gates;
      state.fetchingFeatureFlags = false;
      break;
    }
    case GET_FEATURE_FLAGS_ERROR:
      state.fetchingFeatureFlags = false;
      state.featureFlagsFetchError = action.data;
      break;
    default:
      break;
  }

  return state;
};

export default featureFlagsReducer;
