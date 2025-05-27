package psqlshim

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgproto3"

	"userclouds.com/idp/internal/sqlshim"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	internalSqlshim "userclouds.com/internal/sqlshim"
)

type connection struct {
	clientConn   net.Conn
	serverConn   net.Conn
	tlsCerts     []tls.Certificate
	client       *pgproto3.Backend
	server       *pgproto3.Frontend
	connectionID uuid.UUID
	observer     internalSqlshim.Observer
	dbName       string
}

type dummyWriter int

func (dummyWriter) Write(p []byte) (n int, err error) {
	return 0, ucerr.New("dummy writer can't be written to")
}

const dummyWriterInstance dummyWriter = 0

type dummyChunkReader int

func (dummyChunkReader) Next(n int) (buf []byte, err error) {
	return nil, ucerr.New("dummy chunk reader can't be read from")
}

const dummyChunkReaderInstance dummyChunkReader = 0

const syncTime = time.Millisecond * 10

// ConnectionFactory is a factory for creating new connections to a PostgreSQL database
type ConnectionFactory struct{}

// NewConnection creates a new connection to a PostgreSQL database
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

	serverConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", serverHost, serverPort))
	if err != nil {
		return nil, ucerr.Friendlyf(err, "[psqlshim connection ID %s] Failed to connect to PostgreSQL database at %s:%d", connectionID, serverHost, serverPort)
	}

	client, err := pgproto3.NewBackend(dummyChunkReaderInstance, clientConn)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	server, err := pgproto3.NewFrontend(pgproto3.NewChunkReader(serverConn), serverConn)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &connection{
		clientConn:   clientConn,
		serverConn:   serverConn,
		tlsCerts:     certs,
		connectionID: connectionID,
		client:       client,
		server:       server,
		observer:     observer,
	}, nil
}

func (c *connection) closeConnections(ctx context.Context) {
	uclog.Verbosef(ctx, "[psqlshim connection ID %s] Closing client connection %s:%s", c.connectionID, c.clientConn.RemoteAddr().Network(), c.clientConn.RemoteAddr().String())
	if err := c.clientConn.Close(); err != nil {
		uclog.Infof(ctx, "[psqlshim connection ID %s] Failed to close client connection: %v", c.connectionID, err)
	}
	uclog.Verbosef(ctx, "[psqlshim connection ID %s] Closing server connection %s:%s", c.connectionID, c.serverConn.RemoteAddr().Network(), c.serverConn.RemoteAddr().String())
	if err := c.serverConn.Close(); err != nil {
		uclog.Infof(ctx, "[psqlshim connection ID %s] Failed to close server connection: %v", c.connectionID, err)
	}
}

