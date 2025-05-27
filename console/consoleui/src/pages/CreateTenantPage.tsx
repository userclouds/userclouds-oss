import { v4 as uuidv4 } from 'uuid';
import React, { useState } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  InlineNotification,
  Label,
  TextInput,
} from '@userclouds/ui-component-lib';

import { redirect } from '../routing';
import { RootState, AppDispatch } from '../store';
import { createTenantRequest, createTenantError } from '../actions/tenants';
import { createTenant } from '../API/tenants';
import { PageTitle } from '../mainlayout/PageWrap';
import { TenantState } from '../models/Tenant';
import PageCommon from './PageCommon.module.css';
import { makeCleanPageLink } from '../AppNavigation';

const handleCreateTenant =
  (name: string, companyID: string, useOrgs: boolean) =>
  async (dispatch: AppDispatch) => {
    if (!name || name.length < 2 || name.length > 30) {
      dispatch(
        createTenantError(
          `Tenant name must be between 2 and 30 characters in length. Current length is ${
            name ? name.length : '0'
          }`
        )
      );
      return;
    }

    dispatch(createTenantRequest());
    return createTenant({
      companyID,
      tenant: {
        id: uuidv4(),
        name,
        company_id: companyID,
        use_organizations: useOrgs,
        state: TenantState.CREATING,
      },
      dispatch,
    });
  };

const CreateTenantPage = ({
  selectedCompanyID,
  isBusy,
  errorMessage,
  query,
  inDialog = false,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  isBusy: boolean;
  errorMessage: string;
  query: URLSearchParams;
  inDialog?: boolean;
  dispatch: AppDispatch;
}) => {
  const [dirty, setDirty] = useState<boolean>(false);
  const cleanQuery = makeCleanPageLink(query);

  return (
    <form
      name="create_tenant"
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();

        const form = e.target as HTMLFormElement;
        const data = new FormData(form);
        const name = data.get('name') as string;
        const useOrgs = data.get('use_orgs') as string;
        dispatch(
          handleCreateTenant(
            name.trim(),
            selectedCompanyID as string,
            useOrgs === 'true'
          )
        ).then(() => {
          if (inDialog) {
            const dialog = form.closest('dialog') as HTMLDialogElement;
            dialog?.close();
          }
        });
      }}
    >
      {!inDialog && (
        <div className={PageCommon.listviewtablecontrols}>
          <PageTitle title="Add a new tenant" itemName="New tenant" />
          <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
            <Button
              size="small"
              theme="secondary"
              onClick={() => {
                if (
                  window.confirm(
                    'Are you sure you want to cancel tenant creation?'
                  )
                ) {
                  redirect(`/${cleanQuery}`);
                }
              }}
              isLoading={isBusy}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              size="small"
              theme="primary"
              disabled={!dirty}
              isLoading={isBusy}
            >
              Create Tenant
            </Button>
          </ButtonGroup>
        </div>
      )}
      {errorMessage && (
        <InlineNotification theme="alert">{errorMessage}</InlineNotification>
      )}
      <Card detailview>
        <CardRow title="Basic details" collapsible>
          <div className={PageCommon.carddetailsrow}>
            <Label>
              Tenant Name
              <br />
              <TextInput
                id="tenant_name"
                name="name"
                required
                placeholder="My first tenant"
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  setDirty(e.target.value.trim().length > 0);
                }}
              />
            </Label>
            <Label>
              Use organizations
              <br />
              <input
                type="checkbox"
                name="use_orgs"
                id="use_orgs"
                value="true"
              />
            </Label>
          </div>
        </CardRow>
      </Card>
    </form>
  );
};

export default connect((state: RootState) => {
  return {
    selectedCompanyID: state.selectedCompanyID,
    isBusy: state.creatingTenant,
    errorMessage: state.createTenantError,
    query: state.query,
  };
})(CreateTenantPage);
