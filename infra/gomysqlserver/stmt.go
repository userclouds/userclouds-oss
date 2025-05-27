package gomysqlserver

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/pingcap/errors"

	"userclouds.com/infra/ucerr"
)

var paramFieldData []byte
var columnFieldData []byte

func init() {
	var p = &mysql.Field{Name: []byte("?")}
	var c = &mysql.Field{}

	paramFieldData = p.Dump()
	columnFieldData = c.Dump()
}

// Stmt represents a prepared statement
type Stmt struct {
	ID    uint32
	Query string

	Params  int
	Columns int

	Args []any

	Context any
}

// Rest is unclear
func (s *Stmt) Rest(params int, columns int, context any) {
	s.Params = params
	s.Columns = columns
	s.Context = context
	s.ResetParams()
}

// ResetParams resets the params
func (s *Stmt) ResetParams() {
	s.Args = make([]any, s.Params)
}

func (c *Conn) writePrepare(s *Stmt) error {
	data := make([]byte, 4, 128)

	//status ok
	data = append(data, 0)
	//stmt id
	data = append(data, mysql.Uint32ToBytes(s.ID)...)
	//number columns
	data = append(data, mysql.Uint16ToBytes(uint16(s.Columns))...)
	//number params
	data = append(data, mysql.Uint16ToBytes(uint16(s.Params))...)
	//filter [00]
	data = append(data, 0)
	//warning count
	data = append(data, 0, 0)

	if err := c.WritePacket(data); err != nil {
		return ucerr.Wrap(err)
	}

	if s.Params > 0 {
		for range s.Params {
			data = data[0:4]
			data = append(data, paramFieldData...)

			if err := c.WritePacket(data); err != nil {
				return ucerr.Wrap(errors.Trace(err))
			}
		}

		if err := c.writeEOF(); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if s.Columns > 0 {
		for range s.Columns {
			data = data[0:4]
			data = append(data, columnFieldData...)

			if err := c.WritePacket(data); err != nil {
				return ucerr.Wrap(errors.Trace(err))
			}
		}

		if err := c.writeEOF(); err != nil {
			return ucerr.Wrap(err)
		}
	}
	return nil
}

func (c *Conn) handleStmtExecute(data []byte) (*mysql.Result, error) {
	if len(data) < 9 {
		return nil, ucerr.Wrap(mysql.ErrMalformPacket)
	}

	pos := 0
	id := binary.LittleEndian.Uint32(data[0:4])
	pos += 4

	s, ok := c.stmts[id]
	if !ok {
		return nil, ucerr.Wrap(mysql.NewDefaultError(mysql.ER_UNKNOWN_STMT_HANDLER,
			strconv.FormatUint(uint64(id), 10), "stmt_execute"))
	}

	flag := data[pos]
	pos++
	//now we only support CURSOR_TYPE_NO_CURSOR flag
	if flag != 0 {
		return nil, ucerr.Wrap(mysql.NewError(mysql.ER_UNKNOWN_ERROR, fmt.Sprintf("unsupported flag %d", flag)))
	}

	//skip iteration-count, always 1
	pos += 4

	var nullBitmaps []byte
	var paramTypes []byte
	var paramValues []byte

	if paramNum := s.Params; paramNum > 0 {
		nullBitmapLen := (s.Params + 7) >> 3
		if len(data) < (pos + nullBitmapLen + 1) {
			return nil, ucerr.Wrap(mysql.ErrMalformPacket)
		}
		nullBitmaps = data[pos : pos+nullBitmapLen]
		pos += nullBitmapLen

		//new param bound flag
		if data[pos] == 1 {
			pos++
			if len(data) < (pos + (paramNum << 1)) {
				return nil, ucerr.Wrap(mysql.ErrMalformPacket)
			}

			paramTypes = data[pos : pos+(paramNum<<1)]
			pos += paramNum << 1

			paramValues = data[pos:]
		}

		if err := c.bindStmtArgs(s, nullBitmaps, paramTypes, paramValues); err != nil {
			return nil, ucerr.Wrap(errors.Trace(err))
		}
	}

	var r *mysql.Result
	var err error
	if r, err = c.h.HandleStmtExecute(s.Context, s.Query, s.Args); err != nil {
		return nil, ucerr.Wrap(errors.Trace(err))
	}

	s.ResetParams()

	return r, nil
}

func (c *Conn) bindStmtArgs(s *Stmt, nullBitmap, paramTypes, paramValues []byte) error {
	args := s.Args

	pos := 0

	var v []byte
	var n int
	var isNull bool
	var err error

	for i := range s.Params {
		if nullBitmap[i>>3]&(1<<(uint(i)%8)) > 0 {
			args[i] = nil
			continue
		}

		tp := paramTypes[i<<1]
		isUnsigned := (paramTypes[(i<<1)+1] & 0x80) > 0

		switch tp {
		case mysql.MYSQL_TYPE_NULL:
			args[i] = nil
			continue

		case mysql.MYSQL_TYPE_TINY:
			if len(paramValues) < (pos + 1) {
				return ucerr.Wrap(mysql.ErrMalformPacket)
			}

			if isUnsigned {
				args[i] = paramValues[pos]
			} else {
				args[i] = int8(paramValues[pos])
			}

			pos++
			continue

		case mysql.MYSQL_TYPE_SHORT, mysql.MYSQL_TYPE_YEAR:
			if len(paramValues) < (pos + 2) {
				return ucerr.Wrap(mysql.ErrMalformPacket)
			}

			if isUnsigned {
				args[i] = binary.LittleEndian.Uint16(paramValues[pos : pos+2])
			} else {
				args[i] = int16(binary.LittleEndian.Uint16(paramValues[pos : pos+2]))
			}
			pos += 2
			continue

		case mysql.MYSQL_TYPE_INT24, mysql.MYSQL_TYPE_LONG:
			if len(paramValues) < (pos + 4) {
				return ucerr.Wrap(mysql.ErrMalformPacket)
			}

			if isUnsigned {
				args[i] = binary.LittleEndian.Uint32(paramValues[pos : pos+4])
			} else {
				args[i] = int32(binary.LittleEndian.Uint32(paramValues[pos : pos+4]))
			}
			pos += 4
			continue

		case mysql.MYSQL_TYPE_LONGLONG:
			if len(paramValues) < (pos + 8) {
				return ucerr.Wrap(mysql.ErrMalformPacket)
			}

			if isUnsigned {
				args[i] = binary.LittleEndian.Uint64(paramValues[pos : pos+8])
			} else {
				args[i] = int64(binary.LittleEndian.Uint64(paramValues[pos : pos+8]))
			}
			pos += 8
			continue

		case mysql.MYSQL_TYPE_FLOAT:
			if len(paramValues) < (pos + 4) {
				return ucerr.Wrap(mysql.ErrMalformPacket)
			}

			args[i] = math.Float32frombits(binary.LittleEndian.Uint32(paramValues[pos : pos+4]))
			pos += 4
			continue

		case mysql.MYSQL_TYPE_DOUBLE:
			if len(paramValues) < (pos + 8) {
				return ucerr.Wrap(mysql.ErrMalformPacket)
			}

			args[i] = math.Float64frombits(binary.LittleEndian.Uint64(paramValues[pos : pos+8]))
			pos += 8
			continue

		case mysql.MYSQL_TYPE_DECIMAL, mysql.MYSQL_TYPE_NEWDECIMAL, mysql.MYSQL_TYPE_VARCHAR,
			mysql.MYSQL_TYPE_BIT, mysql.MYSQL_TYPE_ENUM, mysql.MYSQL_TYPE_SET, mysql.MYSQL_TYPE_TINY_BLOB,
			mysql.MYSQL_TYPE_MEDIUM_BLOB, mysql.MYSQL_TYPE_LONG_BLOB, mysql.MYSQL_TYPE_BLOB,
			mysql.MYSQL_TYPE_VAR_STRING, mysql.MYSQL_TYPE_STRING, mysql.MYSQL_TYPE_GEOMETRY,
			mysql.MYSQL_TYPE_DATE, mysql.MYSQL_TYPE_NEWDATE,
			mysql.MYSQL_TYPE_TIMESTAMP, mysql.MYSQL_TYPE_DATETIME, mysql.MYSQL_TYPE_TIME:
			if len(paramValues) < (pos + 1) {
				return ucerr.Wrap(mysql.ErrMalformPacket)
			}

			v, isNull, n, err = mysql.LengthEncodedString(paramValues[pos:])
			pos += n
			if err != nil {
				return ucerr.Wrap(errors.Trace(err))
			}

			if !isNull {
				args[i] = v
				continue
			} else {
				args[i] = nil
				continue
			}
		default:
			return ucerr.Errorf("Stmt Unknown FieldType %d", tp)
		}
	}
	return nil
}

// stmt send long data command has no response
func (c *Conn) handleStmtSendLongData(data []byte) error {
	if len(data) < 6 {
		return nil
	}

	id := binary.LittleEndian.Uint32(data[0:4])

	s, ok := c.stmts[id]
	if !ok {
		return nil
	}

	paramID := binary.LittleEndian.Uint16(data[4:6])
	if paramID >= uint16(s.Params) {
		return nil
	}

	if s.Args[paramID] == nil {
		s.Args[paramID] = data[6:]
	} else {
		if b, ok := s.Args[paramID].([]byte); ok {
			b = append(b, data[6:]...)
			s.Args[paramID] = b
		} else {
			return nil
		}
	}

	return nil
}

func (c *Conn) handleStmtReset(data []byte) (*mysql.Result, error) {
	if len(data) < 4 {
		return nil, ucerr.Wrap(mysql.ErrMalformPacket)
	}

	id := binary.LittleEndian.Uint32(data[0:4])

	s, ok := c.stmts[id]
	if !ok {
		return nil, ucerr.Wrap(mysql.NewDefaultError(mysql.ER_UNKNOWN_STMT_HANDLER,
			strconv.FormatUint(uint64(id), 10), "stmt_reset"))
	}

	s.ResetParams()

	return &mysql.Result{}, nil
}

// stmt close command has no response
func (c *Conn) handleStmtClose(data []byte) error {
	if len(data) < 4 {
		return nil
	}

	id := binary.LittleEndian.Uint32(data[0:4])

	stmt, ok := c.stmts[id]
	if !ok {
		return nil
	}

	if err := c.h.HandleStmtClose(stmt.Context); err != nil {
		return ucerr.Wrap(err)
	}

	delete(c.stmts, id)

	return nil
}
