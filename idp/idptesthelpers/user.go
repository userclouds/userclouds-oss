package idptesthelpers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/cache/testhelpers"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/multitenant"
	"userclouds.com/internal/tenantmap"
	"userclouds.com/internal/uctest"
)

// Default profile fields for CreateUser
const (
	DefaultUserName   = "Default User"
	DefaultUserDomain = "contoso.com"
)

// CreateUser is a helper method that can be used by external tests to create a new IDP user.
// NOTE: storage is internal to IDP, so there's not an easy way to do this otherwise.
func CreateUser(t *testing.T,
	tenantDB *ucdb.DB,
	id uuid.UUID,
	orgID uuid.UUID,
	tenantID uuid.UUID,
	tenantURL string) (uuid.UUID, string) {

	if tenantID.IsNil() {
		// NBD because this is a test
		tenantID = uuid.Must(uuid.NewV4())
	}

	// This is necessary for the authz service to accept the requests in preSaveUser
	req, err := http.NewRequest("GET", "/test", nil)
	assert.NoErr(t, err)
	jwt := uctest.CreateJWT(t, oidc.UCTokenClaims{OrganizationID: orgID.String(),
		StandardClaims: oidc.StandardClaims{Audience: []string{tenantURL}}},
		tenantURL)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", jwt))
	tu, err := url.Parse(tenantURL)
	assert.NoErr(t, err)
	ctx := request.SetRequestData(
		multitenant.SetTenantState(context.Background(), &tenantmap.TenantState{TenantURL: tu}),
		req,
		uuid.Nil)

	s := NewStorage(ctx, t, tenantDB, tenantID)
	us := storage.NewUserStorage(ctx, tenantDB, "", tenantID)
	u := &storage.User{
		BaseUser: storage.BaseUser{
			VersionBaseModel: ucdb.NewVersionBaseWithID(id),
			OrganizationID:   orgID,
		},
	}
	assert.NoErr(t, us.SaveUser(ctx, u))
	email := fmt.Sprintf("%s@%s", id.String(), DefaultUserDomain)
	consentedValues := []storage.ColumnConsentedValue{
		{
			ColumnName: "email",
			Value:      email,
			Ordering:   1,
			ConsentedPurposes: []storage.ConsentedPurpose{
				{
					Purpose:          constants.OperationalPurposeID,
					RetentionTimeout: userstore.DataLifeCycleStateLive.GetDefaultRetentionTimeout(),
				},
			},
		},
		{
			ColumnName: "name",
			Value:      DefaultUserName,
			Ordering:   1,
			ConsentedPurposes: []storage.ConsentedPurpose{
				{
					Purpose:          constants.OperationalPurposeID,
					RetentionTimeout: userstore.DataLifeCycleStateLive.GetDefaultRetentionTimeout(),
				},
			},
		},
	}

	cm, err := storage.NewUserstoreColumnManager(ctx, s)
	assert.NoErr(t, err)
	sim, err := storage.NewSearchIndexManager(ctx, s)
	assert.NoErr(t, err)

	ucvs := []storage.UserColumnLiveValue{}
	for _, consentedValue := range consentedValues {
		column := cm.GetUserColumnByName(consentedValue.ColumnName)
		assert.NotNil(t, column)
		ucv, err := storage.NewUserColumnLiveValue(u.ID, column, &consentedValue)
		assert.NoErr(t, err)
		ucvs = append(ucvs, *ucv)
	}
	assert.NoErr(t, us.InsertUserColumnLiveValues(ctx, cm, sim, nil, ucvs))

	return id, email
}

// DeleteUser is a helper method that can be used by external tests to delete an IDP user.
// NOTE: storage is internal to IDP, so there's not an easy way to do this otherwise.
func DeleteUser(t *testing.T, tenantDB *ucdb.DB, id uuid.UUID, tenantID uuid.UUID) error {
	ctx := context.Background()
	s := storage.NewUserStorage(ctx, tenantDB, "", tenantID)
	return ucerr.Wrap(s.DeleteUser(ctx, id))
}

// NewStorage returns a new tenant storage instance for use in tests
func NewStorage(ctx context.Context, t *testing.T, tenantDB *ucdb.DB, tenantID uuid.UUID) *storage.Storage {
	return storage.New(ctx, tenantDB, tenantID, testhelpers.NewRedisConfigForTests())
}
