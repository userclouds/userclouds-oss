package sqlshim

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auth"
	internalSqlshim "userclouds.com/internal/sqlshim"
	"userclouds.com/internal/tenantmap"
	logServerClient "userclouds.com/logserver/client"
)

// Connection is an interface for handling a connection to a SQL database
type Connection interface {
	Handle(ctx context.Context) error
}

// ConnectionFactory is an interface for creating a new connection to a SQL database
type ConnectionFactory interface {
	NewConnection(ctx context.Context,
		clientConn net.Conn,
		connectionID uuid.UUID,
		serverHost string,
		serverPort int,
		serverUsername string,
		serverPassword string,
		queryHandler internalSqlshim.Observer,
		certs []tls.Certificate,
		pubKey []byte) (Connection, error)
}

// HandlerFactory is an interface for creating a new proxy handler
type HandlerFactory interface {
	NewProxyHandler(ctx context.Context,
		databaseID uuid.UUID,
		ts *tenantmap.TenantState,
		azc *authz.Client,
		wc workerclient.Client,
		lgsc *logServerClient.Client,
		jwtVerifier auth.Verifier) internalSqlshim.Observer
}
