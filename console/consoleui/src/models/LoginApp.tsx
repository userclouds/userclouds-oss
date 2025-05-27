import { JSONValue } from '@userclouds/sharedui';
import { SAMLIDP } from './SAMLIDP';
import { MessageTypeMessageElements } from './MessageElements';
import { PageParametersForPage } from './PageParameters';

type LoginApp = {
  id: string; // TODO: UUID
  name: string;
  description: string;
  client_id: string;
  client_secret: string;
  organization_id: string;
  token_validity: JSONValue;
  provider_app_ids: string[];
  allowed_redirect_uris: string[];
  allowed_logout_uris: string[];
  grant_types: string[];
  synced_from_provider: string;
  restricted_access: boolean;
  saml_idp?: SAMLIDP;
  message_elements: Record<string, MessageTypeMessageElements>;
  page_parameters: Record<string, PageParametersForPage>;
  impersonate_user_config: {
    check_attribute: string;
    bypass_company_admin_check: boolean;
  };
};

export default LoginApp;
