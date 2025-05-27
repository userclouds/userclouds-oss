import { Roles } from './UserRoles';

export type UserInvite = {
  id: string;
  created: string;
  expires: string;
  used: boolean;
  role: Roles.MemberRole | Roles.AdminRole;
  invitee_email: string;
  invitee_user_id: string;
};
