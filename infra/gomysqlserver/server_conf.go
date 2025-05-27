package gomysqlserver

import (
	"crypto/tls"
	"fmt"
	"sync"

	"github.com/go-mysql-org/go-mysql/mysql"

	"userclouds.com/infra/ucerr"
)

var defaultServer *Server

func init() {
	s, err := NewDefaultServer()
	if err != nil {
		panic(err)
	}
	defaultServer = s
}

// Server efines a basic MySQL server with configs.
//
// We do not aim at implementing the whole MySQL connection suite to have the best compatibilities for the clients.
// The MySQL server can be configured to switch auth methods covering 'mysql_old_password', 'mysql_native_password',
// 'mysql_clear_password', 'authentication_windows_client', 'sha256_password', 'caching_sha2_password', etc.
//
// However, since some old auth methods are considered broken with security issues. MySQL major versions like 5.7 and 8.0 default to
// 'mysql_native_password' or 'caching_sha2_password', and most MySQL clients should have already supported at least one of the three auth
// methods 'mysql_native_password', 'caching_sha2_password', and 'sha256_password'. Thus here we will only support these three
// auth methods, and use 'mysql_native_password' as default for maximum compatibility with the clients and leave the other two as
// config options.
//
// The MySQL doc states that 'mysql_old_password' will be used if 'CLIENT_PROTOCOL_41' or 'CLIENT_SECURE_CONNECTION' flag is not set.
// We choose to drop the support for insecure 'mysql_old_password' auth method and require client capability 'CLIENT_PROTOCOL_41' and 'CLIENT_SECURE_CONNECTION'
// are set. Besides, if 'CLIENT_PLUGIN_AUTH' is not set, we fallback to 'mysql_native_password' auth method.
type Server struct {
	serverVersion     string // e.g. "8.0.12"
	protocolVersion   int    // minimal 10
	capability        uint32 // server capability flag
	collationID       uint8
	defaultAuthMethod string // default authentication method, 'mysql_native_password'
	pubKey            []byte
	tlsConfig         *tls.Config
	cacheShaPassword  *sync.Map // 'user@host' -> SHA256(SHA256(PASSWORD))
}

// mysql.CLIENT_FOUND_ROWS is a capability that allows the server to return the number of rows that match a WHERE clause
// instead of rows that are updated. Django 4 incorrectly assumes this is on regardless, so no-op updates (where
// they try an UPDATE, and then an INSERT if db.affected_rows() == 0), fail if we don't support this.

var defaultCapability = mysql.CLIENT_LONG_PASSWORD | mysql.CLIENT_LONG_FLAG | mysql.CLIENT_CONNECT_WITH_DB | mysql.CLIENT_PROTOCOL_41 |
	mysql.CLIENT_TRANSACTIONS | mysql.CLIENT_SECURE_CONNECTION | mysql.CLIENT_PLUGIN_AUTH |
	mysql.CLIENT_PLUGIN_AUTH_LENENC_CLIENT_DATA | mysql.CLIENT_FOUND_ROWS

// NewDefaultServer creates a new mysql server with default settings.
//
// NOTES:
// TLS support will be enabled by default with auto-generated CA and server certificates (however, you can still use
// non-TLS connection). By default, it will verify the client certificate if present. You can enable TLS support on
// the client side without providing a client-side certificate. So only when you need the server to verify client
// identity for maximum security, you need to set a signed certificate for the client.
func NewDefaultServer() (*Server, error) {
	caPem, caKey, err := generateCA()
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	certPem, keyPem, err := generateAndSignRSACerts(caPem, caKey)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	tlsConf, err := NewServerTLSConfig(caPem, certPem, keyPem, tls.VerifyClientCertIfGiven)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	pk, err := getPublicKeyFromCert(certPem)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &Server{
		serverVersion:     "5.7.0",
		protocolVersion:   10,
		capability:        defaultCapability | mysql.CLIENT_SSL,
		collationID:       mysql.DEFAULT_COLLATION_ID,
		defaultAuthMethod: mysql.AUTH_NATIVE_PASSWORD,
		pubKey:            pk,
		tlsConfig:         tlsConf,
		cacheShaPassword:  new(sync.Map),
	}, nil
}

// NewServer creates a new mysql server with customized settings.
//
// NOTES:
// You can control the authentication methods and TLS settings here.
// For auth method, you can specify one of the supported methods 'mysql_native_password', 'caching_sha2_password', and 'sha256_password'.
// The specified auth method will be enforced by the server in the connection phase. That means, client will be asked to switch auth method
// if the supplied auth method is different from the server default.
// And for TLS support, you can specify self-signed or CA-signed certificates and decide whether the client needs to provide
// a signed or unsigned certificate to provide different level of security.
func NewServer(serverVersion string, collationID uint8, defaultAuthMethod string, pubKey []byte, tlsConfig *tls.Config) *Server {
	if !isAuthMethodSupported(defaultAuthMethod) {
		panic(fmt.Sprintf("server authentication method '%s' is not supported", defaultAuthMethod))
	}

	//if !isAuthMethodAllowedByServer(defaultAuthMethod, allowedAuthMethods) {
	//	panic(fmt.Sprintf("default auth method is not one of the allowed auth methods"))
	//}
	var capFlag = defaultCapability
	if tlsConfig != nil {
		capFlag |= mysql.CLIENT_SSL
	}
	return &Server{
		serverVersion:     serverVersion,
		protocolVersion:   10,
		capability:        capFlag,
		collationID:       collationID,
		defaultAuthMethod: defaultAuthMethod,
		pubKey:            pubKey,
		tlsConfig:         tlsConfig,
		cacheShaPassword:  new(sync.Map),
	}
}

func isAuthMethodSupported(authMethod string) bool {
	return authMethod == mysql.AUTH_NATIVE_PASSWORD || authMethod == mysql.AUTH_CACHING_SHA2_PASSWORD || authMethod == mysql.AUTH_SHA256_PASSWORD
}

// InvalidateCache invalidates the cache for the given user@host.
func (s *Server) InvalidateCache(username string, host string) {
	s.cacheShaPassword.Delete(fmt.Sprintf("%s@%s", username, host))
}

// SetDefaultAuthMethod sets the default auth method for the server.
// this is a convenience method since technically you can do this already via NewServer
func (s *Server) SetDefaultAuthMethod(authMethod string) error {
	if !isAuthMethodSupported(authMethod) {
		return ucerr.Errorf("server authentication method '%s' is not supported", authMethod)
	}
	s.defaultAuthMethod = authMethod
	return nil
}
