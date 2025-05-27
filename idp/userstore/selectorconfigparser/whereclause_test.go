package selectorconfigparser

import (
	"testing"

	"userclouds.com/infra/assert"
)

func TestWhereClause(t *testing.T) {
	assert.NoErr(t, ParseWhereClause("{id} = ?"))
	assert.NoErr(t, ParseWhereClause("{id} = ? AND {email} = ?"))
	assert.NoErr(t, ParseWhereClause("{id} = ? AND {email} = ? AND {name} = ?"))
	assert.NoErr(t, ParseWhereClause("{id} = ? AND {email} = ? AND {name} = ? AND {phone} = ?"))
	assert.NoErr(t, ParseWhereClause("{id} = ? AND {email} = ? AND {name} = ? AND {phone} = ? AND {address} = ?"))
	assert.NoErr(t, ParseWhereClause("{id} = ? AND {email} = ? AND {name} = ? AND {phone} = ? AND {address} = ? AND {zip} = ?"))
	assert.NoErr(t, ParseWhereClause("({organization_id} = ? AND {live} = ? AND {organization_id} = ? AND {account_id} = ? AND {organization_id} = ? AND ! {status} = ? AND {conversation_id} = ? AND {status} = ?)"))
}
