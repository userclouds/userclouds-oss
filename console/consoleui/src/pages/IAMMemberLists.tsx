import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  ButtonGroup,
  Card,
  CardFooter,
  CardRow,
  EmptyState,
  InlineNotification,
  IconButton,
  IconDeleteBin,
  IconTeam,
  InputReadOnly,
  Select,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextShortener,
} from '@userclouds/ui-component-lib';
import { APIError } from '@userclouds/sharedui';

import API from '../API';
import actions from '../actions';
import { AppDispatch, RootState } from '../store';
import {
  Roles,
  UserRoles,
  getOrganizationRoleDisplayName,
  getBaselinePolicyRoleDisplayName,
  effectiveRoles,
} from '../models/UserRoles';
import { SelectedTenant } from '../models/Tenant';
import PageCommon from './PageCommon.module.css';
import styles from './IAMMemberLists.module.css';

const fetchUserRoles = (tenantID: string) => (dispatch: AppDispatch) => {
  dispatch({
    type: actions.GET_USERROLES_REQUEST,
  });
  API.listTenantRolesForEmployees(tenantID).then(
    (userRolesList: UserRoles[]) => {
      dispatch({
        type: actions.GET_USERROLES_SUCCESS,
        data: userRolesList,
      });
    },
    (error: APIError) => {
      dispatch({
        type: actions.GET_USERROLES_ERROR,
        data: error.message,
      });
    }
  );
};

const updateUserRoles =
  (tenantID: string, userRoles: UserRoles) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      dispatch({
        type: actions.UPDATE_USERROLES_REQUEST,
      });
      return API.updateTenantRolesForEmployee(tenantID, userRoles).then(
        () => {
          dispatch({
            type: actions.UPDATE_USERROLES_SUCCESS,
            data: userRoles,
          });
          resolve();
        },
        (error) => {
          dispatch({
            type: actions.UPDATE_USERROLES_ERROR,
            data: error.message,
          });
          reject(error);
        }
      );
    });
  };

const saveChanges =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const { selectedTenantID, teamUserRoles, modifiedUserRoles } = getState();
    if (!teamUserRoles || !selectedTenantID) {
      return;
    }

    dispatch({
      type: actions.BULK_UPDATE_USERROLES_START,
    });

    const promises = modifiedUserRoles.map((userRoles) =>
      dispatch(updateUserRoles(selectedTenantID, userRoles))
    );
    Promise.all(promises).then(
      () => {
        dispatch({
          type: actions.BULK_UPDATE_USERROLES_END,
          data: true, // success
        });
        dispatch(fetchUserRoles(selectedTenantID));
      },
      () => {
        dispatch({
          type: actions.BULK_UPDATE_USERROLES_END,
          data: false, // complete or partial failure
        });
        dispatch(fetchUserRoles(selectedTenantID));
      }
    );
  };

