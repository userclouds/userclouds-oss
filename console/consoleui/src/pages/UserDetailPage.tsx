import React, { useEffect, useState } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardRow,
  Label,
  InputReadOnly,
  InlineNotification,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextInput,
  ToolTip,
  TextShortener,
} from '@userclouds/ui-component-lib';

import { JSONValue } from '@userclouds/sharedui';
import { AppDispatch, RootState } from '../store';
import API from '../API';
import ServiceInfo from '../ServiceInfo';
import {
  fetchUser,
  fetchUserConsentedPurposes,
  fetchUserEvents,
  saveUser,
} from '../thunks/users';
import { toggleUserEditMode, modifyTenantUserProfile } from '../actions/users';
import {
  UserProfile,
  UserProfileSerialized,
  UserEvent,
  UserAuthnSerialized,
  MFAChannelSerialized,
  prettyProfileColumnName,
} from '../models/UserProfile';
import { PageTitle } from '../mainlayout/PageWrap';
import { SelectedTenant } from '../models/Tenant';
import Styles from './UserDetailPage.module.css';
import PageCommon from './PageCommon.module.css';

const StandardProfileInfo = ({
  user,
  editedProfile,
  columnPurposes,
  editMode,
  saveError,
  dispatch,
}: {
  user: UserProfile;
  editedProfile: Record<string, JSONValue>;
  columnPurposes: Map<string, Array<object>>;
  editMode: boolean;
  saveError: string;
  dispatch: AppDispatch;
}) => {
  return (
    <CardRow
      title="User Data"
      tooltip="View this user's data and consented purposes. Access is limited to tenant administrators. Access is logged."
      collapsible
    >
      {saveError && (
        <InlineNotification theme="alert">{saveError}</InlineNotification>
      )}
      <Table className={Styles.userdetailstable} id="userdata">
        <TableHead>
          <TableRow>
            <TableRowHead>Column name</TableRowHead>
            <TableRowHead>Column value</TableRowHead>
            <TableRowHead>Consented purposes</TableRowHead>
          </TableRow>
        </TableHead>
        <TableBody>
          {Object.keys(user.profile)
            .sort()
            .map((key) => {
              const colName = prettyProfileColumnName(key);
              const colValue = editMode
                ? editedProfile[key]
                : user.profile[key];
              const colString =
                typeof colValue === 'object'
                  ? JSON.stringify(colValue)
                  : colValue.toString();
              return (
                <TableRow key={key}>
                  <TableCell>{colName}</TableCell>
                  <TableCell>
                    {editMode ? (
                      <TextInput
                        id={key}
                        key={key}
                        name={key}
                        value={colString}
                        onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                          const data: Record<string, JSONValue> = {};
                          const columnName = e.target.name;
                          data[columnName] = e.target.value;
                          dispatch(modifyTenantUserProfile(data));
                        }}
                      />
                    ) : (
                      colString
                    )}
                  </TableCell>
                  <TableCell>
                    {columnPurposes.get(key)
                      ? columnPurposes.get(key)!.join(', ')
                      : ''}
                  </TableCell>
                </TableRow>
              );
            })}
        </TableBody>
      </Table>
    </CardRow>
  );
};

const ConnectedStandardProfileInfo = connect((state: RootState) => ({
  editMode: state.userEditMode,
  editedProfile: state.currentTenantUserProfileEdited,
  saveError: state.saveTenantUserError,
}))(StandardProfileInfo);

const AuthnElements = ({ authns }: { authns: UserAuthnSerialized[] }) => {
  return (
    <CardRow title="Authentication Methods" collapsible>
      {authns.length
        ? authns.map((authn) => (
            <Label
              key={
                authn.authn_type === 'social'
                  ? authn.oidc_subject
                  : authn.username
              }
            >
              {authn.authn_type === 'social'
                ? `OIDC (${authn.oidc_provider
                    .charAt(0)
                    .toUpperCase()}${authn.oidc_provider.substring(
                    1
                  )}) : Issuer URL (${authn.oidc_issuer_url})`
                : 'Username & Password'}
              <br />
              <InputReadOnly>
                {authn.authn_type === 'social'
                  ? `OIDC Subject: ${authn.oidc_subject}`
                  : authn.username}
              </InputReadOnly>
            </Label>
          ))
        : 'None'}
    </CardRow>
  );
};

const MFAMethods = ({
  mfaChannels,
}: {
  mfaChannels: MFAChannelSerialized[];
}) => {
  return (
    <CardRow title="MFA Methods" collapsible>
      {mfaChannels.length
        ? mfaChannels.map((mfaChannel) => (
            <Label
              key={`${mfaChannel.mfa_channel_type}${mfaChannel.mfa_channel_description}`}
            >
              {mfaChannel.primary
                ? `${mfaChannel.mfa_channel_type} (Primary)`
                : mfaChannel.mfa_channel_type}
              <br />
              <InputReadOnly>
                {mfaChannel.mfa_channel_description}
              </InputReadOnly>
            </Label>
          ))
        : 'None'}
    </CardRow>
  );
};

