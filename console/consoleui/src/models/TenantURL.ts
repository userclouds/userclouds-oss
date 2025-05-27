type TenantURL = {
  id: string;
  created: string;
  updated: string;
  deleted: string;
  tenant_id: string;
  tenant_url: string;
  validated: boolean;
  system: boolean;
  active: boolean;
  dns_verifier: string;
  certificate_valid_until: string;
};

export default TenantURL;