func (c *connection) Handle(ctx context.Context) error {

	defer func() {
		uclog.Infof(ctx, "[psqlshim connection ID %s] Handle finished", c.connectionID)
		c.closeConnections(ctx)
	}()

	// handle the startup sequence
	if err := c.handleStartup(ctx); err != nil {
		return ucerr.Wrap(err)
	}

	currentDB, err := c.getCurrentDB()
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "[psqlshim connection ID %s] Current database: %s", c.connectionID, currentDB)
	c.dbName = currentDB

	// extract the schema of the users table
	c.observer.NotifySchemaSelected(ctx, currentDB)

	// copy all output from server to client, except for query messages
	pauseCopy := make(chan bool, 1)
	resumeCopy := make(chan bool, 1)
	quit := make(chan bool, 1)
	copyError := make(chan error, 1)
	go c.copyConn(ctx, false, pauseCopy, resumeCopy, quit, copyError)

	for {
		select {
		case err := <-copyError:
			return ucerr.Wrap(err)
		default:
		}

		start := time.Now().UTC()

		ctx = request.SetRequestID(ctx, uuid.Must(uuid.NewV4()))

		// manually read the message from the client (since pgproto3 has bugs in re-encoding messages)
		var completeMessage []byte
		buf, err := readN(c.clientConn, 5)
		if err != nil {
			return ucerr.Wrap(err)
		}
		completeMessage = append(completeMessage, buf...)
		if msgSize := int(binary.BigEndian.Uint32(buf[1:5]) - 4); msgSize > 0 {
			buf, err := readN(c.clientConn, msgSize)
			if err != nil {
				return ucerr.Wrap(err)
			}
			completeMessage = append(completeMessage, buf...)
		}

		// parse the message from the client by passing the message through a new backend
		f, err := pgproto3.NewBackend(pgproto3.NewChunkReader(bytes.NewReader(completeMessage)), dummyWriterInstance)
		if err != nil {
			return ucerr.Wrap(err)
		}
		clientMsg, err := f.Receive()
		if err != nil {
			return ucerr.Wrap(err)
		}

		var queryString *string
		// handle query messages
		if query, ok := clientMsg.(*pgproto3.Query); ok {
			uclog.Debugf(ctx, "[psqlshim connection ID %s] Query: %v", c.connectionID, query)
			queryString = &query.String
			pauseCopy <- true
			time.Sleep(syncTime)
		} else if parse, ok := clientMsg.(*pgproto3.Parse); ok {
			uclog.Debugf(ctx, "[psqlshim connection ID %s] Parse: %v", c.connectionID, parse)
			queryString = &parse.Query
			pauseCopy <- true
			time.Sleep(syncTime)
		} else if describe, ok := clientMsg.(*pgproto3.Describe); ok {
			uclog.Warningf(ctx, "[psqlshim connection ID %s] Describe: %v", c.connectionID, describe)
		} else if bind, ok := clientMsg.(*pgproto3.Bind); ok {
			uclog.Warningf(ctx, "[psqlshim connection ID %s] Bind: %v", c.connectionID, bind)
		} else if execute, ok := clientMsg.(*pgproto3.Execute); ok {
			uclog.Warningf(ctx, "[psqlshim connection ID %s] Execute: %v", c.connectionID, execute)
		} else if sync, ok := clientMsg.(*pgproto3.Sync); ok {
			uclog.Warningf(ctx, "[psqlshim connection ID %s] Sync: %v", c.connectionID, sync)
		} else if close, ok := clientMsg.(*pgproto3.Close); ok {
			uclog.Warningf(ctx, "[psqlshim connection ID %s] Close: %v", c.connectionID, close)
		} else if copyData, ok := clientMsg.(*pgproto3.CopyData); ok {
			uclog.Warningf(ctx, "[psqlshim connection ID %s] CopyData: %v", c.connectionID, copyData)
		} else if copyFail, ok := clientMsg.(*pgproto3.CopyFail); ok {
			uclog.Warningf(ctx, "[psqlshim connection ID %s] CopyFail: %v", c.connectionID, copyFail)
		} else if _, ok := clientMsg.(*pgproto3.Terminate); ok {
			uclog.Debugf(ctx, "[psqlshim connection ID %s] Terminate", c.connectionID)
		} else {
			uclog.Warningf(ctx, "[psqlshim connection ID %s] Unknown message: %v", c.connectionID, clientMsg)
		}

		if queryString != nil {

			handleResponse, transformers, _, err := c.observer.HandleQuery(ctx, internalSqlshim.DatabaseTypePostgres, *queryString, c.dbName, c.connectionID)
			if err != nil {
				uclog.Errorf(ctx, "[psqlshim connection ID %s] Error by query handler: %v", c.connectionID, err)
			}
			if handleResponse == internalSqlshim.Passthrough {
				if _, err := c.serverConn.Write(completeMessage); err != nil {
					return ucerr.Wrap(err)
				}
			} else if handleResponse == internalSqlshim.TransformResponse {
				if _, err := c.serverConn.Write(completeMessage); err != nil {
					return ucerr.Wrap(err)
				}
				if err := c.readAndTransformQueryResponse(ctx, transformers); err != nil {
					return ucerr.Wrap(err)
				}
				end := time.Now().UTC()
				duration := end.Sub(start)
				uclog.DebugfPII(ctx, "psqlshim query returned in %v", duration)
			} else if handleResponse == internalSqlshim.AccessDenied {
				c.observer.TransformSummary(ctx, transformers, 0, 0, 0)

				// send an error message to the client
				errMsg := &pgproto3.ErrorResponse{
					Severity: "ERROR",
					Code:     "42501",
					Message:  "permission denied",
				}
				if err := c.client.Send(errMsg); err != nil {
					return ucerr.Wrap(err)
				}
				readyForQuery := &pgproto3.ReadyForQuery{}
				if err := c.client.Send(readyForQuery); err != nil {
					return ucerr.Wrap(err)
				}
			}
			resumeCopy <- true
			time.Sleep(syncTime)
		} else {
			// forward the message to the server
			if _, err := c.serverConn.Write(completeMessage); err != nil {
				return ucerr.Wrap(err)
			}
		}

		if _, ok := clientMsg.(*pgproto3.Terminate); ok {
			quit <- true
			return nil
		}
	}
}

