import { v4 as uuidv4 } from 'uuid';
import { connect } from 'react-redux';
import {
  Button,
  Dialog,
  IconManageTeam,
  InlineNotification,
  Label,
  PseudoSelect,
  Select,
  TextInput,
  GlobalStyles,
  DialogFooter,
  DialogBody,
} from '@userclouds/ui-component-lib';

import { AppDispatch, RootState } from '../store';
import actions from '../actions';
import { redirect, TENANTS_CREATE_PATH } from '../routing';
import API from '../API';
import fetchServiceInfo from '../thunks/FetchServiceInfo';
import { MyProfile } from '../models/UserProfile';
import Company, { CompanyType, DefaultCompanyType } from '../models/Company';
import Tenant from '../models/Tenant';
import ServiceInfo from '../ServiceInfo';
import ProfileWidget from './ProfileWidget';
import Breadcrumbs from '../controls/Breadcrumbs';
import Link from '../controls/Link';
import { makeCleanPageLink } from '../AppNavigation';
import CreateTenantPage from '../pages/CreateTenantPage';

import HeaderStyles from './Header.module.css';
import PageCommon from '../pages/PageCommon.module.css';

const createCompany =
  (
    data: FormData,
    form: HTMLFormElement | null,
    dialog: HTMLDialogElement | null
  ) =>
  (dispatch: AppDispatch) => {
    let companyName = data.get('company_name');
    let companyType = data.get('company_type');
    if (!companyName) {
      return;
    }
    companyName = String(companyName);

    if (!companyType) {
      return;
    }
    companyType = String(companyType);

    dispatch({
      type: actions.CREATE_COMPANY_REQUEST,
    });
    return API.createCompany({
      id: uuidv4(),
      name: companyName,
      type: companyType as CompanyType,
      is_admin: true,
    }).then(
      (company: Company) => {
        // serviceinfo response includes list of companies for which
        // user is an admin. We need to refetch it after a new tenant creation
        dispatch(fetchServiceInfo()).then(() => {
          redirect(`/tenants/create?company_id=${company.id}`);
          dispatch({
            type: actions.CREATE_COMPANY_SUCCESS,
            data: company,
          });
          if (form) {
            form.reset();
          }
          if (dialog) {
            dialog.close();
          }
        });
      },
      (error) => {
        dispatch({
          type: actions.CREATE_COMPANY_ERROR,
          data: error.message,
        });
      }
    );
  };

const CreateCompanyDialog = connect((state: RootState) => ({
  isOpen: state.createCompanyDialogIsOpen,
  error: state.createCompanyError,
}))(({
  isOpen,
  error,
  dispatch,
}: {
  isOpen: boolean;
  error: string;
  dispatch: AppDispatch;
}) => {
  return (
    <Dialog id="createCompanyDialog" title="Create Company" open={isOpen}>
      {isOpen && (
        <form
          action="/api/companies"
          method="POST"
          onSubmit={(e: React.FormEvent<HTMLFormElement>) => {
            e.preventDefault();
            const form = e.target as HTMLFormElement;
            const formData = new FormData(form);
            const dialog = form.closest('dialog');
            dispatch(createCompany(formData, form, dialog));
          }}
        >
          <DialogBody>
            <Label>
              Company name:
              <br />
              <TextInput
                type="text"
                name="company_name"
                placeholder="Acme, Inc."
                required
              />
            </Label>

            <Label className={GlobalStyles['mt-3']}>
              Company type:
              <br />
              <Select name="company_type" defaultValue={DefaultCompanyType}>
                {Object.entries(CompanyType).map(([key, val]) => (
                  <option key={val} value={val}>
                    {key}
                  </option>
                ))}
              </Select>
            </Label>
          </DialogBody>
          <DialogFooter>
            <Button theme="primary" type="submit">
              Save
            </Button>
          </DialogFooter>
          {error && (
            <InlineNotification theme="alert" className={GlobalStyles['mt-6']}>
              {error}
            </InlineNotification>
          )}
        </form>
      )}
    </Dialog>
  );
});

const changeCompany =
  (id: string, dialog: HTMLDialogElement | null) => (dispatch: AppDispatch) => {
    redirect(`/?company_id=${id}`);
    dispatch(actions.toggleChangeCompanyDialog(false));
    dialog?.close();
  };

const ChangeCompanyDialog = ({
  companies,
  isOpen,
  dispatch,
  selectedCompanyID,
}: {
  companies: Company[];
  isOpen: boolean;
  dispatch: AppDispatch;
  selectedCompanyID: string | undefined;
}) => {
  return (
    <Dialog id="changeCompanyDialog" title="Switch Company" open={isOpen}>
      {isOpen && (
        <>
          <DialogBody>
            <div className={HeaderStyles.companyChooserContainer}>
              <ol className={HeaderStyles.companyChooser}>
                {companies.map((company: Company) => (
                  <li
                    key={company.id}
                    className={
                      selectedCompanyID === company.id
                        ? HeaderStyles.selectedCompany
                        : ''
                    }
                  >
                    <button
                      onClick={(e: React.MouseEvent<HTMLButtonElement>) => {
                        const dialog = (e.target as HTMLButtonElement).closest(
                          'dialog'
                        );
                        dispatch(changeCompany(company.id, dialog));
                      }}
                    >
                      {company.name}
                    </button>
                  </li>
                ))}
              </ol>
            </div>
          </DialogBody>
          <DialogFooter className={HeaderStyles.createCompanyFooter}>
            <a
              href="/companies/create"
              title="Create company"
              onClick={(e: React.MouseEvent<HTMLAnchorElement>) => {
                e.preventDefault();
                const self = (e.target as HTMLButtonElement).closest('dialog');
                if (self) {
                  self.close();
                }
                const createCompanyDialog = document.getElementById(
                  'createCompanyDialog'
                ) as HTMLDialogElement;
                dispatch(actions.toggleCreateCompanyDialog(true));
                createCompanyDialog?.showModal();
              }}
              className={PageCommon.link}
            >
              Create Company
            </a>
          </DialogFooter>
        </>
      )}
    </Dialog>
  );
};

