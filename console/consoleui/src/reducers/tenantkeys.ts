import { AnyAction } from 'redux';

import { RootState } from '../store';
import {
  GET_TENANT_KEYS_REQUEST,
  GET_TENANT_KEYS_SUCCESS,
  GET_TENANT_KEYS_ERROR,
  ROTATE_TENANT_KEYS_REQUEST,
  ROTATE_TENANT_KEYS_SUCCESS,
  ROTATE_TENANT_KEYS_ERROR,
} from '../actions/keys';

const tenantKeysReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case GET_TENANT_KEYS_REQUEST:
      state.tenantPublicKey = '';
      state.fetchingPublicKeys = true;
      state.fetchTenantPublicKeysError = '';
      break;
    case GET_TENANT_KEYS_SUCCESS:
      state.fetchingPublicKeys = false;
      if (action.data?.length) {
        state.tenantPublicKey = action.data[0];
      }
      break;
    case GET_TENANT_KEYS_ERROR:
      state.fetchTenantPublicKeysError = action.data;
      state.fetchingPublicKeys = false;
      break;
    case ROTATE_TENANT_KEYS_REQUEST:
      state.rotatingTenantKeys = true;
      state.rotateTenantKeysError = '';
      break;
    case ROTATE_TENANT_KEYS_SUCCESS:
      state.rotatingTenantKeys = false;
      break;
    case ROTATE_TENANT_KEYS_ERROR:
      state.rotatingTenantKeys = false;
      state.rotateTenantKeysError = action.data;
      break;
    default:
      break;
  }
  return state;
};

export default tenantKeysReducer;
