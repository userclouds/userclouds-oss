export type UserRoles = {
  id: string;
  name: string;
  organization_role: string;
  policy_role: string;
};

// TODO: we should have a better way to sync constants between TS and Go
export enum Roles {
  NoRole = 'none',
  MemberRole = '_member',
  AdminRole = '_admin',
  UserGroupPolicyFullRole = '_user_group_policy_full',
  UserGroupPolicyReadRole = '_user_group_policy_read',
  // eslint-disable-next-line @typescript-eslint/no-duplicate-enum-values
  UserGroupPolicyNoRole = 'none',
}

export const getOrganizationRoleDisplayName = (role: string) => {
  switch (role) {
    case Roles.AdminRole:
      return 'Admin';
    case Roles.MemberRole:
      return 'Member';
    default:
      return 'No role';
  }
};

export const getBaselinePolicyRoleDisplayName = (role: string) => {
  switch (role) {
    case Roles.UserGroupPolicyFullRole:
      return 'Full access';
    case Roles.UserGroupPolicyReadRole:
      return 'Read all policies';
    default:
      return 'No baseline access';
  }
};

export const effectiveRoles = (
  userRoles: UserRoles,
  modifiedUserRoles: UserRoles[]
) => {
  const modifiedUR = modifiedUserRoles.find((m) => m.id === userRoles.id);
  if (modifiedUR) {
    return { ...userRoles, ...modifiedUR };
  }
  return userRoles;
};
