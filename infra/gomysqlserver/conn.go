package gomysqlserver

import (
	"context"
	"errors"
	"net"
	"sync/atomic"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/packet"
	"github.com/siddontang/go/sync2"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Conn acts like a MySQL server connection, you can use MySQL client to communicate with it.
type Conn struct {
	*packet.Conn

	serverConf     *Server
	capability     uint32
	charset        uint8
	authPluginName string
	attributes     map[string]string
	connectionID   uint32
	status         uint16
	warnings       uint16
	salt           []byte // should be 8 + 12 for auth-plugin-data-part-1 and auth-plugin-data-part-2

	credentialProvider  CredentialProvider
	user                string
	password            string
	cachingSha2FullAuth bool

	h Handler

	stmts  map[uint32]*Stmt
	stmtID uint32

	closed sync2.AtomicBool
}

var baseConnID uint32 = 10000

// NewConn creates connection with default server settings
func NewConn(ctx context.Context, conn net.Conn, user string, password string, h Handler) (*Conn, error) {
	p := NewInMemoryProvider()
	p.AddUser(user, password)

	var packetConn *packet.Conn
	if defaultServer.tlsConfig != nil {
		packetConn = packet.NewTLSConn(conn)
	} else {
		packetConn = packet.NewConn(conn)
	}

	c := &Conn{
		Conn:               packetConn,
		serverConf:         defaultServer,
		credentialProvider: p,
		h:                  h,
		connectionID:       atomic.AddUint32(&baseConnID, 1),
		stmts:              make(map[uint32]*Stmt),
		salt:               mysql.RandomBuf(20),
	}
	c.closed.Set(false)

	if err := c.handshake(); err != nil {
		if err := c.Close(); err != nil {
			uclog.Warningf(ctx, "close connection error %v", err)
		}
		return nil, ucerr.Wrap(err)
	}

	return c, nil
}

// NewCustomizedConn creates connection with customized server settings
func NewCustomizedConn(ctx context.Context, conn net.Conn, serverConf *Server, p CredentialProvider, h Handler) (*Conn, error) {
	var packetConn *packet.Conn
	if serverConf.tlsConfig != nil {
		packetConn = packet.NewTLSConn(conn)
	} else {
		packetConn = packet.NewConn(conn)
	}

	c := &Conn{
		Conn:               packetConn,
		serverConf:         serverConf,
		credentialProvider: p,
		h:                  h,
		connectionID:       atomic.AddUint32(&baseConnID, 1),
		stmts:              make(map[uint32]*Stmt),
		salt:               mysql.RandomBuf(20),
	}
	c.closed.Set(false)

	if err := c.handshake(); err != nil {
		if err := c.Close(); err != nil {
			uclog.Warningf(ctx, "close connection error %v", err)
		}
		return nil, ucerr.Wrap(err)
	}

	return c, nil
}

func (c *Conn) handshake() error {
	if err := c.writeInitialHandshake(); err != nil {
		return ucerr.Wrap(err)
	}

	if err := c.readHandshakeResponse(); err != nil {
		if errors.Is(err, ErrAccessDenied) {
			var usingPasswd uint16 = mysql.ER_YES
			if errors.Is(err, ErrAccessDeniedNoPassword) {
				usingPasswd = mysql.ER_NO
			}
			err = mysql.NewDefaultError(mysql.ER_ACCESS_DENIED_ERROR, c.user, c.RemoteAddr().String(), mysql.MySQLErrName[usingPasswd])
		}

		if werr := c.writeError(err); werr != nil {
			// TODO (sgarrity 6/24): we might want to combine these, but want to test the auth flow more with that
			// err = ucerr.Combine(err, werr)
			_ = werr
		}

		return ucerr.Wrap(err)
	}

	if err := c.writeOK(nil); err != nil {
		return ucerr.Wrap(err)
	}

	c.ResetSequence()

	return nil
}

// Close closes the connection
func (c *Conn) Close() error {
	c.closed.Set(true)
	return ucerr.Wrap(c.Conn.Close())
}

// Closed returns true if the connection is closed
func (c *Conn) Closed() bool {
	return c.closed.Get()
}

// GetUser returns the user name
func (c *Conn) GetUser() string {
	return c.user
}

// Capability returns the capability
func (c *Conn) Capability() uint32 {
	return c.capability
}

// SetCapability sets the capability
func (c *Conn) SetCapability(cap uint32) {
	c.capability |= cap
}

// UnsetCapability unsets the capability
func (c *Conn) UnsetCapability(cap uint32) {
	c.capability &= ^cap
}

// HasCapability returns true if the capability is set
func (c *Conn) HasCapability(cap uint32) bool {
	return c.capability&cap > 0
}

// Charset returns the charset
func (c *Conn) Charset() uint8 {
	return c.charset
}

// Attributes returns the attributes
func (c *Conn) Attributes() map[string]string {
	return c.attributes
}

// ConnectionID returns the connection ID
func (c *Conn) ConnectionID() uint32 {
	return c.connectionID
}

// IsAutoCommit returns true if the connection is in auto-commit mode
func (c *Conn) IsAutoCommit() bool {
	return c.HasStatus(mysql.SERVER_STATUS_AUTOCOMMIT)
}

// IsInTransaction returns true if the connection is in transaction
func (c *Conn) IsInTransaction() bool {
	return c.HasStatus(mysql.SERVER_STATUS_IN_TRANS)
}

// SetInTransaction sets the connection in transaction
func (c *Conn) SetInTransaction() {
	c.SetStatus(mysql.SERVER_STATUS_IN_TRANS)
}

// ClearInTransaction clears the connection in transaction
func (c *Conn) ClearInTransaction() {
	c.UnsetStatus(mysql.SERVER_STATUS_IN_TRANS)
}

// SetStatus sets the status
func (c *Conn) SetStatus(status uint16) {
	c.status |= status
}

// UnsetStatus unsets the status
func (c *Conn) UnsetStatus(status uint16) {
	c.status &= ^status
}

// HasStatus returns true if the status is set
func (c *Conn) HasStatus(status uint16) bool {
	return c.status&status > 0
}

// SetWarnings sets the warnings
func (c *Conn) SetWarnings(warnings uint16) {
	c.warnings = warnings
}
