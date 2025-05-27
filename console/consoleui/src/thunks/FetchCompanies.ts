import API from '../API';
import actions from '../actions';
import { AppDispatch } from '../store';
import Company from '../models/Company';

const fetchCompanies = () => (dispatch: AppDispatch) => {
  dispatch({
    type: actions.GET_COMPANIES_REQUEST,
  });
  API.forceFetchCompanies().then(
    (companies: Company[]) => {
      dispatch({
        type: actions.GET_COMPANIES_SUCCESS,
        data: companies,
      });
    },
    (error) => {
      dispatch({
        type: actions.GET_COMPANIES_ERROR,
        data: error.message,
      });
    }
  );
};

export default fetchCompanies;
