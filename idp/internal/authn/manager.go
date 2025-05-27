package authn

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/bcrypt"

	"userclouds.com/idp"
	"userclouds.com/idp/config"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	userstoreInternal "userclouds.com/idp/internal/userstore"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/oidc"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
)

// ErrUsernamePasswordIncorrect represents a bad username or password
var ErrUsernamePasswordIncorrect = ucerr.New("username or password incorrect")

// ErrUsernameNotFound represents a username that can't be found
var ErrUsernameNotFound = ucerr.New("username not found")

// Manager implements the "business logic" of the IDP
type Manager struct {
	configStorage          *storage.Storage
	userMultiRegionStorage *storage.UserMultiRegionStorage
}

// NewManager returns a new Manager object constructed from a DB.
func NewManager(configStorage *storage.Storage, userMultiRegionStorage *storage.UserMultiRegionStorage) *Manager {
	return &Manager{configStorage: configStorage, userMultiRegionStorage: userMultiRegionStorage}
}

// CheckUsernamePassword checks a username and password directly and returns the user on success
func (m *Manager) CheckUsernamePassword(ctx context.Context, username string, password string) (*storage.BaseUser, error) {
	authn, err := m.configStorage.GetPasswordAuthnForUsername(ctx, username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUsernameNotFound
		}
		return nil, ucerr.Wrap(err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(authn.Password), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return nil, ErrUsernamePasswordIncorrect
	} else if err != nil {
		return nil, ucerr.Wrap(err)
	}

	user, _, err := m.userMultiRegionStorage.GetBaseUser(ctx, authn.UserID, false)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return user, nil
}

// UpdateUsernamePassword updates the u/p for a login based (likely) a successful primary login
func (m *Manager) UpdateUsernamePassword(ctx context.Context, username string, password string) error {
	var authn *storage.PasswordAuthn
	authn, err := m.configStorage.GetPasswordAuthnForUsername(ctx, username)
	if err != nil {
		return ucerr.Wrap(err)
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		return ucerr.Wrap(err)
	}

	authn.Password = string(hashedPassword)

	return ucerr.Wrap(m.configStorage.SavePasswordAuthn(ctx, authn))
}

func coerceRecordToSchema(cm *storage.ColumnManager, record userstore.Record) error {
	for name, value := range record {
		if column := cm.GetUserColumnByName(name); column == nil {
			return ucerr.Friendlyf(nil, "Column `%s` doesn't exist", name)
		}

		if value == "" {
			record[name] = nil
		}
	}
	return nil
}

func (m *Manager) getMutatorValuesForCreate(ctx context.Context, req idp.CreateUserAndAuthnRequest) (map[string]idp.ValueAndPurposes, int, error) {
	mutator, err := m.configStorage.GetLatestMutator(ctx, constants.UpdateUserMutatorID)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	cm, err := storage.NewUserstoreColumnManager(ctx, m.configStorage)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	if req.Profile == nil {
		req.Profile = userstore.Record{}
	} else if err := coerceRecordToSchema(cm, req.Profile); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	values := map[string]idp.ValueAndPurposes{}

	for _, columnID := range mutator.ColumnIDs {
		c := cm.GetColumnByID(columnID)
		if c == nil {
			return nil, http.StatusInternalServerError, ucerr.Errorf("column %v not found", columnID)
		}

		switch c.Name {
		case "name":
			if req.Profile.StringValue("name") == "" {
				req.Profile["name"] = req.Username
			}
		case "nickname":
			if req.Profile.StringValue("nickname") == "" {
				splitName := strings.Split(req.Username, "@")
				if len(splitName) > 0 {
					req.Profile["nickname"] = splitName[0]
				} else {
					req.Profile["nickname"] = ""
				}
			}
		case "picture":
			if req.Profile.StringValue("picture") == "" {
				displayName := req.Profile.StringValue("name")
				if displayName == "" {
					displayName = req.Username
				}
				if email := req.Profile.StringValue("email"); email != "" {
					req.Profile["picture"] = GenerateDefaultAvatarURL(email, displayName).String()
				} else {
					req.Profile["picture"] = GenerateDefaultAvatarURL(displayName, displayName).String()
				}
			}
		default:
			if _, found := req.Profile[c.Name]; !found {
				if c.DefaultValue != "" {
					req.Profile[c.Name] = idp.MutatorColumnDefaultValue
				} else {
					req.Profile[c.Name] = idp.MutatorColumnCurrentValue
				}
			}
		}

		if c.Attributes.Constraints.PartialUpdates {
			values[c.Name] = idp.ValueAndPurposes{
				ValueAdditions:   req.Profile[c.Name],
				PurposeAdditions: []userstore.ResourceID{{ID: constants.OperationalPurposeID}},
			}
		} else {
			values[c.Name] = idp.ValueAndPurposes{
				Value:            req.Profile[c.Name],
				PurposeAdditions: []userstore.ResourceID{{ID: constants.OperationalPurposeID}},
			}
		}
	}

	return values, http.StatusOK, nil
}

