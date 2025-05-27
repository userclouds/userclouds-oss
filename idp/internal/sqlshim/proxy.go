package sqlshim

import (
	"context"
	"crypto/tls"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/request"
	"userclouds.com/infra/secret"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	logServerClient "userclouds.com/logserver/client"
	"userclouds.com/plex/manager"
)

// Proxy is a struct that represents a DB (MySQL or Postgres) Connection proxy server
type Proxy struct {
	port int

	tm                   *tenantmap.StateMap
	cacheConfig          *cache.Config
	jwtVerifier          auth.Verifier
	companyConfigStorage *companyconfig.Storage
	workerClient         workerclient.Client
	lgsc                 *logServerClient.Client

	connectionFactory ConnectionFactory
	handlerFactory    HandlerFactory
}

// HealthCheckProxy is a struct that represents a proxy server for NLB Health Checks, specifically when running in EKS
// where we can and configure the health check to a different port than the actual DB proxy
type HealthCheckProxy struct {
	port int
}

// NewProxy creates a new Proxy instance
func NewProxy(port int,
	tm *tenantmap.StateMap,
	cacheConfig *cache.Config,
	jwtVerifier auth.Verifier,
	companyConfigStorage *companyconfig.Storage,
	workerClient workerclient.Client,
	lgsc *logServerClient.Client,
	connectionFactory ConnectionFactory,
	handlerFactory HandlerFactory,
) *Proxy {

	return &Proxy{
		port:                 port,
		tm:                   tm,
		cacheConfig:          cacheConfig,
		jwtVerifier:          jwtVerifier,
		companyConfigStorage: companyConfigStorage,
		workerClient:         workerClient,
		lgsc:                 lgsc,
		connectionFactory:    connectionFactory,
		handlerFactory:       handlerFactory,
	}
}

// Start starts the proxy server
func (p *Proxy) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))
	if err != nil {
		return ucerr.Wrap(err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				uclog.Errorf(ctx, "Failed to accept new connection on port %d: %s", p.port, err.Error())
				continue
			}
			p.handleConnection(conn)
		}
	}()
	return nil
}

func (p *Proxy) handleConnection(conn net.Conn) {
	connectionID := uuid.Must(uuid.NewV4())
	ctxConnection := request.SetRequestIDIfNotSet(context.Background(), connectionID)
	uclog.Verbosef(ctxConnection, "Connection accepted on port %d from %s", p.port, conn.RemoteAddr())
	go func() {
		if err := p.handle(ctxConnection, conn, connectionID); errors.Is(err, sql.ErrNoRows) {
			uclog.Verbosef(ctxConnection, "No matching tenant for port %d", p.port)
		} else if errors.Is(err, io.EOF) {
			// Probably a healthcheck, so no need to log
		} else if err != nil {
			uclog.Warningf(ctxConnection, "Error handling connection on port %d: %v", p.port, err)
		}
	}()

}

