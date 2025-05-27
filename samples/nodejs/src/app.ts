import dotenv from 'dotenv';
import express, { Express } from 'express';
import session from 'express-session';
import http from 'http';
import path from 'path';
import { Issuer, Strategy, TokenSet, UserinfoResponse } from 'openid-client';
import passport from 'passport';
import {
  AuthZClient,
  getClientCredentialsToken,
  PlexClient,
  UserstoreClient,
} from '@userclouds/sdk-typescript';

import Client from './client';
import Init from './routes';
import checkTypes from './setup_authz';
import setupUserstore from './setup_userstore';

const app = express();
const port = process.env.PORT || 3000;

// load config & make ts type checking easier
dotenv.config();

if (
  process.env.TENANT_URL === undefined ||
  process.env.CLIENT_ID === undefined ||
  process.env.CLIENT_SECRET === undefined
) {
  throw new Error('Missing environment variables');
}

const tenantUrl = process.env.TENANT_URL;
const clientId = process.env.CLIENT_ID;
const clientSecret = process.env.CLIENT_SECRET;

app.use(
  express.urlencoded({
    extended: true,
  })
);

app.use(
  session({
    secret: 'secret',
    resave: false,
    saveUninitialized: true,
  })
);

app.set('views', path.join(__dirname, 'views'));
app.set('view engine', 'ejs');
app.use(express.static(path.join(__dirname, 'public')));

declare global {
  // eslint-disable-next-line @typescript-eslint/no-namespace
  namespace Express {
    interface Request {
      client: Client;
    }
  }
}

app.use(passport.initialize());
app.use(passport.session());

passport.serializeUser(
  (
    user: Express.User,
    done: (err: Error | null, user?: Express.User) => void
  ) => done(null, user)
);
passport.deserializeUser(
  (
    user: Express.User,
    done: (err?: Error | null, user?: Express.User) => void
  ) => {
    done(null, user);
  }
);

app.use(async (req, res, next) => {
  if (
    req.user !== undefined &&
    req.user.tokenSet !== undefined &&
    req.user.tokenSet.id_token !== undefined
  ) {
    const token = await getClientCredentialsToken(
      tenantUrl,
      clientId,
      clientSecret
    );

    req.client = new Client(
      new AuthZClient(tenantUrl, req.user.tokenSet.id_token),
      new UserstoreClient(tenantUrl, req.user.tokenSet.id_token),
      new PlexClient(tenantUrl, token)
    );
  }
  next();
});

Issuer.discover(process.env.TENANT_URL).then((oidcIssuer: Issuer) => {
  const client = new oidcIssuer.Client({
    client_id: clientId,
    client_secret: clientSecret,
    redirect_uris: [`http://localhost:${port}/callback`],
    response_types: ['code'],
  });

  passport.use(
    'oidc',
    new Strategy(
      { client, passReqToCallback: true },
      (
        req: http.IncomingMessage,
        tokenSet: TokenSet,
        userinfo: UserinfoResponse,
        done: (err: Error | null, user?: Express.User) => void
      ) =>
        done(null, {
          tokenSet,
          userinfo,
          claims: tokenSet.claims(),
        })
    )
  );

  const inviteClient = new oidcIssuer.Client({
    client_id: clientId,
    client_secret: clientSecret,
    redirect_uris: [`http://localhost:${port}/inviteCallback`],
    response_types: ['code'],
  });

  passport.use(
    'oidc-invite',
    new Strategy(
      { client: inviteClient, passReqToCallback: true },
      (
        req: http.IncomingMessage,
        tokenSet: TokenSet,
        userinfo: UserinfoResponse,
        done: (err: Error | null, user?: Express.User) => void
      ) =>
        done(null, {
          tokenSet,
          userinfo,
          claims: tokenSet.claims(),
        })
    )
  );
});

checkTypes(tenantUrl, clientId, clientSecret);
setupUserstore(tenantUrl, clientId, clientSecret);

Init(app);

app.listen(port, () =>
  // eslint-disable-next-line no-console
  console.log(`Express is listening at http://localhost:${port}`)
);
