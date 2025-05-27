import dotenv from 'dotenv';
import connectEnsureLogin from 'connect-ensure-login';
import { Express, Request, Response } from 'express';
import { IdTokenClaims, TokenSet, UserinfoResponse } from 'openid-client';
import passport from 'passport';
import { v4 as uuidv4 } from 'uuid';
import { org1ID, org2ID } from './constants';

declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace Express {
    interface User {
      claims: IdTokenClaims;
      tokenSet: TokenSet;
      userinfo: UserinfoResponse;
    }
  }
}

function randomString(length) {
  let result = '';
  const characters =
    'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  const charactersLength = characters.length;
  let counter = 0;
  while (counter < length) {
    result += characters.charAt(Math.floor(Math.random() * charactersLength));
    counter += 1;
  }
  return result;
}

dotenv.config();

type AuthenticatedRequest = Request & { user: Express.User };

const renderCompany = async (req: AuthenticatedRequest, res: Response) => {
  const orgs = await req.client.listOrganizations();
  res.render('company', {
    userId: req.user.claims.sub,
    orgs: orgs[0],
  });
};

const renderOrg = async (
  req: AuthenticatedRequest,
  res: Response,
  orgID: string
) => {
  const org = await req.client.getOrganization(orgID);
  const datasets = await req.client.listDatasets(org.id);
  const users = await req.client.listUsersInOrg(org.id);
  const admins = await req.client.listAdmins(org.id);
  const isEmployee = await req.client.isEmployee(req.user.claims.sub);
  const adminIDs = admins ? admins.map((admin) => admin.source_object_id) : [];

  res.render('organization', {
    organization: org,
    fires: datasets,
    users,
    adminIDs,
    isEmployee,
  });
};

const Init = (app: Express) => {
  const states: string[] = [];

  app.get('/', (req: Request, res: Response) => {
    res.render('index');
  });

  app.get('/login', passport.authenticate('oidc', { scope: 'openid' }));

  app.get(
    '/callback',
    (req: Request, res: Response, next: (...args: unknown[]) => unknown) => {
      passport.authenticate('oidc', {
        successRedirect: '/home',
        failureRedirect: '/',
        failureMessage: true,
      })(req, res, next);
    }
  );

  app.get(
    '/home',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      try {
        if (await req.client.isEmployee(req.user.claims.sub)) {
          renderCompany(req, res);
          return;
        }

        const orgID = await req.client.getOrgIDForUser(req.user.claims.sub);
        renderOrg(req, res, orgID);
      } catch (e) {
        res.status(500).send(e.message);
      }
    }
  );

  app.get(
    '/createOrganization',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      try {
        await req.client.createOrganization(uuidv4(), req.query.name as string);
        res.redirect('/home');
      } catch (e) {
        res.status(500).send(e.message);
      }
    }
  );

  app.get(
    '/org/:id',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      renderOrg(req, res, req.params.id);
    }
  );

  app.get(
    '/org/:id/fires/:fireid',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      const fire = await req.client.getDataset(req.params.fireid);
      res.render('fire', {
        fire,
        orgID: req.params.id,
      });
    }
  );

  app.get(
    '/org/:id/users/:userid',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      const profileStrs = await req.client.getUserProfileForApp(
        req.params.userid
      );
      const profile =
        profileStrs.data.length > 0 ? JSON.parse(profileStrs.data[0]) : {};
      const isAdmin = await req.client.isAdmin(
        req.params.userid,
        req.params.id
      );
      res.render('user', {
        profile,
        isAdmin,
        orgID: req.params.id,
      });
    }
  );

  app.get(
    '/org/:id/users/:userid/makeAdmin',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      await req.client.makeAdmin(req.params.userid, req.params.id);
      res.redirect(`/org/${req.params.id}/users/${req.params.userid}`);
    }
  );

  app.get(
    '/org/:id/users/:userid/revokeAdmin',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      await req.client.revokeAdmin(req.params.userid, req.params.id);
      res.redirect(`/org/${req.params.id}/users/${req.params.userid}`);
    }
  );

  app.get(
    '/updateUser',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      try {
        const addresses = req.query.userAddresses
          ? JSON.parse(req.query.userAddresses as string)
          : '[]';
        await req.client.updateUserProfileForApp(
          req.query.userID as string,
          req.query.userPhone as string,
          addresses
        );
        res.redirect(`/org/${req.query.orgID}/users/${req.query.userID}`);
      } catch (e) {
        res.status(500).send(e.message);
      }
    }
  );

  app.get(
    '/inviteUser',
    connectEnsureLogin.ensureLoggedIn(),
    async (req: AuthenticatedRequest, res: Response) => {
      const newState = randomString(128);
      states.push(newState);

      let clientID = '';
      if (req.query.orgID === org1ID) {
        clientID = process.env.ORG1_CLIENT_ID;
      } else if (req.query.orgID === org2ID) {
        clientID = process.env.ORG2_CLIENT_ID;
      } else {
        clientID = process.env.CLIENT_ID;
      }

      try {
        await req.client.inviteUser(
          req.query.email as string, // email we're sending the invite to
          req.user.claims.sub as string, // user id of sending user
          req.user.claims.name, // name we're sending from
          req.user.claims.email as string, // email we're sending from
          newState, // state param for OIDC flow
          clientID, // client ID of the loginapp for the organization we're inviting to
          'http://localhost:3000/inviteCallback', // callback URL
          'You have been invited', // email subject
          '2036-01-01T00:00:00.000000Z' // invite expiry
        );
        res.redirect('/home');
      } catch (e) {
        res.status(500).send(e.message);
      }
    }
  );

  app.get(
    '/inviteCallback',
    async (
      req: Request,
      res: Response,
      next: (...args: unknown[]) => unknown
    ) => {
      req.session[
        `oidc:${process.env.TENANT_URL.replace(/:[0-9]*$/, '').replace(
          /^http[s]{0,1}:\/\//,
          ''
        )}`
      ] = {
        state: req.query.state,
        response_type: 'code',
      };
      passport.authenticate('oidc-invite', {
        successRedirect: '/inviteHome',
        failureRedirect: '/',
        failureMessage: true,
      })(req, res, next);
    }
  );

  app.get('/inviteHome', async (req: Request, res: Response) => {
    const state = req.query.state as string;
    if (states.includes(state)) {
      res.redirect('/home');
    } else {
      res.redirect('/');
    }
  });
};

export default Init;
