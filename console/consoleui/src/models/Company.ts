enum CompanyType {
  // See: internal/companyconfig/models.go CompanyType
  Prospect = 'prospect',
  Customer = 'customer',
  Internal = 'internal',
  Demo = 'demo',
}
const DefaultCompanyType = CompanyType.Internal;

type Company = {
  id: string;
  name: string;
  type: CompanyType;
  is_admin: boolean;
};

function IsCompanyType(value: string): boolean {
  return Object.values(CompanyType).includes(value as CompanyType);
}

export default Company;
export { CompanyType, DefaultCompanyType, IsCompanyType };
