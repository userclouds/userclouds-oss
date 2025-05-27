import React, { useState } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  FormNote,
  Label,
  GlobalStyles,
  InlineNotification,
  Select,
  TextInput,
} from '@userclouds/ui-component-lib';
import { Roles } from '../models/UserRoles';
import API from '../API';
import { RootState } from '../store';
import Tenant from '../models/Tenant';
import Styles from './InviteWidget.module.css';

const InviteWidget = ({
  companyID,
  tenants,
  onSend,
  isLoading,
}: {
  companyID: string | undefined;
  tenants: Tenant[] | undefined;
  onSend: () => void;
  isLoading: boolean;
}) => {
  // TODO: we should do some validation on set so the client UI can give
  // the user feedback (right now the validation is server-side)
  const [inviteeEmail, setInviteeEmail] = useState<string>('');
  const [statusText, setStatusText] = useState<string>('');
  const [statusError, setStatusError] = useState<boolean>(false);
  const [sending, setSending] = useState<boolean>(false);

  const companyRoleItems = [
    {
      key: Roles.MemberRole,
      label: 'Member',
      payload: Roles.MemberRole,
    },
    {
      key: Roles.AdminRole,
      label: 'Admin',
      payload: Roles.AdminRole,
    },
  ];

  const tenantRoleItems = [
    {
      key: Roles.NoRole,
      label: 'No Role',
      payload: Roles.NoRole,
    },
    {
      key: Roles.MemberRole,
      label: 'Member',
      payload: Roles.MemberRole,
    },
    {
      key: Roles.AdminRole,
      label: 'Admin',
      payload: Roles.AdminRole,
    },
  ];

  const [companyRole, setCompanyRole] = useState<string>(Roles.MemberRole);
  const [tenantRoles, setTenantRoles] = useState<Record<string, string>>({});

  const onClick = async () => {
    const filteredRoles: Record<string, string> = {};
    Object.keys(tenantRoles).forEach((key) => {
      if (tenantRoles[key] !== Roles.NoRole) {
        filteredRoles[key] = tenantRoles[key];
      }
    });
    setSending(true);

    const maybeError = await API.inviteUserToExistingCompany(
      companyID as string,
      inviteeEmail,
      companyRole,
      filteredRoles
    );
    if (!maybeError) {
      // TODO: Some way to track & show pending invites in user page?
      // Since we don't create user accounts til they sign up, we need a way to list pending invites.
      setStatusError(false);
      setStatusText(`Successfully sent invitation(s)`);
      setInviteeEmail('');
      onSend();
    } else {
      setStatusError(true);
      setStatusText(`${maybeError.message}`);
      onSend();
    }
    setSending(false);
  };

  return (
    <>
      <div className={Styles.invitewidget}>
        <Label>
          Email(s)
          <TextInput
            id="inviteeEmail"
            value={inviteeEmail}
            type="email"
            required
            onChange={(e: React.ChangeEvent) => {
              const val = (e.target as HTMLInputElement).value;
              setInviteeEmail(val);
            }}
          />
          <FormNote>
            To invite 2+ teammates, add a comma-separated list of emails.
          </FormNote>
        </Label>

        <Label>
          Company Role
          <br />
          <Select
            full
            defaultValue={Roles.MemberRole}
            onChange={(e: React.ChangeEvent) => {
              const r = (e.target as HTMLSelectElement).value;
              setCompanyRole(r);
            }}
          >
            {companyRoleItems.map((item) => (
              <option key={item.key} value={item.payload}>
                {item.label}
              </option>
            ))}
          </Select>
        </Label>

        {tenants &&
          tenants.map(
            (tenant) =>
              !tenant.is_console_tenant && (
                <Label key={tenant.id}>
                  {tenant.name + ' Tenant Role'}
                  <br />
                  <Select
                    full
                    defaultValue={Roles.NoRole}
                    onChange={(e: React.ChangeEvent) => {
                      const r = (e.target as HTMLSelectElement).value;
                      tenantRoles[tenant.id] = r;
                      setTenantRoles(tenantRoles);
                    }}
                  >
                    {tenantRoleItems.map((item) => (
                      <option key={item.key} value={item.payload}>
                        {item.label}
                      </option>
                    ))}
                  </Select>
                </Label>
              )
          )}
      </div>

      <Button
        theme="primary"
        onClick={onClick}
        disabled={inviteeEmail.length === 0}
        className={GlobalStyles['mt-6']}
        isLoading={isLoading || sending}
      >
        Send Invitation
      </Button>
      {statusText ? (
        <InlineNotification theme={statusError ? 'alert' : 'success'}>
          {statusText}
        </InlineNotification>
      ) : (
        ''
      )}
    </>
  );
};

export default connect((state: RootState) => ({
  companyID: state.selectedCompanyID,
  tenants: state.tenants,
}))(InviteWidget);
