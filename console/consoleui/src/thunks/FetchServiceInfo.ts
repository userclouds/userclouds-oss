import { APIError } from '@userclouds/sharedui';
import API from '../API';
import actions from '../actions';
import { AppDispatch } from '../store';
import ServiceInfo from '../ServiceInfo';

const fetchServiceInfo = () => (dispatch: AppDispatch) => {
  dispatch({
    type: actions.GET_SERVICE_INFO_REQUEST,
  });
  return API.fetchServiceInfo().then(
    (data: ServiceInfo) => {
      dispatch({
        type: actions.GET_SERVICE_INFO_SUCCESS,
        data,
      });
    },
    (error: APIError) => {
      dispatch({
        type: actions.GET_SERVICE_INFO_ERROR,
        data: error,
      });
    }
  );
};

export default fetchServiceInfo;
