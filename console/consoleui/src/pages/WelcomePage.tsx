import { v4 as uuidv4 } from 'uuid';
import React, { useState } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  Checkbox,
  InlineNotification,
  Label,
  TextInput,
  Select,
  Accordion,
  AccordionItem,
  HorizontalRule,
  LoaderDots,
} from '@userclouds/ui-component-lib';

import API from '../API';
import { createTenant } from '../API/tenants';
import fetchServiceInfo from '../thunks/FetchServiceInfo';
import actions from '../actions';
import { AppDispatch } from '../store';
import Company, { CompanyType, DefaultCompanyType } from '../models/Company';
import { SelectedTenant, TenantState } from '../models/Tenant';
import { GetLogoutURL } from '../Auth';
import WelcomePageStyles from './WelcomePage.module.css';

type CompanyAndTenant = {
  company: Company;
  tenant: SelectedTenant;
};

// Helper method that returns either `CompanyAndTenant` (on success) or a string error message string (on failure).
const createCompanyAndTenant =
  (
    companyName: string,
    companyType: CompanyType,
    tenantName: string,
    useOrgs: boolean
  ) =>
  (dispatch: AppDispatch): Promise<CompanyAndTenant> => {
    return new Promise((resolve, reject) => {
      let createdCompany: Company | undefined;
      let createdTenant: SelectedTenant | undefined;
      const companyID = uuidv4();

      dispatch({
        type: actions.CREATE_COMPANY_REQUEST,
      });
      return API.createCompany({
        id: companyID,
        name: companyName,
        type: companyType,
        is_admin: true,
      })
        .then(
          (company: Company) => {
            createdCompany = company;
            dispatch({
              type: actions.CREATE_COMPANY_SUCCESS,
              data: company,
            });
            // https://github.com/userclouds/userclouds/issues/904
            //
            // serviceinfo response includes list of companies for which
            // user is an admin. We need to refetch it after a new tenant creation
            dispatch(fetchServiceInfo());
            return createTenant({
              companyID: company.id,
              tenant: {
                id: uuidv4(),
                name: tenantName,
                company_id: company.id,
                use_organizations: useOrgs,
                state: TenantState.CREATING,
              },
              dispatch,
            });
          },
          (error) => {
            reject(`Error creating company ${companyName}: ${error.message}`);
          }
        )
        .then(
          (tenant: SelectedTenant | void) => {
            if (tenant) {
              createdTenant = tenant;
              resolve({
                company: createdCompany as Company,
                tenant: createdTenant,
              });
            } else {
              reject(
                `Company ${
                  (createdCompany as Company).name
                } created but an error occurred while creating tenant ${tenantName}`
              );
            }
          },
          (error) => {
            reject(
              `Company ${
                (createdCompany as Company).name
              } created but an error occurred while creating tenant ${tenantName}: ${
                error.message
              }`
            );
          }
        );
    });
  };

const WelcomePage = ({
  name,
  userID,
  dispatch,
}: {
  name: string;
  userID: string;
  dispatch: AppDispatch;
}) => {
  const heading = `Welcome to UserClouds, ${name}!`;
  const subHeading = `Let's get you set up with a new company.`;
  const goButtonText = `Let's go!`;
  const notYouButtonText = 'Not You?';
  const logoutButtonText = 'Logout';
  const [companyName, setCompanyName] = useState<string>();
  const [companyType, setCompanyType] = useState<CompanyType>();
  const [tenantName, setTenantName] = useState<string>();
  const [useOrgs, setUseOrgs] = useState<boolean>(false);
  const [errorMessage, setErrorMessage] = useState<string>('');
  const [statusError, setStatusError] = useState<boolean>(false);
  const [isLoading, setIsLoading] = useState<boolean>(false);

  const inputInvalid = !companyName || !tenantName;
  const disallowSubmit = inputInvalid || errorMessage !== '' || isLoading;

  const onSubmit = async () => {
    if (disallowSubmit) {
      return;
    }

    setIsLoading(true);
    setStatusError(false);

    // Create company, then tenant, then force re-fetch all companies to ensure UI has latest data & redirect to new tenant.
    dispatch(
      createCompanyAndTenant(
        companyName,
        companyType || DefaultCompanyType,
        tenantName,
        useOrgs
      )
    ).catch((error: string) => {
      setIsLoading(false);
      setErrorMessage(error);
      setStatusError(true);
    });
  };

  return (
    <div className={WelcomePageStyles.welcomepage}>
      <div>
        <Card
          title={heading}
          description={subHeading}
          collapsible={false}
          className={WelcomePageStyles.card}
        >
          <Label className={WelcomePageStyles.label}>
            Company Name
            <br />
            <TextInput
              id="company_name"
              name="company_name"
              value={companyName || ''}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                const val = e.target.value;
                setCompanyName(val);
              }}
            />
          </Label>

          <Label className={WelcomePageStyles.label}>
            Tenant Name
            <br />
            <TextInput
              label="Tenant"
              id="tenant_name"
              name="tenant_name"
              value={tenantName || ''}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                const val = e.target.value;
                setTenantName(val);
              }}
            />
          </Label>

          <div className={WelcomePageStyles.accordion}>
            <Accordion>
              <AccordionItem title="Advanced">
                <Label className={WelcomePageStyles.label}>
                  Company Type
                  <br />
                  <Select
                    name="company_type"
                    defaultValue="internal"
                    full
                    onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                      const val = e.target.value;
                      setCompanyType(val as CompanyType);
                    }}
                  >
                    {Object.values(CompanyType).map((ct: CompanyType) => (
                      <option value={ct} key={ct}>
                        {ct}
                      </option>
                    ))}
                  </Select>
                </Label>

                <Checkbox
                  checked={useOrgs}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    const { checked } = e.target;
                    setUseOrgs(checked);
                  }}
                >
                  Use organizations
                </Checkbox>
              </AccordionItem>
            </Accordion>
            <HorizontalRule className={WelcomePageStyles.horizontalRule} />
          </div>

          {statusError ? (
            <InlineNotification theme={statusError ? 'alert' : 'info'}>
              {errorMessage}
            </InlineNotification>
          ) : (
            ''
          )}

          <div className={WelcomePageStyles.buttonContainer}>
            <Button disabled={disallowSubmit} onClick={onSubmit}>
              {isLoading ? (
                <LoaderDots
                  size="small"
                  theme="inverse"
                  assistiveText="Creating tenant…"
                />
              ) : (
                goButtonText
              )}
            </Button>
            <div className={WelcomePageStyles.logoutLink}>
              <span>{notYouButtonText}</span>
              <a
                className={WelcomePageStyles.notyoulink}
                href={GetLogoutURL('/')}
              >
                {logoutButtonText}
              </a>
            </div>
          </div>
        </Card>
        <div className={WelcomePageStyles.userid}>{
          // TODO: get rid of userID here; it's useful now to figure out your UUID
          // so you can make yourself an company admin as needed via make-company-admin.sh
          `Your user ID is ${userID}`
        }</div>
      </div>
    </div>
  );
};

export default connect()(WelcomePage);
