interface Tenant {
  id: string;
  name: string;
  company_id: string;
  tenant_url?: string;
  use_organizations: boolean;
  state: string;
  is_console_tenant?: boolean;
  user_regions?: string[];
}

export interface SelectedTenant extends Tenant {
  is_admin: boolean;
  is_member: boolean;
}

// TODO: line up with BE when BE validation is built
// export const DATABASE_TYPES = ['msql', 'psql'];

const TenantState = {
  CREATING: 'creating',
  ACTIVE: 'active',
  FAILED_TO_PROVISION: 'failed_to_provision',
};

export type TenantStateType = (typeof TenantState)[keyof typeof TenantState];

export default Tenant;
export { TenantState };