const TenantMemberList = ({
  selectedTenant,
  editMode,
  isSaving,
  teamUserRoles,
  isFetching,
  fetchError,
  modifiedUserRoles,
  bulkSaveErrors,
  dispatch,
  readOnly,
}: {
  selectedTenant: SelectedTenant | undefined;
  editMode: boolean;
  isSaving: boolean;
  teamUserRoles: UserRoles[] | undefined;
  isFetching: boolean;
  fetchError: string;
  modifiedUserRoles: UserRoles[];
  bulkSaveErrors: string[];
  dispatch: AppDispatch;
  readOnly: boolean;
}) => {
  const isDirty = modifiedUserRoles.length > 0;

  useEffect(() => {
    if (selectedTenant) {
      dispatch(fetchUserRoles(selectedTenant.id));
    }
  }, [selectedTenant, dispatch]);

  return selectedTenant ? (
    <Card
      title="Manage Tenant Roles"
      description={`Manage roles and add/remove teammates from ${selectedTenant?.name}`}
      isDirty={editMode}
    >
      {fetchError && (
        <InlineNotification theme="alert">{fetchError}</InlineNotification>
      )}
      {selectedTenant && teamUserRoles && teamUserRoles.length ? (
        <>
          {bulkSaveErrors.length > 0 && (
            <InlineNotification theme="alert">
              {bulkSaveErrors.length === 1
                ? bulkSaveErrors[0]
                : `Problems editing or deleting ${bulkSaveErrors.length} users`}
            </InlineNotification>
          )}
          <Table spacing="nowrap" className={styles.tenantRoleTable}>
            <TableHead>
              <TableRow>
                <TableRowHead key="name_head">Name</TableRowHead>
                <TableRowHead key="id_head">User ID</TableRowHead>
                <TableRowHead key="role_head">Tenant role</TableRowHead>
                <TableRowHead key="policy_baseline_head">
                  Baseline policy access
                </TableRowHead>
              </TableRow>
            </TableHead>
            <TableBody>
              {teamUserRoles
                .sort((a, b) => (a.id < b.id ? -1 : 1))
                .map((userRoles) => (
                  <TableRow key={userRoles.id}>
                    <TableCell key="name">{userRoles.name}</TableCell>
                    <TableCell key="id" className={PageCommon.uuidtablecell}>
                      <TextShortener text={userRoles.id} length={36} />
                    </TableCell>
                    <TableCell key="role">
                      {editMode ? (
                        <Select
                          value={
                            modifiedUserRoles.find((m) => m.id === userRoles.id)
                              ?.organization_role || userRoles.organization_role
                          }
                          onChange={(e: React.ChangeEvent) => {
                            dispatch({
                              type: actions.MODIFY_USER_ROLE,
                              data: {
                                id: userRoles.id,
                                organization_role: (
                                  e.target as HTMLSelectElement
                                ).value,
                              },
                            });
                          }}
                        >
                          <option value={Roles.NoRole}>
                            {getOrganizationRoleDisplayName(Roles.NoRole)}
                          </option>
                          <option value={Roles.MemberRole}>
                            {getOrganizationRoleDisplayName(Roles.MemberRole)}
                          </option>
                          <option value={Roles.AdminRole}>
                            {getOrganizationRoleDisplayName(Roles.AdminRole)}
                          </option>
                        </Select>
                      ) : (
                        <InputReadOnly>
                          {getOrganizationRoleDisplayName(
                            userRoles.organization_role
                          )}
                        </InputReadOnly>
                      )}
                    </TableCell>
                    <TableCell key="policyBaseline">
                      {editMode ? (
                        <Select
                          value={
                            modifiedUserRoles.find((m) => m.id === userRoles.id)
                              ?.policy_role || userRoles.policy_role
                          }
                          disabled={
                            effectiveRoles(userRoles, modifiedUserRoles)
                              .organization_role !== Roles.MemberRole
                          }
                          onChange={(e: React.ChangeEvent) => {
                            dispatch({
                              type: actions.MODIFY_USER_ROLE,
                              data: {
                                id: userRoles.id,
                                policy_role: (e.target as HTMLSelectElement)
                                  .value,
                              },
                            });
                          }}
                        >
                          <option value={Roles.UserGroupPolicyNoRole}>
                            {getBaselinePolicyRoleDisplayName(
                              Roles.UserGroupPolicyNoRole
                            )}
                          </option>
                          <option value={Roles.UserGroupPolicyReadRole}>
                            {getBaselinePolicyRoleDisplayName(
                              Roles.UserGroupPolicyReadRole
                            )}
                          </option>
                          <option value={Roles.UserGroupPolicyFullRole}>
                            {getBaselinePolicyRoleDisplayName(
                              Roles.UserGroupPolicyFullRole
                            )}
                          </option>
                        </Select>
                      ) : (
                        <InputReadOnly>
                          {getBaselinePolicyRoleDisplayName(
                            userRoles.organization_role === Roles.AdminRole
                              ? Roles.UserGroupPolicyFullRole
                              : userRoles.policy_role
                          )}
                        </InputReadOnly>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
          <CardFooter>
            {!readOnly && (
              <ButtonGroup>
                {!editMode ? (
                  <ButtonGroup>
                    <Button
                      theme="secondary"
                      onClick={() => {
                        dispatch({
                          type: actions.TOGGLE_USERROLES_EDIT_MODE,
                          data: true,
                        });
                      }}
                    >
                      Edit
                    </Button>
                  </ButtonGroup>
                ) : (
                  <ButtonGroup>
                    <Button
                      theme="primary"
                      isLoading={isSaving}
                      disabled={!isDirty}
                      onClick={() => {
                        dispatch(saveChanges());
                      }}
                    >
                      Save changes
                    </Button>
                    <Button
                      theme="secondary"
                      disabled={isSaving}
                      onClick={() => {
                        dispatch({
                          type: actions.TOGGLE_USERROLES_EDIT_MODE,
                          data: false,
                        });
                      }}
                    >
                      Cancel
                    </Button>
                  </ButtonGroup>
                )}
              </ButtonGroup>
            )}
          </CardFooter>
        </>
      ) : (
        <CardRow>
          {isFetching ? (
            <Text>Loading ...</Text>
          ) : (
            <EmptyState
              title="No members yet"
              image={<IconTeam size="large" />}
            />
          )}
        </CardRow>
      )}
    </Card>
  ) : (
    <> </>
  );
};
const ConnectedTenantMemberList = connect((state: RootState) => {
  return {
    selectedTenant: state.selectedTenant,
    editMode: state.userRolesEditMode,
    isSaving: state.savingUserRoles,
    teamUserRoles: state.teamUserRoles,
    isFetching: state.fetchingUserRoles,
    fetchError: state.fetchUserRolesError,
    modifiedUserRoles: state.modifiedUserRoles,
    bulkSaveErrors: state.userRolesBulkSaveErrors,
  };
})(TenantMemberList);

const fetchCompanyUserRoles =
  (companyID: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: actions.GET_COMPANY_USERROLES_REQUEST,
    });
    API.listCompanyRolesForEmployees(companyID).then(
      (userRolesList: UserRoles[]) => {
        dispatch({
          type: actions.GET_COMPANY_USERROLES_SUCCESS,
          data: userRolesList,
        });
      },
      (error: APIError) => {
        dispatch({
          type: actions.GET_COMPANY_USERROLES_ERROR,
          data: error.message,
        });
      }
    );
  };

const updateCompanyUserRoles =
  (companyID: string, userRoles: UserRoles) =>
  (dispatch: AppDispatch): Promise<void> => {
    return new Promise((resolve, reject) => {
      dispatch({
        type: actions.UPDATE_COMPANY_USERROLES_REQUEST,
      });
      return API.updateCompanyRolesForEmployee(companyID, userRoles).then(
        () => {
          dispatch({
            type: actions.UPDATE_COMPANY_USERROLES_SUCCESS,
            data: userRoles,
          });
          resolve();
        },
        (error) => {
          dispatch({
            type: actions.UPDATE_COMPANY_USERROLES_ERROR,
            data: error.message,
          });
          reject(error);
        }
      );
    });
  };

const deleteCompanyUserRoles =
  (companyID: string, userID: string) => (dispatch: AppDispatch) => {
    dispatch({
      type: actions.DELETE_COMPANY_USERROLES_REQUEST,
    });
    return API.removeCompanyRolesForEmployee(companyID, userID).then(
      () => {
        dispatch({
          type: actions.DELETE_COMPANY_USERROLES_SUCCESS,
          data: userID,
        });
      },
      (error) => {
        dispatch({
          type: actions.DELETE_COMPANY_USERROLES_ERROR,
          data: error.message,
        });
        throw error;
      }
    );
  };

const saveCompanyChanges =
  () => async (dispatch: AppDispatch, getState: () => RootState) => {
    const {
      selectedCompanyID,
      selectedTenantID,
      companyUserRoles,
      companyUserRolesDeleteQueue,
      modifiedCompanyUserRoles,
    } = getState();
    if (!companyUserRoles || !selectedCompanyID) {
      return;
    }

    dispatch({
      type: actions.BULK_UPDATE_COMPANY_USERROLES_START,
    });
    let promises: Array<Promise<void>> = [];
    if (companyUserRolesDeleteQueue.length) {
      if (
        window.confirm(
          `Are you sure you want to remove ${
            companyUserRolesDeleteQueue.length
          } team member${companyUserRolesDeleteQueue.length > 1 ? 's' : ''}?`
        )
      ) {
        promises = companyUserRolesDeleteQueue
          .map((id) => dispatch(deleteCompanyUserRoles(selectedCompanyID, id)))
          .concat(
            // we don='t want to do this map outside the conditional or we'll
            // ignore the result of window.confirm
            modifiedCompanyUserRoles.map((userRoles) =>
              dispatch(updateCompanyUserRoles(selectedCompanyID, userRoles))
            )
          );
      }
    } else {
      // so we have to duplicate the map here
      promises = modifiedCompanyUserRoles.map((userRoles) =>
        dispatch(updateCompanyUserRoles(selectedCompanyID, userRoles))
      );
    }
    Promise.all(promises as Array<Promise<void>>).then(
      () => {
        dispatch({
          type: actions.BULK_UPDATE_COMPANY_USERROLES_END,
          data: true, // success
        });
        dispatch(fetchCompanyUserRoles(selectedCompanyID));
        selectedTenantID && dispatch(fetchUserRoles(selectedTenantID));
      },
      () => {
        dispatch({
          type: actions.BULK_UPDATE_COMPANY_USERROLES_END,
          data: false, // complete or partial failure
        });
        dispatch(fetchCompanyUserRoles(selectedCompanyID));
        selectedTenantID && dispatch(fetchUserRoles(selectedTenantID));
      }
    );
  };

const CompanyMemberList = ({
  companyID,
  editMode,
  isSaving,
  teamUserRoles,
  isFetching,
  fetchError,
  deleteQueue,
  modifiedUserRoles,
  bulkSaveErrors,
  dispatch,
  userID,
}: {
  companyID: string | undefined;
  editMode: boolean;
  isSaving: boolean;
  teamUserRoles: UserRoles[] | undefined;
  isFetching: boolean;
  fetchError: string;
  deleteQueue: string[];
  modifiedUserRoles: UserRoles[];
  bulkSaveErrors: string[];
  dispatch: AppDispatch;
  userID: string | undefined;
}) => {
  const isDirty = deleteQueue.length > 0 || modifiedUserRoles.length > 0;

  // TODO: fetching should really happen outside this component
  useEffect(() => {
    if (companyID) {
      dispatch(fetchCompanyUserRoles(companyID));
    }
  }, [companyID, dispatch]);

  return (
    <>
      {fetchError && (
        <InlineNotification theme="alert">{fetchError}</InlineNotification>
      )}
      {teamUserRoles && teamUserRoles.length ? (
        <>
          {bulkSaveErrors.length > 0 && (
            <InlineNotification theme="alert">
              {bulkSaveErrors.length === 1
                ? bulkSaveErrors[0]
                : `Problems editing or deleting ${bulkSaveErrors.length} users`}
            </InlineNotification>
          )}
          <Table className={styles.companyRoleTable}>
            <TableHead>
              <TableRow>
                <TableRowHead key="name_head">Name</TableRowHead>
                <TableRowHead key="id_head">User ID</TableRowHead>
                <TableRowHead key="role_head">Company role</TableRowHead>
                {editMode && <TableRowHead key="delete_head" />}
              </TableRow>
            </TableHead>
            <TableBody>
              {teamUserRoles
                .sort((a, b) => (a.id < b.id ? -1 : 1))
                .map((userRoles) => (
                  <TableRow
                    key={userRoles.id}
                    title={
                      deleteQueue.includes(userRoles.id)
                        ? 'Queued for delete'
                        : ''
                    }
                    className={
                      deleteQueue.includes(userRoles.id)
                        ? PageCommon.queuedfordelete
                        : ''
                    }
                  >
                    <TableCell key="name">{userRoles.name}</TableCell>
                    <TableCell key="id">
                      <TextShortener text={userRoles.id} length={36} />
                    </TableCell>
                    <TableCell key="role">
                      {editMode ? (
                        <Select
                          value={
                            modifiedUserRoles.find((m) => m.id === userRoles.id)
                              ?.organization_role || userRoles.organization_role
                          }
                          disabled={
                            deleteQueue.includes(userRoles.id) ||
                            userRoles.id === userID
                          }
                          onChange={(e: React.ChangeEvent) => {
                            dispatch({
                              type: actions.MODIFY_COMPANY_USER_ROLE,
                              data: {
                                id: userRoles.id,
                                organization_role: (
                                  e.target as HTMLSelectElement
                                ).value,
                              },
                            });
                          }}
                        >
                          <option value={Roles.AdminRole}>
                            {getOrganizationRoleDisplayName(Roles.AdminRole)}
                          </option>
                          <option value={Roles.MemberRole}>
                            {getOrganizationRoleDisplayName(Roles.MemberRole)}
                          </option>
                        </Select>
                      ) : (
                        <InputReadOnly>
                          {getOrganizationRoleDisplayName(
                            userRoles.organization_role
                          )}
                        </InputReadOnly>
                      )}
                    </TableCell>
                    {editMode && (
                      <TableCell>
                        <IconButton
                          icon={<IconDeleteBin />}
                          onClick={() => {
                            dispatch({
                              type: actions.TOGGLE_COMPANY_USERROLES_FOR_DELETE,
                              data: userRoles.id,
                            });
                          }}
                          title="Delete user"
                          disabled={userRoles.id === userID}
                          aria-label="Delete user"
                        />
                      </TableCell>
                    )}
                  </TableRow>
                ))}
            </TableBody>
          </Table>
          <CardFooter>
            <ButtonGroup>
              {!editMode ? (
                <ButtonGroup>
                  <Button
                    theme="secondary"
                    onClick={() => {
                      dispatch({
                        type: actions.TOGGLE_COMPANY_USERROLES_EDIT_MODE,
                        data: true,
                      });
                    }}
                  >
                    Edit
                  </Button>
                </ButtonGroup>
              ) : (
                <ButtonGroup>
                  <Button
                    theme="primary"
                    isLoading={isSaving}
                    disabled={!isDirty}
                    onClick={() => {
                      dispatch(saveCompanyChanges());
                    }}
                  >
                    Save changes
                  </Button>
                  <Button
                    theme="secondary"
                    disabled={isSaving}
                    onClick={() => {
                      dispatch({
                        type: actions.TOGGLE_COMPANY_USERROLES_EDIT_MODE,
                        data: false,
                      });
                    }}
                  >
                    Cancel
                  </Button>
                </ButtonGroup>
              )}
            </ButtonGroup>
          </CardFooter>
        </>
      ) : (
        <CardRow>
          {isFetching ? (
            <Text>Loading ...</Text>
          ) : (
            <EmptyState
              title="No members yet"
              image={<IconTeam size="large" />}
            />
          )}
        </CardRow>
      )}
    </>
  );
};

export { ConnectedTenantMemberList, CompanyMemberList };