func (c *connection) copyConn(ctx context.Context, clientToServer bool, pauseCopy, resumeCopy, quit <-chan bool, failed chan<- error) {
	var srcConn, dstConn net.Conn
	if clientToServer {
		srcConn = c.clientConn
		dstConn = c.serverConn
	} else {
		srcConn = c.serverConn
		dstConn = c.clientConn
	}

	defer func() {
		c.unsetReadDeadline(ctx, srcConn)
	}()

	pauseCopying := false
	for {
		select {
		case <-pauseCopy:
			pauseCopying = true
		case <-resumeCopy:
			pauseCopying = false
		case <-quit:
			return
		default:
			if !pauseCopying {
				c.setReadDeadline(ctx, srcConn)
				n, err := io.Copy(dstConn, srcConn)
				if err != nil {
					if !errors.Is(err, os.ErrDeadlineExceeded) {
						failed <- ucerr.Wrap(err)
						return
					}
				} else if n == 0 {
					// err == nil and n == 0 means EOF
					failed <- ucerr.Friendlyf(nil, "[psqlshim connection ID %s] EOF copying between connections", c.connectionID)
					return
				}
			} else {
				time.Sleep(syncTime)
			}
		}
	}
}

const sslNegotiationStartupMessageType uint32 = 80877103

func (c *connection) handleStartup(ctx context.Context) error {

	skipClientRead := false
	for {
		if !skipClientRead {
			uclog.Infof(ctx, "[psqlshim connection ID %s] Startup reading from client", c.connectionID)

			buf, err := readN(c.clientConn, 4)
			if err != nil {
				return ucerr.Wrap(err)
			}
			msgSize := int(binary.BigEndian.Uint32(buf) - 4)

			msg, err := readN(c.clientConn, msgSize)
			if err != nil {
				return ucerr.Wrap(err)
			}

			if msgSize == 4 && binary.BigEndian.Uint32(msg) == sslNegotiationStartupMessageType {
				// SSLRequest message - we don't support SSL
				if _, err := c.clientConn.Write([]byte("N")); err != nil {
					return ucerr.Wrap(err)
				}
				continue
			}

			uclog.Infof(ctx, "[psqlshim connection ID %s] Startup writing to server", c.connectionID)

			if _, err := c.serverConn.Write(append(buf, msg...)); err != nil {
				return ucerr.Wrap(err)
			}
		}

		uclog.Infof(ctx, "[psqlshim connection ID %s] Startup reading from server", c.connectionID)

		msg, err := c.server.Receive()
		if err != nil {
			return ucerr.Wrap(err)
		}

		uclog.Infof(ctx, "[psqlshim connection ID %s] Startup writing to client", c.connectionID)

		if err := c.client.Send(msg); err != nil {
			return ucerr.Wrap(err)
		}

		if _, ok := msg.(*pgproto3.ReadyForQuery); ok {
			break
		}

		if auth, ok := msg.(*pgproto3.Authentication); ok {
			if auth.Type == pgproto3.AuthTypeSASL {
				if err := c.readUntilSaslFinal(ctx); err != nil {
					return ucerr.Wrap(err)
				}
			}
			skipClientRead = true
		}

		if _, ok := msg.(*pgproto3.ErrorResponse); ok {
			return ucerr.Friendlyf(nil, "startup sequence failed")
		}
	}

	return nil
}

func (c *connection) readAndTransformQueryResponse(ctx context.Context, transformInfo any) error {

	quit := make(chan bool, 1)
	copyError := make(chan error, 1)
	go c.copyConn(ctx, true, make(chan bool), make(chan bool), quit, copyError)

	quitSent := false
	defer func() {
		if !quitSent {
			quit <- true
			time.Sleep(syncTime)
		}
	}()

	c.unsetReadDeadline(ctx, c.serverConn)

	defer c.observer.CleanupTransformerExecution(transformInfo)

	colNames := []string{}
	var numSelectorRows, numReturned, numDenied int

	for {
		select {
		case err := <-copyError:
			return ucerr.Wrap(err)
		default:
		}

		// manually read the message from the server (since pgproto3 has bugs in re-encoding messages)
		var completeMessage []byte
		buf, err := readN(c.serverConn, 5)
		if err != nil {
			return ucerr.Wrap(err)
		}
		completeMessage = append(completeMessage, buf...)

		if msgSize := int(binary.BigEndian.Uint32(buf[1:5]) - 4); msgSize > 0 {
			buf, err = readN(c.serverConn, msgSize)
			if err != nil {
				return ucerr.Wrap(err)
			}
			completeMessage = append(completeMessage, buf...)
		}

		// Parse the response from the server by passing the message through a new frontend
		f, err := pgproto3.NewFrontend(pgproto3.NewChunkReader(bytes.NewReader(completeMessage)), dummyWriterInstance)
		if err != nil {
			return ucerr.Wrap(err)
		}

		msg, err := f.Receive()
		if err != nil {
			return ucerr.Wrap(err)
		}

		if rd, ok := msg.(*pgproto3.RowDescription); ok {
			for _, col := range rd.Fields {
				colNames = append(colNames, col.Name)
			}
			if _, err := c.clientConn.Write(completeMessage); err != nil {
				return ucerr.Wrap(err)
			}
		} else if dr, ok := msg.(*pgproto3.DataRow); ok {
			numSelectorRows++
			passedAP, err := c.observer.TransformDataRow(context.Background(), colNames, dr.Values, transformInfo, numReturned)
			if err != nil {
				return ucerr.Wrap(err)
			}
			if passedAP {
				numReturned++
				if err := c.client.Send(dr); err != nil {
					return ucerr.Wrap(err)
				}
			} else {
				numDenied++
			}
		} else {
			if _, ok := msg.(*pgproto3.ReadyForQuery); ok {
				c.observer.TransformSummary(ctx, transformInfo, numSelectorRows, numReturned, numDenied)

				quitSent = true
				quit <- true
				time.Sleep(syncTime)
			}

			if _, err := c.clientConn.Write(completeMessage); err != nil {
				return ucerr.Wrap(err)
			}

			if quitSent {
				return nil
			}
		}
	}
}

