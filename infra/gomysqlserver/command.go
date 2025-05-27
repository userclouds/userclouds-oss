package gomysqlserver

import (
	"bytes"
	"context"
	"fmt"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
	"github.com/siddontang/go/hack"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

// Handler defines a query handler interface
type Handler interface {
	//handle COM_INIT_DB command, you can check whether the dbName is valid, or other.
	UseDB(dbName string) error
	//handle COM_QUERY command, like SELECT, INSERT, UPDATE, etc...
	//If Result has a Resultset (SELECT, SHOW, etc...), we will send this as the response, otherwise, we will send Result
	HandleQuery(query string) (*mysql.Result, error)
	//handle COM_FILED_LIST command
	HandleFieldList(table string, fieldWildcard string) ([]*mysql.Field, error)
	//handle COM_STMT_PREPARE, params is the param number for this statement, columns is the column number
	//context will be used later for statement execute
	HandleStmtPrepare(query string) (params int, columns int, context any, err error)
	//handle COM_STMT_EXECUTE, context is the previous one set in prepare
	//query is the statement prepare query, and args is the params for this statement
	HandleStmtExecute(context any, query string, args []any) (*mysql.Result, error)
	//handle COM_STMT_CLOSE, context is the previous one set in prepare
	//this handler has no response
	HandleStmtClose(context any) error
	//handle any other command that is not currently handled by the library,
	//default implementation for this method will return an ER_UNKNOWN_ERROR
	HandleOtherCommand(cmd byte, data []byte) error
}

// ReplicationHandler defines a replication interface
type ReplicationHandler interface {
	// handle Replication command
	HandleRegisterSlave(data []byte) error
	HandleBinlogDump(pos mysql.Position) (*replication.BinlogStreamer, error)
	HandleBinlogDumpGTID(gtidSet *mysql.MysqlGTIDSet) (*replication.BinlogStreamer, error)
}

// HandleCommand handles a command from the client
func (c *Conn) HandleCommand(ctx context.Context) error {
	if c.Conn == nil {
		return ucerr.Errorf("connection closed")
	}

	data, err := c.ReadPacket()
	if err != nil {
		if err := c.Close(); err != nil {
			uclog.Warningf(ctx, "close connection error %v", err)
		}
		c.Conn = nil
		return ucerr.Wrap(err)
	}

	v := c.dispatch(ctx, data)

	err = c.WriteValue(v)

	if c.Conn != nil {
		c.ResetSequence()
	}

	if err != nil {
		if err := c.Close(); err != nil {
			uclog.Warningf(ctx, "close connection error %v", err)
		}
		c.Conn = nil
	}
	return ucerr.Wrap(err)
}

func (c *Conn) dispatch(ctx context.Context, data []byte) any {
	cmd := data[0]
	data = data[1:]

	switch cmd {
	case mysql.COM_QUIT:
		if err := c.Close(); err != nil {
			uclog.Warningf(ctx, "close connection error %v", err)
		}
		c.Conn = nil
		return noResponse{}
	case mysql.COM_QUERY:
		r, err := c.h.HandleQuery(hack.String(data))
		if err != nil {
			return err
		}
		return r
	case mysql.COM_PING:
		return nil
	case mysql.COM_INIT_DB:
		if err := c.h.UseDB(hack.String(data)); err != nil {
			return err
		}
		return nil
	case mysql.COM_FIELD_LIST:
		index := bytes.IndexByte(data, 0x00)
		table := hack.String(data[0:index])
		wildcard := hack.String(data[index+1:])
		fs, err := c.h.HandleFieldList(table, wildcard)
		if err != nil {
			return err
		}
		return fs
	case mysql.COM_STMT_PREPARE:
		c.stmtID++
		st := new(Stmt)
		st.ID = c.stmtID
		st.Query = hack.String(data)
		var err error
		if st.Params, st.Columns, st.Context, err = c.h.HandleStmtPrepare(st.Query); err != nil {
			return err
		}
		st.ResetParams()
		c.stmts[c.stmtID] = st
		return st
	case mysql.COM_STMT_EXECUTE:
		r, err := c.handleStmtExecute(data)
		if err != nil {
			return err
		}
		return r
	case mysql.COM_STMT_CLOSE:
		if err := c.handleStmtClose(data); err != nil {
			return err
		}
		return noResponse{}
	case mysql.COM_STMT_SEND_LONG_DATA:
		if err := c.handleStmtSendLongData(data); err != nil {
			return err
		}
		return noResponse{}
	case mysql.COM_STMT_RESET:
		r, err := c.handleStmtReset(data)
		if err != nil {
			return err
		}
		return r
	case mysql.COM_SET_OPTION:
		if err := c.h.HandleOtherCommand(cmd, data); err != nil {
			return err
		}

		return eofResponse{}
	case mysql.COM_REGISTER_SLAVE:
		if h, ok := c.h.(ReplicationHandler); ok {
			return h.HandleRegisterSlave(data)
		}
		return c.h.HandleOtherCommand(cmd, data)
	case mysql.COM_BINLOG_DUMP:
		if h, ok := c.h.(ReplicationHandler); ok {
			pos, err := parseBinlogDump(data)
			if err != nil {
				return err
			}
			s, err := h.HandleBinlogDump(pos)
			if err != nil {
				return err
			}
			return s
		}
		return c.h.HandleOtherCommand(cmd, data)
	case mysql.COM_BINLOG_DUMP_GTID:
		if h, ok := c.h.(ReplicationHandler); ok {
			gtidSet, err := parseBinlogDumpGTID(data)
			if err != nil {
				return err
			}
			s, err := h.HandleBinlogDumpGTID(gtidSet)
			if err != nil {
				return err
			}
			return s
		}
		return c.h.HandleOtherCommand(cmd, data)
	default:
		return c.h.HandleOtherCommand(cmd, data)
	}
}

// EmptyHandler is a default handler that does nothing
type EmptyHandler struct {
}

// EmptyReplicationHandler is a default replication handler that does nothing
type EmptyReplicationHandler struct {
	EmptyHandler
}

// UseDB implements Handler
func (h EmptyHandler) UseDB(dbName string) error {
	return nil
}

// HandleQuery implements Handler
func (h EmptyHandler) HandleQuery(query string) (*mysql.Result, error) {
	return nil, ucerr.Errorf("not supported now")
}

// HandleFieldList implements Handler
func (h EmptyHandler) HandleFieldList(table string, fieldWildcard string) ([]*mysql.Field, error) {
	return nil, ucerr.Errorf("not supported now")
}

// HandleStmtPrepare implements Handler
func (h EmptyHandler) HandleStmtPrepare(query string) (int, int, any, error) {
	return 0, 0, nil, ucerr.Errorf("not supported now")
}

// HandleStmtExecute implements Handler
func (h EmptyHandler) HandleStmtExecute(context any, query string, args []any) (*mysql.Result, error) {
	return nil, ucerr.Errorf("not supported now")
}

// HandleStmtClose implements Handler
func (h EmptyHandler) HandleStmtClose(context any) error {
	return nil
}

// HandleRegisterSlave implements ReplicationHandler
func (h EmptyReplicationHandler) HandleRegisterSlave(data []byte) error {
	return ucerr.Errorf("not supported now")
}

// HandleBinlogDump implements ReplicationHandler
func (h EmptyReplicationHandler) HandleBinlogDump(pos mysql.Position) (*replication.BinlogStreamer, error) {
	return nil, ucerr.Errorf("not supported now")
}

// HandleBinlogDumpGTID implements ReplicationHandler
func (h EmptyReplicationHandler) HandleBinlogDumpGTID(gtidSet *mysql.MysqlGTIDSet) (*replication.BinlogStreamer, error) {
	return nil, ucerr.Errorf("not supported now")
}

// HandleOtherCommand implements Handler
func (h EmptyHandler) HandleOtherCommand(cmd byte, data []byte) error {
	return ucerr.Wrap(mysql.NewError(
		mysql.ER_UNKNOWN_ERROR,
		fmt.Sprintf("command %d is not supported now", cmd),
	))
}
