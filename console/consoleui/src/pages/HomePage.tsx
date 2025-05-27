import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  GlobalStyles,
  Heading,
  IconBookOpen,
  IconCopy,
  IconFileCode,
  IconButton,
  IconNews,
  InputReadOnly,
  Label,
  Text,
} from '@userclouds/ui-component-lib';

import { RootState } from '../store';
import { MyProfile } from '../models/UserProfile';
import { SelectedTenant } from '../models/Tenant';
import Link from '../controls/Link';
import { PageTitle } from '../mainlayout/PageWrap';
import Styles from './HomePage.module.css';
import PageCommon from './PageCommon.module.css';

const HomePage = ({
  myProfile,
  companyID,
  selectedTenant,
  fetchingTenants,
  fetchingSelectedTenant,
}: {
  myProfile: MyProfile | undefined;
  companyID: string | undefined;
  selectedTenant: SelectedTenant | undefined;
  fetchingTenants: boolean;
  fetchingSelectedTenant: boolean;
}) => {
  // Maybe different universes should point to different URLs? e.g. staging could point to
  // https://docs-staging.userclouds.com, and dev could point to http://localhost:4567,
  // which is the port that 'middleman' runs the dev server on in our slateapidocs repo.
  // For now, just point to the prod docs.

  return (
    <div id={Styles.homePageContent}>
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle
          title="Tenant Home"
          itemName={
            selectedTenant
              ? selectedTenant.name
              : fetchingTenants || fetchingSelectedTenant
                ? '...'
                : 'No tenants'
          }
          id={Styles.homePageTitle}
        />

        <ButtonGroup className={Styles.homePageButton}>
          {selectedTenant?.is_admin && (
            <Button
              theme="primary"
              href={`/tenants/${selectedTenant.id}?company_id=${companyID}&tenant_id=${selectedTenant.id}`}
            >
              Edit Tenant
            </Button>
          )}
        </ButtonGroup>
      </div>
      <section id={Styles.hero}>
        <h1>
          Welcome to <em>UserClouds</em>
          {myProfile ? ', ' + myProfile.userProfile.name() : ''}!
        </h1>
        <Heading size="3" headingLevel="2" className={GlobalStyles['mt-6']}>
          How are you keeping users' data safe?
          <br />
          Use the side menu to get going.
        </Heading>
      </section>
      {selectedTenant ? (
        <></>
      ) : fetchingTenants || fetchingSelectedTenant ? (
        'Loading ...'
      ) : (
        <Text>
          This company doesn't have any tenants yet. You can{' '}
          <Link href={`/tenants/create?company_id=${companyID}`}>
            create one now.
          </Link>
        </Text>
      )}

      {selectedTenant && (
        <aside>
          <Card
            title="Tenant Details"
            collapsible={false}
            className={Styles.card}
          >
            <div>
              <Label htmlFor="tenant_id">ID</Label>
              <InputReadOnly id="tenant_id">
                {selectedTenant.id}
                &nbsp;
                <IconButton
                  icon={<IconCopy />}
                  onClick={() => {
                    navigator.clipboard.writeText(selectedTenant.id);
                  }}
                  title="Copy tenant ID to clipboard"
                  aria-label="Copy tenant ID to clipboard"
                />
              </InputReadOnly>
              <Label className={GlobalStyles['mt-4']} htmlFor="tenant_url">
                URL
              </Label>
              <InputReadOnly id="tenant_url">
                {selectedTenant.tenant_url}
                &nbsp;
                <IconButton
                  icon={<IconCopy />}
                  onClick={() => {
                    navigator.clipboard.writeText(
                      selectedTenant.tenant_url as string
                    );
                  }}
                  title="Copy tenant URL to clipboard"
                  aria-label="Copy tenant URL to clipboard"
                />
              </InputReadOnly>
            </div>
          </Card>
          <Card title="Codegen SDKs" collapsible={false}>
            <ButtonGroup className={GlobalStyles['mt-2']}>
              <Button
                theme="secondary"
                onClick={() => {
                  window.open(
                    `/api/tenants/${encodeURIComponent(
                      selectedTenant.id || ''
                    )}/userstore/codegensdk.go`,
                    '_blank'
                  );
                }}
              >
                Download Go SDK
              </Button>
              <Button
                theme="secondary"
                onClick={() => {
                  window.open(
                    `/api/tenants/${encodeURIComponent(
                      selectedTenant.id || ''
                    )}/userstore/codegensdk.py`,
                    '_blank'
                  );
                }}
              >
                Download Python SDK
              </Button>
            </ButtonGroup>
          </Card>
          <Card title="Resources" collapsible={false}>
            <ul className={Styles.resourceList}>
              <li>
                <IconBookOpen />
                <a
                  href="https://docs.userclouds.com/docs/"
                  title="Open documentation site in a new tab"
                  target="_blank"
                  rel="noreferrer"
                  className={PageCommon.link}
                >
                  Documentation
                </a>
              </li>
              <li>
                <IconFileCode />
                <a
                  href="https://docs.userclouds.com/reference"
                  title="Open API reference in a new tab"
                  target="_blank"
                  rel="noreferrer"
                  className={PageCommon.link}
                >
                  API reference
                </a>
              </li>
              <li>
                <IconNews />
                <a
                  href="https://www.userclouds.com/blog"
                  title="Visit the UserClouds blog (opens in a new tab)"
                  target="_blank"
                  rel="noreferrer"
                  className={PageCommon.link}
                >
                  Blog
                </a>
              </li>
            </ul>
          </Card>
        </aside>
      )}
    </div>
  );
};

export default connect((state: RootState) => {
  return {
    myProfile: state.myProfile,
    companyID: state.selectedCompanyID,
    selectedTenant: state.selectedTenant,
    fetchingTenants: state.fetchingTenants,
    fetchingSelectedTenant: state.fetchingSelectedTenant,
  };
})(HomePage);