// CreateUser just creates the user object itself, without any authn.
func (m *Manager) CreateUser(
	ctx context.Context,
	searchUpdateConfig *config.SearchUpdateConfig,
	req idp.CreateUserAndAuthnRequest,
) (*storage.BaseUser, int, error) {
	// get the mutator values that should be applied against the created user

	mutatorValues, code, err := m.getMutatorValuesForCreate(ctx, req)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	baseUser, code, err := userstoreInternal.CreateUserHelper(
		ctx,
		searchUpdateConfig,
		idp.CreateUserWithMutatorRequest{
			ID:             req.ID,
			MutatorID:      constants.UpdateUserMutatorID,
			Context:        policy.ClientContext{},
			RowData:        mutatorValues,
			OrganizationID: req.OrganizationID,
			Region:         req.Region,
		},
		len(mutatorValues) > 0, // empty mutator values means we don't need to execute UpdateUser mutator
	)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	return baseUser, http.StatusOK, nil
}

// CreateUserWithPassword creates a new user account and credential-based AuthN entry.
// TODO: detect/disallow duplicate email? Could be a per-tenant setting (default to disallow).
func (m *Manager) CreateUserWithPassword(
	ctx context.Context,
	searchUpdateConfig *config.SearchUpdateConfig,
	req idp.CreateUserAndAuthnRequest,
) (*storage.BaseUser, int, error) {
	u, code, err := m.CreateUser(ctx, searchUpdateConfig, req)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	if err := m.AddPasswordAuthnToUser(ctx, u.ID, req.Username, req.Password); err != nil {
		// Try and delete user we just created but ignore the error if it fails.
		// This codepath can be hit due to random infra failures but, more commonly,
		// due to a duplicate username, in which case we should back out the user too.
		// TODO: might be nice to use a transaction to do all of this at once?
		ignoreError := m.userMultiRegionStorage.DeleteUser(ctx, u.ID)
		_ = ignoreError // lint: errcheck safe :)
		return nil, uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	return u, http.StatusOK, nil
}

// AddPasswordAuthnToUser adds username/password authentication to an existing user
func (m *Manager) AddPasswordAuthnToUser(ctx context.Context, userID uuid.UUID, username string, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ucerr.Wrap(err)
	}

	authn := &storage.PasswordAuthn{
		UserBaseModel: ucdb.NewUserBase(userID),
		Username:      username,
		Password:      string(hashedPassword),
	}

	return ucerr.Wrap(m.configStorage.SavePasswordAuthn(ctx, authn))
}

// CreateUserWithOIDCLogin creates a new user account with a 3rd party OIDC AuthN entry.
// TODO: we don't handle email duplication (e.g. FB and Google login with same email) but probably ought to
// have the option to flag this based on tenant config.
// TODO: we don't yet log/track access times, session lengths, etc.
func (m *Manager) CreateUserWithOIDCLogin(
	ctx context.Context,
	searchUpdateConfig *config.SearchUpdateConfig,
	req idp.CreateUserAndAuthnRequest,
) (*storage.BaseUser, int, error) {
	u, code, err := m.CreateUser(ctx, searchUpdateConfig, req)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	oidcAuthn := &storage.OIDCAuthn{
		UserBaseModel: ucdb.NewUserBase(u.ID),
		Type:          req.OIDCProvider,
		OIDCIssuerURL: req.OIDCIssuerURL,
		OIDCSubject:   req.OIDCSubject,
	}

	if err := m.configStorage.SaveOIDCAuthn(ctx, oidcAuthn); err != nil {
		// Try and delete user we just created but ignore the error if it fails.
		// This codepath can be hit due to random infra failures but, more commonly,
		// due to a duplicate type + OIDC subject, in which case we should back out the user too.
		// TODO: might be nice to use a transaction to do all of this at once?
		ignoreError := m.userMultiRegionStorage.DeleteUser(ctx, u.ID)
		_ = ignoreError // lint: errcheck safe
		return nil, uchttp.SQLWriteErrorMapper(err), ucerr.Wrap(err)
	}

	return u, http.StatusOK, nil
}

// AddOIDCAuthnToUser adds an OIDC provider as an authenticator to an existing user
func (m *Manager) AddOIDCAuthnToUser(ctx context.Context, userID uuid.UUID, provider oidc.ProviderType, issuerURL string, oidcSubject string) error {
	if !provider.IsSupported() {
		return ucerr.New("unsupported OIDC provider")
	}

	oidcAuthn := &storage.OIDCAuthn{
		UserBaseModel: ucdb.NewUserBase(userID),
		Type:          provider,
		OIDCIssuerURL: issuerURL,
		OIDCSubject:   oidcSubject,
	}

	return ucerr.Wrap(m.configStorage.SaveOIDCAuthn(ctx, oidcAuthn))
}