func (p *Proxy) handle(ctx context.Context, clientConn net.Conn, connectionID uuid.UUID) error {
	proxyInfo, err := p.companyConfigStorage.GetSQLShimProxyForPort(ctx, p.port)
	if err != nil {
		return ucerr.Wrap(err)
	}
	ctx = uclog.SetTenantID(ctx, proxyInfo.TenantID)
	ts, err := p.tm.GetTenantStateForID(ctx, proxyInfo.TenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// if we don't have a cert/keypair on file, let's create one
	// TODO (sgarrity 7/24): remove the public key check after migration is complete (and remove the default from the DB column)
	if len(proxyInfo.Certificates) == 0 || proxyInfo.PublicKey == "" {
		uclog.Infof(ctx, "Generating new private key and certificate for tenant %v [%s@%s] on port %d", ts.ID, ts.TenantName, ts.CompanyName, p.port)
		cert, pubKeyPEM, privKey, err := generateKeys()
		if err != nil {
			return ucerr.Wrap(err)
		}
		secretName := fmt.Sprintf("private_key_%s_%s", ts.ID, proxyInfo.ID)

		// note we generate some randomness so we don't accidentally overwrite keys
		keySecret, err := secret.NewString(ctx, "proxy", secretName, string(privKey))
		if err != nil {
			return ucerr.Wrap(err)
		}

		proxyInfo.Certificates = []companyconfig.Certificate{{Certificate: string(cert), PrivateKey: *keySecret}}
		proxyInfo.PublicKey = string(pubKeyPEM)

		if err := p.companyConfigStorage.SaveSQLShimProxy(ctx, proxyInfo); err != nil {
			return ucerr.Wrap(err)
		}
	}

	var certs []tls.Certificate
	for _, cert := range proxyInfo.Certificates {
		pk, err := cert.PrivateKey.Resolve(ctx)
		if err != nil {
			return ucerr.Wrap(err)
		}

		tlsCert, err := tls.X509KeyPair([]byte(cert.Certificate), []byte(pk))
		if err != nil {
			return ucerr.Wrap(err)
		}

		certs = append(certs, tlsCert)
	}

	mgr := manager.NewFromDB(ts.TenantDB, p.cacheConfig)
	apps, err := mgr.GetLoginApps(ctx, ts.ID, ts.CompanyID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if len(apps) == 0 {
		return ucerr.Errorf("no login apps found for tenant ID %v", ts.ID)
	}

	cs, err := apps[0].ClientSecret.Resolve(ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	tokenSource, err := jsonclient.ClientCredentialsForURL(ts.GetTenantURL(), apps[0].ClientID, cs, nil)
	if err != nil {
		return ucerr.Wrap(err)
	}
	azc, err := authz.NewClient(ts.GetTenantURL(), authz.JSONClient(tokenSource))
	if err != nil {
		return ucerr.Wrap(err)
	}

	proxyCtx := multitenant.SetTenantState(auth.SetSubjectTypeAndUUID(ctx, apps[0].ID, authz.ObjectTypeLoginApp), ts)
	configStorage := storage.NewFromTenantState(proxyCtx, ts)
	databaseInfo, err := configStorage.GetSQLShimDatabase(proxyCtx, proxyInfo.DatabaseID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	password, err := databaseInfo.Password.Resolve(proxyCtx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	queryHandler := p.handlerFactory.NewProxyHandler(proxyCtx, databaseInfo.ID, ts, azc, p.workerClient, p.lgsc, p.jwtVerifier)
	connection, err := p.connectionFactory.NewConnection(proxyCtx,
		clientConn,
		connectionID,
		databaseInfo.Host,
		databaseInfo.Port,
		databaseInfo.Username,
		password,
		queryHandler,
		certs,
		[]byte(proxyInfo.PublicKey))
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(connection.Handle(proxyCtx))
}

// NewHealthCheckProxy creates a new Proxy instance
func NewHealthCheckProxy(port int) *HealthCheckProxy {
	return &HealthCheckProxy{port: port}
}

// Start starts the proxy server
func (p *HealthCheckProxy) Start(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", p.port))
	if err != nil {
		return ucerr.Wrap(err)
	}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				uclog.Errorf(ctx, "Failed to accept new connection on port %d: %s", p.port, err.Error())
				continue
			}
			p.handleConnection(conn)
		}
	}()
	return nil
}

func (p *HealthCheckProxy) handleConnection(conn net.Conn) {
	connectionID := uuid.Must(uuid.NewV4())
	ctxConnection := request.SetRequestIDIfNotSet(context.Background(), connectionID)
	go p.handle(ctxConnection, conn)
}

func (p *HealthCheckProxy) handle(ctx context.Context, clientConn net.Conn) {
	buf := make([]byte, 1024)
	for {
		n, err := clientConn.Read(buf)
		if errors.Is(err, io.EOF) { // this is the expected behavior for a health check
			break
		} else if err != nil {
			uclog.Warningf(ctx, "Error reading from HealthCheck connection: %v", err)

		} else {
			uclog.Debugf(ctx, "Read %d bytes from HealthCheck connection", n)
		}
	}
	if err := clientConn.Close(); err != nil {
		uclog.Warningf(ctx, "Error closing HealthCheck connection: %v", err)
	}
}
