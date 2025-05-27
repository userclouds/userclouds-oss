import { AnyAction } from 'redux';

import { RootState } from '../store';
import actions from '../actions';

const serviceInfoReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case actions.GET_SERVICE_INFO_REQUEST:
      break;
    case actions.GET_SERVICE_INFO_SUCCESS:
      state.serviceInfo = action.data;
      break;
    case actions.GET_SERVICE_INFO_ERROR:
      state.serviceInfoError = action.data;
      break;
    default:
      break;
  }
  return state;
};

export default serviceInfoReducer;
