import React, { useEffect, useState } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  GlobalStyles,
  Label,
  Select,
  Text,
} from '@userclouds/ui-component-lib';

import { RootState, AppDispatch } from '../store';
import actions from '../actions';
import API from '../API';
import Company from '../models/Company';
import { PageTitle } from '../mainlayout/PageWrap';
import { CompanyMemberList } from './IAMMemberLists';
import Breadcrumbs from '../controls/Breadcrumbs';

const ConnectedCompanyMemberList = connect((state: RootState) => {
  return {
    editMode: state.companyUserRolesEditMode,
    isSaving: state.savingCompanyUserRoles,
    teamUserRoles: state.companyUserRoles,
    isFetching: state.fetchingCompanyUserRoles,
    fetchError: state.fetchCompanyUserRolesError,
    deleteQueue: state.companyUserRolesDeleteQueue,
    modifiedUserRoles: state.modifiedCompanyUserRoles,
    bulkSaveErrors: state.companyUserRolesBulkSaveErrors,
    userID: state.myProfile?.userProfile.id,
  };
})(CompanyMemberList);

const AllCompanyMemberLists = ({
  companies,
  editMode,
  error,
  dispatch,
}: {
  companies: Company[] | undefined;
  editMode: boolean;
  error: string;
  dispatch: AppDispatch;
}) => {
  const [currentCompanyID, setCurrentCompanyID] = useState<string>();
  useEffect(() => {
    if (!companies) {
      dispatch({
        type: actions.GET_ALL_COMPANIES_REQUEST,
      });
      API.fetchAllCompanies().then(
        (allCompanies) => {
          dispatch({
            type: actions.GET_ALL_COMPANIES_SUCCESS,
            data: allCompanies,
          });
        },
        (err) => {
          // Don='t set companies to an array on error otherwise it'll retry infinitely since
          // the state changes each time.
          dispatch({
            type: actions.GET_ALL_COMPANIES_ERROR,
            data: err.message,
          });
        }
      );
    } else if (companies.length) {
      setCurrentCompanyID(companies[0].id);
    }
  }, [companies, dispatch]);

  if (error) {
    return <div>{error}</div>;
  }

  const company = companies?.find(
    (comp: Company) => comp.id === currentCompanyID
  );
  return (
    <>
      <Card title="View company users">
        {companies && companies.length ? (
          <Label>
            Select a company to view users:
            <br className={GlobalStyles['mb-3']} />
            <Select
              onChange={(e: React.ChangeEvent) => {
                const val = (e.target as HTMLSelectElement).value;
                setCurrentCompanyID(val);
              }}
            >
              {companies.map((c) => (
                <option key={`company_${c.id}`} value={c.id}>
                  {c.name}
                </option>
              ))}
            </Select>
          </Label>
        ) : (
          'No companies found'
        )}
      </Card>
      <Card
        title="Manage Company Roles"
        description={`Manage roles for teammates at ${
          company ? company.name : '...'
        }`}
        isDirty={editMode}
      >
        {currentCompanyID ? (
          <ConnectedCompanyMemberList companyID={currentCompanyID} />
        ) : (
          <Text>No company selected</Text>
        )}
      </Card>
    </>
  );
};

const ConnectedLists = connect((state: RootState) => {
  return {
    companies: state.globalCompanies,
    editMode: state.companyUserRolesEditMode,
    error: state.companiesFetchError,
  };
})(AllCompanyMemberLists);

const GlobalIAMPage = () => {
  return (
    <>
      <Breadcrumbs />
      <PageTitle
        title="[dev] Global IAM"
        description="[dev only] Manage roles across all companies"
      />
      <ConnectedLists />
    </>
  );
};

export default GlobalIAMPage;