func (c *connection) getCurrentDB() (string, error) {
	if err := c.server.Send(&pgproto3.Query{String: "SELECT current_database()"}); err != nil {
		return "", ucerr.Wrap(err)
	}

	database := ""
	for {
		msg, err := c.server.Receive()
		if err != nil {
			return "", ucerr.Wrap(err)
		}
		if dr, ok := msg.(*pgproto3.DataRow); ok {
			if len(dr.Values) > 0 {
				database = string(dr.Values[0])
			}
		}
		if _, ok := msg.(*pgproto3.ReadyForQuery); ok {
			return database, nil
		}
	}
}

func (c *connection) readUntilSaslFinal(ctx context.Context) error {

	quit := make(chan bool, 1)
	copyError := make(chan error, 1)
	go c.copyConn(ctx, true, make(chan bool), make(chan bool), quit, copyError)

	defer func() {
		quit <- true
		time.Sleep(syncTime)
	}()

	for {
		uclog.Infof(ctx, "[psqlshim connection ID %s] Sasl reading from server", c.connectionID)

		// manually read the message from the server (since pgproto3 has bugs in re-encoding messages)
		var completeMessage []byte
		for {
			select {
			case err := <-copyError:
				return ucerr.Wrap(err)
			default:
				uclog.Infof(ctx, "[psqlshim connection ID %s] Sasl reading from server", c.connectionID)
			}
			c.setReadDeadline(ctx, c.serverConn)
			buf, err := readN(c.serverConn, 5)
			if err != nil && !errors.Is(err, os.ErrDeadlineExceeded) {
				return ucerr.Wrap(err)
			}
			if err == nil {
				completeMessage = append(completeMessage, buf...)
				break
			}
		}
		c.unsetReadDeadline(ctx, c.serverConn)

		if msgSize := int(binary.BigEndian.Uint32(completeMessage[1:5]) - 4); msgSize > 0 {
			buf, err := readN(c.serverConn, msgSize)
			if err != nil {
				return ucerr.Wrap(err)
			}
			completeMessage = append(completeMessage, buf...)
		}

		uclog.Infof(ctx, "[psqlshim connection ID %s] Sasl writing to client", c.connectionID)

		if _, err := c.clientConn.Write(completeMessage); err != nil {
			return ucerr.Wrap(err)
		}

		// Parse the response from the server by passing the message through a new frontend
		f, err := pgproto3.NewFrontend(pgproto3.NewChunkReader(bytes.NewReader(completeMessage)), dummyWriterInstance)
		if err != nil {
			return ucerr.Wrap(err)
		}

		msg, err := f.Receive()
		if err != nil {
			return ucerr.Wrap(err)
		}

		if auth, ok := msg.(*pgproto3.Authentication); ok && auth.Type == pgproto3.AuthTypeSASLFinal {
			return nil
		}
	}
}

func (c *connection) setReadDeadline(ctx context.Context, conn net.Conn) {
	if err := conn.SetReadDeadline(time.Now().UTC().Add(syncTime)); err != nil {
		uclog.Infof(ctx, "[psqlshim connection ID %s] Error setting read deadline: %v", c.connectionID, err)
	}
}

func (c *connection) unsetReadDeadline(ctx context.Context, conn net.Conn) {
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		uclog.Infof(ctx, "[psqlshim connection ID %s] Error un-setting read deadline: %v", c.connectionID, err)
	}
}

func readN(conn net.Conn, n int) ([]byte, error) {
	buf := make([]byte, n)
	filled := 0
	for {
		remainingBuf := buf[filled:]
		read, err := conn.Read(remainingBuf)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if read == 0 {
			// indicates EOF
			return nil, ucerr.Wrap(io.EOF)
		}
		filled += read
		if filled == n {
			return buf, nil
		}
	}
}
