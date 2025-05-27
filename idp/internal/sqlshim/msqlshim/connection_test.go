package msqlshim

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"

	"userclouds.com/infra/gomysqlserver"
	"userclouds.com/infra/ucerr"
)

func TestConnectionTimeout(t *testing.T) {
	ctx := context.Background()
	cf := ConnectionFactory{}
	conn, err := cf.NewConnection(ctx, nil, uuid.Must(uuid.NewV4()), "127.0.0.1", 3333, "root", "password", nil, nil, nil)
	assert.NoError(t, err) // this won't error because it's lazily created

	// this is where the failure will happen
	cp := conn.(gomysqlserver.CredentialProvider)
	_, err = cp.CheckPassword("root", "password", 0)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to authenticate with target DB: dial tcp 127.0.0.1:3333: connect: connection refused", ucerr.UserFriendlyMessage(err))
}
