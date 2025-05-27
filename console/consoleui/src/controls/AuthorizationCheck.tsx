import React, { useEffect } from 'react';
import { connect } from 'react-redux';

import {
  Button,
  Card,
  CardRow,
  Heading,
  Label,
  InlineNotification,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableRowHead,
  Text,
  TextInput,
} from '@userclouds/ui-component-lib';

import { AppDispatch, RootState } from '../store';
import { makeCleanPageLink } from '../AppNavigation';
import {
  CheckAttributePathRow,
  CheckAttributeResponse,
  AuthorizationRequest,
  getCheckAttributeRows,
} from '../models/authz/CheckAttribute';
import {
  CHECK_AUTHORIZATION_REQUEST,
  CHECK_AUTHORIZATION_RESULT,
  CHECK_AUTHORIZATION_ERROR,
  CHANGE_AUTHORIZATION_REQUEST,
} from '../actions/authz';
import { runTenantAuthZAuthorizationCheck } from '../API/authzAPI';
import Link from './Link';
import PageCommon from '../pages/PageCommon.module.css';

const runTest =
  (tenantID: string, authorizationRequest: AuthorizationRequest) =>
  (dispatch: AppDispatch) => {
    dispatch({
      type: CHECK_AUTHORIZATION_REQUEST,
    });
    runTenantAuthZAuthorizationCheck(tenantID, authorizationRequest).then(
      (response: CheckAttributeResponse) => {
        dispatch({
          type: CHECK_AUTHORIZATION_RESULT,
          data: response,
        });
      },
      (error) => {
        dispatch({
          type: CHECK_AUTHORIZATION_ERROR,
          data: error.message,
        });
      }
    );
  };

const AuthorizationPath = ({
  authorizationPath,
  query,
}: {
  authorizationPath: CheckAttributePathRow[];
  query: URLSearchParams;
}) => {
  if (!authorizationPath.length) {
    return <></>;
  }

  const rows = getCheckAttributeRows(authorizationPath);

  return (
    <>
      {authorizationPath.length && (
        <Label>
          <Heading>Complete Path</Heading>
          <Table>
            <TableHead>
              <TableRow>
                <TableRowHead>Source Object</TableRowHead>
                <TableRowHead>Connecting Edge</TableRowHead>
                <TableRowHead>Target Object</TableRowHead>
              </TableRow>
            </TableHead>
            <TableBody>
              {rows.map((row) => (
                <TableRow key={row}>
                  <TableCell>
                    <Link
                      key={row.sourceObject}
                      href={
                        `/objects/${row.sourceObject}` +
                        makeCleanPageLink(query)
                      }
                    >
                      {row.sourceObject}
                    </Link>
                  </TableCell>
                  <TableCell>
                    {' '}
                    <Link
                      key={row.edge}
                      href={`/edges/${row.edge}` + makeCleanPageLink(query)}
                    >
                      {row.edge}
                    </Link>
                  </TableCell>
                  <TableCell>
                    <Link
                      key={row.targetObject}
                      href={
                        `/objects/${row.targetObject}` +
                        makeCleanPageLink(query)
                      }
                    >
                      {row.targetObject}
                    </Link>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </Label>
      )}
    </>
  );
};
const ConnectedAuthorizationPath = connect((state: RootState) => ({
  query: state.query,
}))(AuthorizationPath);

const AuthorizationCheck = ({
  selectedTenantID,
  authorizationRequest,
  authorizationSuccess,
  authorizationFailure,
  authorizationPath,
  sourceObjectID,
  dispatch,
}: {
  selectedTenantID: string | undefined;
  authorizationRequest: AuthorizationRequest | undefined;
  authorizationSuccess: string;
  authorizationFailure: string;
  authorizationPath: CheckAttributePathRow[];
  sourceObjectID?: string;
  dispatch: AppDispatch;
}) => {
  useEffect(() => {
    if (!authorizationRequest) {
      dispatch({
        type: CHANGE_AUTHORIZATION_REQUEST,
        data: {
          attribute: '',
          sourceObjectID: sourceObjectID,
          targetObjectID: '',
        },
      });
    }
  }, [authorizationRequest, sourceObjectID, dispatch]);

  return authorizationRequest ? (
    <CardRow
      title="Test Authorization Scenarios"
      tooltip={
        <>Test whether a given Object has a Permission for another Object.</>
      }
      collapsible
    >
      <div className={PageCommon.carddetailsrow}>
        <Label>
          Source Object ID
          <br />
          <TextInput
            id="source object id"
            name="source_object_id"
            value={authorizationRequest.sourceObjectID}
            onChange={(e: React.ChangeEvent) => {
              authorizationRequest.sourceObjectID = (
                e.target as HTMLInputElement
              ).value;
              dispatch({
                type: CHANGE_AUTHORIZATION_REQUEST,
                data: authorizationRequest,
              });
            }}
          />
        </Label>

        <Label>
          Attribute
          <br />
          <TextInput
            id="attribute"
            name="attribute"
            value={authorizationRequest.attribute}
            size="medium"
            onChange={(e: React.ChangeEvent) => {
              authorizationRequest.attribute = (
                e.target as HTMLInputElement
              ).value;
              dispatch({
                type: CHANGE_AUTHORIZATION_REQUEST,
                data: authorizationRequest,
              });
            }}
          />
        </Label>

        <Label>
          Target Object ID
          <br />
          <TextInput
            id="target object id"
            name="target_object_id"
            value={authorizationRequest.targetObjectID}
            onChange={(e: React.ChangeEvent) => {
              authorizationRequest.targetObjectID = (
                e.target as HTMLInputElement
              ).value;
              dispatch({
                type: CHANGE_AUTHORIZATION_REQUEST,
                data: authorizationRequest,
              });
            }}
          />
        </Label>
      </div>

      {authorizationSuccess && authorizationPath.length && (
        <>
          <InlineNotification theme="success">
            {authorizationSuccess}
          </InlineNotification>
          <ConnectedAuthorizationPath authorizationPath={authorizationPath} />
        </>
      )}
      {authorizationFailure && (
        <InlineNotification theme="alert">
          {authorizationFailure}
        </InlineNotification>
      )}

      <Button
        theme="primary"
        onClick={() => {
          dispatch(runTest(selectedTenantID as string, authorizationRequest));
        }}
        className={PageCommon['mt-6']}
      >
        Run Test
      </Button>
    </CardRow>
  ) : (
    <Card>
      <Text>Fetching Authorization Request...</Text>
    </Card>
  );
};

const AuthZAuthorizationCheck = connect((state: RootState) => {
  return {
    authorizationRequest: state.authorizationRequest,
    authorizationSuccess: state.authorizationSuccess,
    authorizationFailure: state.authorizationFailure,
    authorizationPath: state.authorizationPath,
  };
})(AuthorizationCheck);

export default AuthZAuthorizationCheck;
