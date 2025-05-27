import { AnyAction } from 'redux';
import {
  RootState,
  initialCompanyPersistedState,
  initialCompanyAppState,
  initialCompanyPageState,
  initialTenantPersistedState,
} from '../store';
import actions from '../actions';
import Company from '../models/Company';
import { getNewToggleEditValue } from './reducerHelper';

const companiesReducer = (state: RootState, action: AnyAction) => {
  switch (action.type) {
    case actions.GET_ALL_COMPANIES_REQUEST:
      state.fetchingAllCompanies = true;
      break;
    case actions.GET_ALL_COMPANIES_SUCCESS:
      state.fetchingAllCompanies = false;
      state.globalCompanies = action.data;
      break;
    case actions.GET_ALL_COMPANIES_ERROR:
      state.fetchingAllCompanies = false;
      state.companiesFetchError = action.data;
      break;
    case actions.CREATE_COMPANY_REQUEST:
      state.creatingCompany = true;
      state.createCompanyError = '';
      break;
    case actions.CREATE_COMPANY_SUCCESS: {
      state.creatingCompany = false;
      Object.assign(
        state,
        initialCompanyPersistedState,
        initialCompanyAppState(),
        initialTenantPersistedState
      );
      const company: Company = action.data;
      // TODO: is it better to re-fetch than add? probably
      state.companies = [company, ...(state.companies as Company[])];
      state.selectedCompanyID = company.id;
      state.selectedCompany = company;
      state.createCompanyDialogIsOpen = false;
      break;
    }
    case actions.CREATE_COMPANY_ERROR:
      state.creatingCompany = false;
      state.createCompanyError = `Error creating company: ${
        action.data as string
      }`;
      break;
    case actions.CHANGE_SELECTED_COMPANY:
      Object.assign(
        state,
        initialCompanyPersistedState,
        initialCompanyAppState(),
        initialCompanyPageState,
        initialTenantPersistedState
      );
      state.selectedCompanyID = action.data;
      state.companyFetchError = '';
      if (state.companies) {
        const matchingCompany = state.companies.find(
          (t: Company) => t.id === action.data
        );
        if (matchingCompany) {
          state.selectedCompany = matchingCompany;
        } else if (state.companies.length) {
          state.selectedCompany = state.companies[0];
          state.selectedCompanyID = state.companies[0].id;
        } else {
          state.selectedCompany = undefined;
          state.selectedCompanyID = undefined;
        }
      }
      state.changeCompanyDialogIsOpen = false;
      break;
    case actions.GET_COMPANIES_REQUEST:
      state.fetchingCompanies = true;
      state.companies = undefined;
      break;
    case actions.GET_COMPANIES_SUCCESS:
      state.fetchingCompanies = false;
      state.companies = action.data || [];
      break;
    case actions.GET_COMPANIES_ERROR:
      state.fetchingCompanies = false;
      state.companyFetchError = action.data;
      break;
    case actions.TOGGLE_CREATE_COMPANY_DIALOG:
      state.createCompanyDialogIsOpen = getNewToggleEditValue(
        action.data,
        state.createCompanyDialogIsOpen
      );
      state.changeCompanyDialogIsOpen = false;
      break;
    case actions.TOGGLE_CHANGE_COMPANY_DIALOG:
      state.changeCompanyDialogIsOpen = getNewToggleEditValue(
        action.data,
        state.changeCompanyDialogIsOpen
      );
      break;
    default:
      break;
  }
  return state;
};

export default companiesReducer;
