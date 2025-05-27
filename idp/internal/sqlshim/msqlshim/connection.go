package msqlshim

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strconv"

	"github.com/go-mysql-org/go-mysql/client"
	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/sqlshim"
	"userclouds.com/infra/gomysqlserver"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	internalSqlshim "userclouds.com/internal/sqlshim"
)

type connection struct {
	clientConn net.Conn

	serverHost     string
	serverPort     int
	serverUsername string
	serverPassword string
	serverContext  context.Context

	serverConn *client.Conn

	tlsCerts []tls.Certificate
	pubKey   []byte

	connectionID uuid.UUID

	observer internalSqlshim.Observer

	startupDB string
	currentDB string
}

// ConnectionFactory is a factory for creating new connections
type ConnectionFactory struct{}

// NewConnection creates a new Connection
func (c ConnectionFactory) NewConnection(ctx context.Context,
	clientConn net.Conn,
	connectionID uuid.UUID,
	serverHost string,
	serverPort int,
	serverUsername string,
	serverPassword string,
	observer internalSqlshim.Observer,
	certs []tls.Certificate,
	pubKey []byte) (sqlshim.Connection, error) {

	return &connection{
		clientConn:     clientConn,
		serverHost:     serverHost,
		serverPort:     serverPort,
		serverUsername: serverUsername,
		serverPassword: serverPassword,
		serverContext:  ctx,
		tlsCerts:       certs,
		pubKey:         pubKey,
		connectionID:   connectionID,
		observer:       observer,
	}, nil
}

// HandleQuery executes an actual query
func (c *connection) HandleQuery(query string) (*mysql.Result, error) {
	ctx := c.serverContext

	if handleResponse, transformers, _, err := c.observer.HandleQuery(ctx, internalSqlshim.DatabaseTypeMySQL, query+";", c.currentDB, c.connectionID); err != nil {
		uclog.Errorf(ctx, "[msqlshim connection ID %s] Error by query handler: %v", c.connectionID, err)
	} else if handleResponse == internalSqlshim.TransformResponse {
		uclog.Infof(ctx, "handled query: %s", query)
		return c.executeQueryAndTransformResults(ctx, query, transformers)
	} else if handleResponse == internalSqlshim.AccessDenied {
		uclog.Warningf(ctx, "query denied: %s", query)
		c.observer.TransformSummary(ctx, transformers, 0, 0, 0)
		return nil, ucerr.Errorf("ACCESS DENIED")
	} else {
		uclog.Infof(ctx, "passthrough query: %s", query)
	}

	return c.serverConn.Execute(query)
}

// UseDB changes the active DB for this connection
func (c *connection) UseDB(dbName string) error {
	uclog.Infof(c.serverContext, "use db: %s", dbName)
	if c.serverConn != nil {
		if err := c.serverConn.UseDB(dbName); err != nil {
			return ucerr.Wrap(err)
		}
		if c.currentDB != dbName {
			c.currentDB = dbName
			c.observer.NotifySchemaSelected(c.serverContext, dbName)
		}
		return nil
	}

	// if the connection isn't active yet, store it for later
	c.startupDB = dbName
	return nil
}

// HandleFieldList lists fields in a table
func (c *connection) HandleFieldList(table, fieldWildcard string) ([]*mysql.Field, error) {
	return c.serverConn.FieldList(table, fieldWildcard)
}

// HandleStmtPrepare prepares a statement
func (c *connection) HandleStmtPrepare(query string) (params int, columns int, context any, err error) {
	s, err := c.serverConn.Prepare(query)
	if err != nil {
		return 0, 0, nil, ucerr.Wrap(err)
	}

	return s.ParamNum(), s.ColumnNum(), s, nil
}

// HandleStmtClose executes a prepared statement
func (c *connection) HandleStmtExecute(context any, query string, args []any) (*mysql.Result, error) {
	s, ok := context.(*client.Stmt)
	if !ok {
		return nil, ucerr.Errorf("Statement context not *client.Stmt")
	}
	return s.Execute(args)
}

// HandleStmtClose closes a prepared statement
func (c *connection) HandleStmtClose(context any) error {
	s, ok := context.(*client.Stmt)
	if !ok {
		return ucerr.Errorf("Statement context not *client.Stmt got %T", context)
	}

	return ucerr.Wrap(s.Close())
}

// HandleOtherCommand handles other (unknown) commands
func (c *connection) HandleOtherCommand(cmd byte, data []byte) error {
	return ucerr.Errorf("unknown other command %v", string(cmd))
}

