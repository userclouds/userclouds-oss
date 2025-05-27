import { v4 as uuidv4 } from 'uuid';
import { useState } from 'react';
import { connect } from 'react-redux';

import {
  Card,
  CardRow,
  Button,
  ButtonGroup,
  InlineNotification,
  Label,
  TextInput,
  Select,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import { redirect } from '../routing';
import { RootState, AppDispatch } from '../store';
import Organization from '../models/Organization';
import Region, { DefaultRegion } from '../models/Region';
import {
  createOrganizationRequest,
  createOrganizationSuccess,
  createOrganizationError,
} from '../actions/organizations';
import { createOrganization } from '../API/organizations';
import { postSuccessToast } from '../thunks/notifications';
import { PageTitle } from '../mainlayout/PageWrap';
import PageCommon from './PageCommon.module.css';

const createOrg =
  (
    companyID: string | undefined,
    tenantID: string | undefined,
    data: FormData
  ) =>
  (dispatch: AppDispatch) => {
    if (companyID && tenantID) {
      const id = data.get('organization_id') as string;
      const name = data.get('organization_name') as string;
      const region = data.get('organization_region') as Region;
      dispatch(createOrganizationRequest());
      createOrganization(tenantID, id, name, region).then(
        (result: Organization) => {
          dispatch(createOrganizationSuccess(result));
          dispatch(
            postSuccessToast(
              `Organization "${result.name}" successfully created`
            )
          );
          redirect(
            `/organizations/${result.id}?company_id${companyID}&tenant_id=${tenantID}`
          );
        },
        (error: APIError) => {
          dispatch(createOrganizationError(error));
        }
      );
    }
  };

const CreateOrgPage = ({
  selectedCompanyID,
  selectedTenantID,
  isSaving,
  saveError,
  newOrgID,
  dispatch,
}: {
  selectedCompanyID: string | undefined;
  selectedTenantID: string | undefined;
  isSaving: boolean;
  saveError: string;
  newOrgID: string;
  dispatch: AppDispatch;
}) => {
  const [disabled, setDisabled] = useState<boolean>(true);

  return (
    <form
      onSubmit={(e: React.FormEvent) => {
        e.preventDefault();

        const data = new FormData(e.target as HTMLFormElement);
        dispatch(createOrg(selectedCompanyID, selectedTenantID, data));
      }}
    >
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle title="Create Organization" itemName="New Organization" />

        <ButtonGroup className={PageCommon.listviewtablecontrolsButtonGroup}>
          <Button
            type="submit"
            theme="primary"
            size="small"
            isLoading={isSaving}
            disabled={isSaving || disabled}
          >
            Create Organization
          </Button>
        </ButtonGroup>
      </div>

      <Card detailview>
        <CardRow title="Basic Details" collapsible>
          {saveError ? (
            <InlineNotification theme="alert">{saveError}</InlineNotification>
          ) : (
            ''
          )}
          <input type="hidden" name="organization_id" value={newOrgID} />
          <Label>
            Name
            <br />
            <TextInput
              name="organization_name"
              type="text"
              required
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                const val = e.target.value;
                setDisabled(val.trim().length === 0);
              }}
            />
          </Label>
          <Label>
            Preferred Region
            <br />
            <Select name="organization_region" defaultValue={DefaultRegion}>
              {Object.values(Region).map((region: Region) => (
                <option value={region} key={region}>
                  {region}
                </option>
              ))}
            </Select>
          </Label>
        </CardRow>
      </Card>
    </form>
  );
};

export default connect((state: RootState) => ({
  selectedCompanyID: state.selectedCompanyID,
  selectedTenantID: state.selectedTenantID,
  isSaving: state.savingOrganization,
  saveError: state.createOrganizationError,
  newOrgID: uuidv4(),
  location: state.location,
  query: state.query,
}))(CreateOrgPage);
