package gomysqlserver

import (
	"encoding/binary"

	"github.com/go-mysql-org/go-mysql/mysql"

	"userclouds.com/infra/ucerr"
)

func parseBinlogDump(data []byte) (mysql.Position, error) {
	if len(data) < 10 {
		return mysql.Position{}, ucerr.Wrap(mysql.ErrMalformPacket)
	}
	var p mysql.Position
	p.Pos = binary.LittleEndian.Uint32(data[0:4])
	p.Name = string(data[10:])

	return p, nil
}

func parseBinlogDumpGTID(data []byte) (*mysql.MysqlGTIDSet, error) {
	if len(data) < 15 {
		return nil, ucerr.Wrap(mysql.ErrMalformPacket)
	}
	lenPosName := binary.LittleEndian.Uint32(data[11:15])
	if len(data) < 22+int(lenPosName) {
		return nil, ucerr.Wrap(mysql.ErrMalformPacket)
	}

	return mysql.DecodeMysqlGTIDSet(data[22+lenPosName:])
}