// CheckPassword implements gomysqlserver.CredentialProvider
func (c *connection) CheckPassword(username, password string, capability uint32) (bool, error) {
	// pass auth through to the target DB to check
	// note this only happens once the client has spoken well-formed MySQL
	// so we shouldn't be DDoSed by port scans, just well-crafted u/p attacks
	opts := []client.Option{}

	// turn this on if the client requests it
	if capability&mysql.CLIENT_FOUND_ROWS > 0 {
		opts = append(opts, SetCapability(mysql.CLIENT_FOUND_ROWS))
	}

	conn, err := client.Connect(fmt.Sprintf("%s:%d", c.serverHost, c.serverPort),
		username, password, c.startupDB, opts...)
	if err != nil {
		var me *mysql.MyError
		if errors.As(err, &me) && me.Code == mysql.ER_ACCESS_DENIED_ERROR {
			return false, nil
		}

		// this error comes from client.Connect (not other internal UC code), so it should be safe
		// to pass through to the client to make debugging easier
		return false, ucerr.Friendlyf(err, "failed to authenticate with target DB: %v", err.Error()) // lint-safe-wrap
	}
	c.serverConn = conn
	c.currentDB = c.startupDB
	return true, nil
}

// GetCredential implements gomysqlserver.CredentialProvider
func (c *connection) GetCredential(username string) (string, bool, error) {
	if username == c.serverUsername {
		return c.serverPassword, true, nil
	}
	return "", false, nil
}

// SetCapability sets a capability on the client connection
func SetCapability(capability uint32) client.Option {
	return func(c *client.Conn) error {
		c.SetCapability(capability)
		return nil
	}
}

func (c *connection) Handle(ctx context.Context) error {
	defer func() {
		// TODO c.closeConnections(ctx)
	}()

	tlsC := &tls.Config{
		ClientAuth:   tls.NoClientCert, // TODO (sgarrity 7/24): support this some day?
		Certificates: c.tlsCerts,
	}

	// NB: for multi-user support, we require AUTH_CACHING_SHA2_PASSWORD or AUTH_SHA256_PASSWORD
	// to be set as the default auth method. This is because AUTH_NATIVE_PASSWORD (which is old)
	// sends a hash of the password instead of the password in the clear, and we need the plaintext
	// password to pass along to the target DB right now.
	s := gomysqlserver.NewServer("5.7.0", mysql.DEFAULT_COLLATION_ID, mysql.AUTH_CACHING_SHA2_PASSWORD, c.pubKey, tlsC)

	// actually set up the connection from client to us
	sc, err := gomysqlserver.NewCustomizedConn(ctx, c.clientConn, s, c, c)
	if err != nil {
		return ucerr.Wrap(err)
	}

	// if we didn't use pass-through auth via connection.CheckPassword, meaning that the client logged into
	// UserClouds using the default username/password they set up in SQLShimConfig, then we need to set up the
	// server connection.
	if c.serverConn == nil {
		opts := []client.Option{}

		// turn this on if the client requests it
		if sc.HasCapability(mysql.CLIENT_FOUND_ROWS) {
			opts = append(opts, SetCapability(mysql.CLIENT_FOUND_ROWS))
		}

		conn, err := client.Connect(fmt.Sprintf("%s:%d", c.serverHost, c.serverPort),
			c.serverUsername, c.serverPassword, c.startupDB, opts...)
		if err != nil {
			return ucerr.Wrap(err)
		}
		c.serverConn = conn
		c.currentDB = c.startupDB
	}

	// notify observer of the schema selection
	c.observer.NotifySchemaSelected(ctx, c.currentDB)

	// now handle commands
	for {
		if err := sc.HandleCommand(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}
}

func (c *connection) executeQueryAndTransformResults(ctx context.Context, query string, transformInfo any) (*mysql.Result, error) {
	res, err := c.serverConn.Execute(query)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	colNames := []string{}
	for _, col := range res.Fields {
		colNames = append(colNames, string(col.Name))
	}

	transformedRows := []mysql.RowData{}
	var numSelectorRows, numReturned, numDenied int

	defer c.observer.CleanupTransformerExecution(transformInfo)

	for _, row := range res.Values {
		numSelectorRows++
		values := [][]byte{}
		isNull := []bool{}
		for _, fv := range row {
			switch fv.Type {
			case mysql.FieldValueTypeNull:
				values = append(values, nil)
			case mysql.FieldValueTypeUnsigned:
				values = append(values, []byte(strconv.FormatUint(fv.AsUint64(), 10)))
			case mysql.FieldValueTypeSigned:
				values = append(values, []byte(strconv.FormatInt(fv.AsInt64(), 10)))
			case mysql.FieldValueTypeFloat:
				values = append(values, []byte(strconv.FormatFloat(fv.AsFloat64(), 'f', -1, 64)))
			case mysql.FieldValueTypeString:
				values = append(values, []byte(fv.AsString()))
			default:
				return nil, ucerr.New("unknown field value type")
			}
			if fv.Type == mysql.FieldValueTypeNull {
				isNull = append(isNull, true)
			} else {
				isNull = append(isNull, false)
			}
		}
		passedAP, err := c.observer.TransformDataRow(ctx, colNames, values, transformInfo, numReturned)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if !passedAP {
			numDenied++
			continue
		}
		numReturned++

		data := []byte{}
		for i, v := range values {
			if isNull[i] {
				data = append(data, 0xfb)
			} else {
				data = append(data, mysql.PutLengthEncodedString(v)...)
			}
		}
		transformedRows = append(transformedRows, data)
	}
	res.RowDatas = transformedRows

	c.observer.TransformSummary(ctx, transformInfo, numSelectorRows, numReturned, numDenied)

	return res, nil
}