const UserEvents = ({ events }: { events: UserEvent[] | undefined }) => {
  return (
    <CardRow title="User Events" collapsible>
      {events?.length ? (
        <Table>
          <TableHead>
            <TableRow>
              <TableRowHead key="event_id">ID</TableRowHead>
              <TableRowHead key="event_time">Time</TableRowHead>
              <TableRowHead key="event_name">Name</TableRowHead>
              <TableRowHead key="event_payload">Payload</TableRowHead>
            </TableRow>
          </TableHead>
          <TableBody>
            {events.map((event: UserEvent) => {
              return (
                <TableRow key={event.id}>
                  <TableCell>
                    <TextShortener text={event.id} length={6} />
                  </TableCell>
                  <TableCell>
                    {new Date(event.created).toLocaleString('en-us')}
                  </TableCell>
                  <TableCell>{event.type}</TableCell>
                  <TableCell>
                    <pre>{JSON.stringify(event.payload, null, 2)}</pre>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      ) : (
        'None'
      )}
    </CardRow>
  );
};

const UserAdmin = ({
  selectedTenantID,
  userID,
}: {
  selectedTenantID: string;
  userID: string;
}) => {
  return (
    <CardRow title="Admin" collapsible>
      <Button
        onClick={() => {
          API.impersonateUser(selectedTenantID, userID);
        }}
      >
        Log in as this user
      </Button>
    </CardRow>
  );
};

const UserDetail = ({
  selectedTenant,
  serviceInfo,
  user,
  editedProfile,
  editMode,
  userError,
  consentedPurposes,
  consentedPurposesError,
  events,
  eventsError,
  routeParams,
  dispatch,
}: {
  selectedTenant: SelectedTenant | undefined;
  serviceInfo: ServiceInfo | undefined;
  user: UserProfileSerialized | undefined;
  editedProfile: Record<string, JSONValue>;
  editMode: boolean;
  userError: string;
  consentedPurposes: Array<object> | undefined;
  consentedPurposesError: string;
  events: UserEvent[] | undefined;
  eventsError: string;
  routeParams: Record<string, string>;
  dispatch: AppDispatch;
}) => {
  const [showUserData, setShowUserData] = useState(false);
  const { userID } = routeParams;

  useEffect(() => {
    if (showUserData && selectedTenant && userID) {
      dispatch(fetchUser(selectedTenant.id, userID!));
      dispatch(fetchUserConsentedPurposes(selectedTenant.id, userID!));
      dispatch(fetchUserEvents(selectedTenant.id, userID!));
    }
  }, [showUserData, selectedTenant, userID, dispatch]);

  if (
    !showUserData ||
    eventsError ||
    userError ||
    consentedPurposesError ||
    !user
  ) {
    return (
      <>
        <div className={PageCommon.listviewtablecontrols}>
          <PageTitle title="User Data" itemName="User" />
          <div className={PageCommon.listviewtablecontrolsToolTip}>
            <ToolTip>
              <>
                View this user's data and consented purposes. Access is limited
                to tenant administrators. Access is logged.
              </>
            </ToolTip>
          </div>
          <ButtonGroup>
            <Button
              theme="dangerous"
              onClick={() => {
                setShowUserData(true);
              }}
            >
              Reveal User Data
            </Button>
          </ButtonGroup>
        </div>
        <Card
          detailview
          lockedMessage={!showUserData ? 'User Data Hidden' : ''}
        >
          {showUserData &&
            (((eventsError || userError || consentedPurposesError) && (
              <InlineNotification theme="alert">
                {eventsError || userError || consentedPurposesError}
              </InlineNotification>
            )) ||
              (!user && <Text>Fetching user...</Text>))}
        </Card>
      </>
    );
  }

  const userProfile = UserProfile.fromJSON(user);

  const columnPurposeMap: Map<string, Array<object>> = new Map();
  if (consentedPurposes) {
    consentedPurposes.forEach((columnPurposes: any) => {
      columnPurposeMap.set(
        columnPurposes.column.name,
        columnPurposes.consented_purposes.map((purpose: any) => purpose.name)
      );
    });
  }

  return (
    <>
      <div className={PageCommon.listviewtablecontrols}>
        <PageTitle title="User Data" itemName="User" />
        <div className={PageCommon.listviewtablecontrolsToolTip}>
          <ToolTip>
            <>
              View this user's data and consented purposes. Access is limited to
              tenant administrators. Access is logged.
            </>
          </ToolTip>
        </div>
        <ButtonGroup>
          {editMode ? (
            <>
              <Button
                theme="secondary"
                onClick={() => {
                  dispatch(toggleUserEditMode());
                }}
              >
                Cancel
              </Button>
              <Button
                theme="primary"
                onClick={() => {
                  const savedUser = UserProfile.fromJSON(user);
                  savedUser.profile = editedProfile;
                  if (savedUser && selectedTenant) {
                    dispatch(saveUser(selectedTenant?.id, savedUser));
                  }
                }}
              >
                Save User
              </Button>
            </>
          ) : (
            <Button
              theme="primary"
              onClick={() => {
                dispatch(toggleUserEditMode());
              }}
            >
              Edit User Data
            </Button>
          )}
        </ButtonGroup>
      </div>
      <Card detailview>
        <ConnectedStandardProfileInfo
          user={userProfile}
          columnPurposes={columnPurposeMap}
        />

        <AuthnElements authns={userProfile.authns} />

        <MFAMethods mfaChannels={userProfile.mfaChannels} />

        <UserEvents events={events} />

        {serviceInfo?.uc_admin && selectedTenant && (
          <UserAdmin
            selectedTenantID={selectedTenant.id}
            userID={userProfile.id}
          />
        )}
      </Card>
    </>
  );
};

const ConnectedUserDetail = connect((state: RootState) => ({
  selectedTenant: state.selectedTenant,
  serviceInfo: state.serviceInfo,
  user: state.currentTenantUser,
  userError: state.currentTenantUserError,
  consentedPurposes: state.currentTenantUserConsentedPurposes,
  consentedPurposesError: state.currentTenantUserConsentedPurposesError,
  events: state.currentTenantUserEvents,
  eventsError: state.currentTenantUserEventsError,
  editMode: state.userEditMode,
  editedProfile: state.currentTenantUserProfileEdited,
  routeParams: state.routeParams,
}))(UserDetail);

export default ConnectedUserDetail;