export function onSelectTenant(
  selectedCompanyID: string | undefined,
  newTenantID: string | undefined
) {
  redirect(`/?company_id=${selectedCompanyID}&tenant_id=${newTenantID}`);
}

const CreateTenantDialog = ({ isOpen }: { isOpen: boolean }) => {
  return (
    <Dialog id="createTenantDialog" title="Create Tenant" open={isOpen}>
      {isOpen && <CreateTenantPage inDialog />}
    </Dialog>
  );
};

const unimpersonateUser = () => {
  API.unimpersonateUser().then(() => {
    window.location.reload();
  });
};

const Header = ({
  serviceInfo,
  myProfile,
  companies,
  selectedCompanyID,
  tenants,
  selectedTenantID,
  tenantFetchError,
  createTenantDialogIsOpen,
  changeCompanyDialogIsOpen,
  location,
  query,
  dispatch,
}: {
  serviceInfo: ServiceInfo | undefined;
  myProfile: MyProfile | undefined;
  companies: Company[] | undefined;
  selectedCompanyID: string | undefined;
  tenants: Tenant[] | undefined;
  selectedTenantID: string | undefined;
  tenantFetchError: string;
  createTenantDialogIsOpen: boolean;
  changeCompanyDialogIsOpen: boolean;
  location: URL;
  query: URLSearchParams;
  dispatch: AppDispatch;
}) => {
  const cleanQuery = makeCleanPageLink(query);

  const tenantSelectorID =
    location.pathname === '/tenants/create' ? 'create' : selectedTenantID;

  let tenantSelectorItems: { label: string; value: string }[] | undefined = [];
  const tenantSelectorActions: React.ReactElement[] | undefined = [];
  if (tenants) {
    tenantSelectorItems = tenants.map((tenant) => ({
      label: tenant.name,
      value: tenant.id,
    }));
  }
  if (companies && serviceInfo) {
    if (serviceInfo.uc_admin) {
      tenantSelectorActions.push(
        <a
          href={TENANTS_CREATE_PATH + cleanQuery}
          title="Go to create tenant page"
        >
          Create Tenant
        </a>
      );
    }

    if (companies.length > 1) {
      tenantSelectorActions.push(
        <button
          title="Change company"
          onClick={() => {
            const changeCompanyDialog = document.getElementById(
              'changeCompanyDialog'
            ) as HTMLDialogElement;
            changeCompanyDialog?.showModal();
            dispatch(actions.toggleChangeCompanyDialog(true));
          }}
        >
          Change Company
        </button>
      );
    }
  }

  return (
    <header id="pageHeader" className={HeaderStyles.root}>
      <div className={HeaderStyles.controls}>
        <Link href="/" title="Return to home">
          <img src="/logo.svg" alt="UserClouds logo" width={30} height={30} />
        </Link>
        {myProfile && (
          <>
            <div className={HeaderStyles.dropdowns}>
              <label
                className={HeaderStyles.dropdownLabel}
                id="tenantSelectLabel"
                htmlFor="tenantSelectDropdown"
              >
                Tenant:
              </label>
              {selectedCompanyID &&
              (tenantSelectorItems?.length || tenantSelectorActions?.length) ? (
                <PseudoSelect
                  id="tenantSelectDropdown"
                  value={tenantSelectorID || 'create'}
                  labeledBy="tenantSelectLabel"
                  options={tenantSelectorItems}
                  actions={tenantSelectorActions}
                  changeHandler={(val: string) => {
                    onSelectTenant(selectedCompanyID as string, val);
                  }}
                />
              ) : tenantFetchError ? (
                'ERROR'
              ) : (
                '...'
              )}
            </div>
            <Breadcrumbs />
          </>
        )}
      </div>
      {myProfile && (
        <div className={HeaderStyles.profile}>
          <Link
            href={'/iam' + cleanQuery}
            title="Manage Team"
            applyStyles={false}
          >
            <IconManageTeam />
            Manage Team
          </Link>
          <ProfileWidget
            displayName={myProfile.userProfile.name()}
            email={myProfile.userProfile.email()}
            userID={myProfile.userProfile.id}
            pictureURL={myProfile.userProfile.pictureURL()}
            impersonatorName={myProfile.impersonatorProfile?.name()}
            impersonatorUserID={myProfile.impersonatorProfile?.id}
            unimpersonateUser={unimpersonateUser}
          />
        </div>
      )}
      <CreateTenantDialog isOpen={createTenantDialogIsOpen} />
      <CreateCompanyDialog />
      {companies?.length && (
        <ChangeCompanyDialog
          companies={companies}
          isOpen={changeCompanyDialogIsOpen}
          dispatch={dispatch}
          selectedCompanyID={selectedCompanyID}
        />
      )}
    </header>
  );
};

export default connect((state: RootState) => ({
  serviceInfo: state.serviceInfo,
  myProfile: state.myProfile,
  companies: state.companies,
  selectedCompanyID: state.selectedCompanyID,
  tenants: state.tenants,
  selectedTenantID: state.selectedTenantID,
  tenantFetchError: state.tenantFetchError,
  createTenantDialogIsOpen: state.createTenantDialogIsOpen,
  changeCompanyDialogIsOpen: state.changeCompanyDialogIsOpen,
  location: state.location,
  query: state.query,
}))(Header);
