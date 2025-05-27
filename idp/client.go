package idp

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/paths"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/jsonclient"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/sdkclient"
	"userclouds.com/infra/ucerr"
)

type options struct {
	ifNotExists         bool
	debug               bool
	organizationID      uuid.UUID
	userID              uuid.UUID
	dataRegion          region.DataRegion
	accessPrimaryDBOnly bool
	paginationOptions   []pagination.Option
	jsonclientOptions   []jsonclient.Option
}

// Option makes idp.Client extensible
type Option interface {
	apply(*options)
}

type optFunc func(*options)

func (o optFunc) apply(opts *options) {
	o(opts)
}

// IfNotExists returns an Option that will cause the client not to return an error if an identical object to the one being created already exists
func IfNotExists() Option {
	return optFunc(func(opts *options) {
		opts.ifNotExists = true
	})
}

// Debug returns an Option that will cause the client to return debug information, if available
func Debug() Option {
	return optFunc(func(opts *options) {
		opts.debug = true
	})
}

// OrganizationID returns an Option that will cause the client to use the specified organization ID for the request
func OrganizationID(organizationID uuid.UUID) Option {
	return optFunc(func(opts *options) {
		opts.organizationID = organizationID
	})
}

// UserID returns an Option that will cause the client to use the specified user ID for the create user request
func UserID(userID uuid.UUID) Option {
	return optFunc(func(opts *options) {
		opts.userID = userID
	})
}

// DataRegion returns an Option that will cause the client to use the specified region for the request
func DataRegion(dataRegion region.DataRegion) Option {
	return optFunc(func(opts *options) {
		opts.dataRegion = dataRegion
	})
}

// AccessPrimaryDBOnly returns an Option that will cause the client to use the primary database only for the request
func AccessPrimaryDBOnly() Option {
	return optFunc(func(opts *options) {
		opts.accessPrimaryDBOnly = true
	})
}

// Pagination is a wrapper around pagination.Option
func Pagination(opt ...pagination.Option) Option {
	return optFunc(func(opts *options) {
		opts.paginationOptions = append(opts.paginationOptions, opt...)
	})
}

// JSONClient is a wrapper around jsonclient.Option
func JSONClient(opt ...jsonclient.Option) Option {
	return optFunc(func(opts *options) {
		opts.jsonclientOptions = append(opts.jsonclientOptions, opt...)
	})
}

// Client represents a client to talk to the Userclouds IDP
type Client struct {
	*TokenizerClient

	client  *sdkclient.Client
	options options
}

// NewClient constructs a new IDP client
func NewClient(url string, opts ...Option) (*Client, error) {

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	c := &Client{
		client:  sdkclient.New(url, "idp", options.jsonclientOptions...),
		options: options,
	}
	tc := &TokenizerClient{client: c.client, options: options}
	c.TokenizerClient = tc

	if err := c.client.ValidateBearerTokenHeader(); err != nil {
		return nil, ucerr.Wrap(err)
	}
	return c, nil
}

// CreateUserAndAuthnRequest creates a user on the IDP
type CreateUserAndAuthnRequest struct {
	ID uuid.UUID `json:"id"`

	Profile userstore.Record `json:"profile"`

	OrganizationID uuid.UUID `json:"organization_id"`

	UserAuthn `validate:"skip"`

	Region region.DataRegion `json:"region"`
}

//go:generate genvalidate CreateUserAndAuthnRequest

// UserResponse is the response body for methods which return user data.
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	UpdatedAt int64     `json:"updated_at"` // seconds since the Unix Epoch (UTC)

	Profile userstore.Record `json:"profile"`

	OrganizationID uuid.UUID `json:"organization_id"`
}

// AuthN user methods

// CreateUser creates a user without authn. Profile is optional (okay to pass nil)
func (c *Client) CreateUser(ctx context.Context, profile userstore.Record, opts ...Option) (uuid.UUID, error) {
	// TODO: we don't validate the profile here, since we don't require email in this path
	// this probably should be refactored to be more consistent in this client

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateUserAndAuthnRequest{
		ID:             options.userID,
		Profile:        profile,
		OrganizationID: options.organizationID,
		Region:         options.dataRegion,
	}

	var res UserResponse
	if err := c.client.Post(ctx, paths.CreateUser, req, &res); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	return res.ID, nil
}

// GetUser gets a user by ID
func (c *Client) GetUser(ctx context.Context, id uuid.UUID) (*UserResponse, error) {

	requestURL := url.URL{
		Path: fmt.Sprintf("/authn/users/%s", id),
	}

	var res UserResponse
	if err := c.client.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdateUserRequest optionally updates some or all mutable fields of a user struct.
// Pointers are used to distinguish between unset vs. set to default value (false, "", etc).
// TODO: should we allow changing Email? That's a more complex one as there are more implications to
// changing email that may affect AuthNs and security (e.g. account hijacking, unverified emails, etc).
type UpdateUserRequest struct {
	// Only fields set in the underlying map will be updated
	Profile userstore.Record `json:"profile"`

	Region region.DataRegion `json:"region"`
}

//go:generate genvalidate UpdateUserRequest

// UpdateUser updates user profile data for a given user ID
func (c *Client) UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*UserResponse, error) {
	requestURL := url.URL{
		Path: fmt.Sprintf("/authn/users/%s", id),
	}

	var resp UserResponse

	if err := c.client.Put(ctx, requestURL.String(), &req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DeleteUser deletes a user by ID
func (c *Client) DeleteUser(ctx context.Context, id uuid.UUID) error {
	requestURL := url.URL{
		Path: fmt.Sprintf("/authn/users/%s", id),
	}

	return ucerr.Wrap(c.client.Delete(ctx, requestURL.String(), nil))
}

// ListUsersResponse is the paginated response from listing users.
type ListUsersResponse struct {
	Data []UserResponse `json:"data"`
	pagination.ResponseFields
}

// ListUsers lists all users
func (c *Client) ListUsers(ctx context.Context, opts ...Option) (*ListUsersResponse, error) {

	var options options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	var res ListUsersResponse
	query := pager.Query()
	if options.organizationID != uuid.Nil {
		query.Set("organization_id", options.organizationID.String())
	}
	requestURL := url.URL{
		Path:     "/authn/users",
		RawQuery: query.Encode(),
	}

	if err := c.client.Get(ctx, requestURL.String(), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// ListUserRegions gets the regions available to be passed in as a region for user creation
func (c *Client) ListUserRegions(ctx context.Context) ([]region.DataRegion, error) {
	var regions []region.DataRegion
	if err := c.client.Get(ctx, paths.ListUserRegionsPath, &regions); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return regions, nil
}

// CreateDatabaseRequest is the request body for creating a new database
type CreateDatabaseRequest struct {
	Database userstore.SQLShimDatabase `json:"database"`
}

//go:generate genvalidate CreateDatabaseRequest

// CreateDatabase creates a new sqlshim database for the tenant
func (c *Client) CreateDatabase(ctx context.Context, database userstore.SQLShimDatabase, opts ...Option) (*userstore.SQLShimDatabase, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateDatabaseRequest{
		Database: database,
	}

	var resp userstore.SQLShimDatabase
	if options.ifNotExists {
		exists, existingID, err := c.client.CreateIfNotExists(ctx, paths.CreateDatabasePath, req, &resp)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if exists {
			resp = req.Database
			resp.ID = existingID
		}
	} else {
		if err := c.client.Post(ctx, paths.CreateDatabasePath, req, &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &resp, nil
}

// DeleteDatabase deletes the database specified by the database ID for the associated tenant
func (c *Client) DeleteDatabase(ctx context.Context, databaseID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.DeleteDatabasePath(databaseID), nil))
}

// GetDatabase returns the database specified by the database ID for the associated tenant
func (c *Client) GetDatabase(ctx context.Context, databaseID uuid.UUID) (*userstore.SQLShimDatabase, error) {
	var resp userstore.SQLShimDatabase
	if err := c.client.Get(ctx, paths.GetDatabasePath(databaseID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListDatabasesResponse is the paginated response struct for listing databases
type ListDatabasesResponse struct {
	Data []userstore.SQLShimDatabase `json:"data"`
	pagination.ResponseFields
}

// ListDatabases lists all databases for the associated tenant
func (c *Client) ListDatabases(ctx context.Context, opts ...Option) (*ListDatabasesResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()
	var res ListDatabasesResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.DatabasePath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdateDatabaseRequest is the request body for updating a database
type UpdateDatabaseRequest struct {
	Database userstore.SQLShimDatabase `json:"database"`
}

//go:generate genvalidate UpdateDatabaseRequest

// UpdateDatabase updates the database specified by the database ID with the specified data for the associated tenant
func (c *Client) UpdateDatabase(ctx context.Context, databaseID uuid.UUID, updatedDatabase userstore.SQLShimDatabase) (*userstore.SQLShimDatabase, error) {
	req := UpdateDatabaseRequest{
		Database: updatedDatabase,
	}

	var resp userstore.SQLShimDatabase
	if err := c.client.Put(ctx, paths.UpdateDatabasePath(databaseID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// TestDatabaseRequest is the request body for testing a database connection
type TestDatabaseRequest struct {
	Database userstore.SQLShimDatabase `json:"database"`
}

//go:generate genvalidate TestDatabaseRequest

// TestDatabaseResponse is the response body for testing a database connection
type TestDatabaseResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// TestDatabase tests the connection for the specified database
func (c *Client) TestDatabase(ctx context.Context, database userstore.SQLShimDatabase) (*TestDatabaseResponse, error) {
	req := TestDatabaseRequest{
		Database: database,
	}

	var resp TestDatabaseResponse
	if err := c.client.Post(ctx, paths.TestDatabasePath, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// CreateObjectStoreRequest is the request body for creating a new object store
type CreateObjectStoreRequest struct {
	ObjectStore userstore.ShimObjectStore `json:"objectStore"`
}

//go:generate genvalidate CreateObjectStoreRequest

// CreateObjectStore creates a new sqlshim object store for the tenant
func (c *Client) CreateObjectStore(ctx context.Context, objectStore userstore.ShimObjectStore, opts ...Option) (*userstore.ShimObjectStore, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateObjectStoreRequest{
		ObjectStore: objectStore,
	}

	var resp userstore.ShimObjectStore
	if options.ifNotExists {
		exists, existingID, err := c.client.CreateIfNotExists(ctx, paths.CreateObjectStorePath, req, &resp)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if exists {
			resp = req.ObjectStore
			resp.ID = existingID
		}
	} else {
		if err := c.client.Post(ctx, paths.CreateObjectStorePath, req, &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &resp, nil
}

// DeleteObjectStore deletes the object store specified by the object store ID for the associated tenant
func (c *Client) DeleteObjectStore(ctx context.Context, objectStoreID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.DeleteObjectStorePath(objectStoreID), nil))
}

// GetObjectStore returns the object store specified by the object store ID for the associated tenant
func (c *Client) GetObjectStore(ctx context.Context, objectStoreID uuid.UUID) (*userstore.ShimObjectStore, error) {
	var resp userstore.ShimObjectStore
	if err := c.client.Get(ctx, paths.GetObjectStorePath(objectStoreID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListObjectStoresResponse is the paginated response struct for listing object stores
type ListObjectStoresResponse struct {
	Data []userstore.ShimObjectStore `json:"data"`
	pagination.ResponseFields
}

// ListObjectStores lists all object stores for the associated tenant
func (c *Client) ListObjectStores(ctx context.Context, opts ...Option) (*ListObjectStoresResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()
	var res ListObjectStoresResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.ObjectStorePath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdateObjectStoreRequest is the request body for updating a object store
type UpdateObjectStoreRequest struct {
	ObjectStore userstore.ShimObjectStore `json:"objectStore"`
}

//go:generate genvalidate UpdateObjectStoreRequest

// UpdateObjectStore updates the object store specified by the object store ID with the specified data for the associated tenant
func (c *Client) UpdateObjectStore(ctx context.Context, objectStoreID uuid.UUID, updatedObjectStore userstore.ShimObjectStore) (*userstore.ShimObjectStore, error) {
	req := UpdateObjectStoreRequest{
		ObjectStore: updatedObjectStore,
	}

	var resp userstore.ShimObjectStore
	if err := c.client.Put(ctx, paths.UpdateObjectStorePath(objectStoreID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// CreateDataTypeRequest is the request body for creating a new data type
type CreateDataTypeRequest struct {
	DataType userstore.ColumnDataType `json:"data_type"`
}

//go:generate genvalidate CreateDataTypeRequest

// CreateDataType creates a new data type for the associated tenant
func (c *Client) CreateDataType(
	ctx context.Context,
	dataType userstore.ColumnDataType,
	opts ...Option,
) (*userstore.ColumnDataType, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateDataTypeRequest{
		DataType: dataType,
	}

	var resp userstore.ColumnDataType
	if options.ifNotExists {
		exists, existingID, err := c.client.CreateIfNotExists(ctx, paths.DataTypePath, req, &resp)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if exists {
			resp = req.DataType
			resp.ID = existingID
		}
	} else {
		if err := c.client.Post(ctx, paths.DataTypePath, req, &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &resp, nil
}

// DeleteDataType deletes the data type specified by the data type ID for the associated tenant
func (c *Client) DeleteDataType(ctx context.Context, dataTypeID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.DataTypePathForID(dataTypeID), nil))
}

// GetDataType returns the data type specified by the data type ID for the associated tenant
func (c *Client) GetDataType(ctx context.Context, dataTypeID uuid.UUID) (*userstore.ColumnDataType, error) {
	var resp userstore.ColumnDataType
	if err := c.client.Get(ctx, paths.DataTypePathForID(dataTypeID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListDataTypesResponse is the paginated response struct for listing data types
type ListDataTypesResponse struct {
	Data []userstore.ColumnDataType `json:"data"`
	pagination.ResponseFields
}

// ListDataTypes lists all data types for the associated tenant
func (c *Client) ListDataTypes(ctx context.Context, opts ...Option) (*ListDataTypesResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	query := pager.Query()

	var res ListDataTypesResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.DataTypePath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdateDataTypeRequest is the request body for updating a data type
type UpdateDataTypeRequest struct {
	DataType userstore.ColumnDataType `json:"data_type"`
}

//go:generate genvalidate UpdateDataTypeRequest

// UpdateDataType updates the data type specified by the data type ID with the specified data for the associated tenant
func (c *Client) UpdateDataType(ctx context.Context, dataTypeID uuid.UUID, updatedDataType userstore.ColumnDataType) (*userstore.ColumnDataType, error) {
	req := UpdateDataTypeRequest{
		DataType: updatedDataType,
	}

	var resp userstore.ColumnDataType
	if err := c.client.Put(ctx, paths.DataTypePathForID(dataTypeID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// CreateColumnRequest is the request body for creating a new column
// TODO: should this support multiple at once before we ship this API?
type CreateColumnRequest struct {
	Column userstore.Column `json:"column"`
}

//go:generate genvalidate CreateColumnRequest

// CreateColumn creates a new column for the associated tenant
func (c *Client) CreateColumn(ctx context.Context, column userstore.Column, opts ...Option) (*userstore.Column, error) {

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateColumnRequest{
		Column: column,
	}

	var resp userstore.Column
	if options.ifNotExists {
		exists, existingID, err := c.client.CreateIfNotExists(ctx, paths.CreateColumnPath, req, &resp)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if exists {
			resp = req.Column
			resp.ID = existingID
		}
	} else {
		if err := c.client.Post(ctx, paths.CreateColumnPath, req, &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &resp, nil
}

// DeleteColumn deletes the column specified by the column ID for the associated tenant
func (c *Client) DeleteColumn(ctx context.Context, columnID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.DeleteColumnPath(columnID), nil))
}

// GetColumn returns the column specified by the column ID for the associated tenant
func (c *Client) GetColumn(ctx context.Context, columnID uuid.UUID) (*userstore.Column, error) {
	var resp userstore.Column
	if err := c.client.Get(ctx, paths.GetColumnPath(columnID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListColumnsResponse is the paginated response struct for listing columns
type ListColumnsResponse struct {
	Data []userstore.Column `json:"data"`
	pagination.ResponseFields
}

// ListColumns lists all columns for the associated tenant
func (c *Client) ListColumns(ctx context.Context, opts ...Option) (*ListColumnsResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}
	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()

	var res ListColumnsResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.ListColumnsPath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdateColumnRequest is the request body for updating a column
type UpdateColumnRequest struct {
	Column userstore.Column `json:"column"`
}

//go:generate genvalidate UpdateColumnRequest

// UpdateColumn updates the column specified by the column ID with the specified data for the associated tenant
func (c *Client) UpdateColumn(ctx context.Context, columnID uuid.UUID, updatedColumn userstore.Column) (*userstore.Column, error) {
	req := UpdateColumnRequest{
		Column: updatedColumn,
	}

	var resp userstore.Column
	if err := c.client.Put(ctx, paths.UpdateColumnPath(columnID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DurationUnit identifies the unit of measurement for a duration
type DurationUnit string

// Supported duration units
const (
	DurationUnitIndefinite DurationUnit = "indefinite"
	DurationUnitYear       DurationUnit = "year"
	DurationUnitMonth      DurationUnit = "month"
	DurationUnitWeek       DurationUnit = "week"
	DurationUnitDay        DurationUnit = "day"
	DurationUnitHour       DurationUnit = "hour"
)

//go:generate genconstant DurationUnit

// RetentionDuration represents a duration with a specific duration unit
type RetentionDuration struct {
	Unit     DurationUnit `json:"unit"`
	Duration int          `json:"duration"`
}

func (d *RetentionDuration) extraValidate() error {
	if d.Duration < 0 {
		return ucerr.New("Duration must be non-negative")
	}

	if d.Unit == DurationUnitIndefinite && d.Duration != 0 {
		return ucerr.New("Duration must be 0 if Unit is DurationUnitIndefinite")
	}

	return nil
}

//go:generate genvalidate RetentionDuration

// AddToTime will add the retention duration to a passed in time
func (d RetentionDuration) AddToTime(t time.Time) time.Time {
	switch d.Unit {
	case DurationUnitIndefinite:
		return userstore.GetRetentionTimeoutIndefinite()
	case DurationUnitYear:
		return t.AddDate(d.Duration, 0, 0)
	case DurationUnitMonth:
		return t.AddDate(0, d.Duration, 0)
	case DurationUnitWeek:
		return t.AddDate(0, 0, 7*d.Duration)
	case DurationUnitDay:
		return t.AddDate(0, 0, d.Duration)
	case DurationUnitHour:
		return t.Add(time.Duration(d.Duration) * time.Hour)
	}

	return t
}

// LessThan returns true if the duration is strictly smaller than other
func (d RetentionDuration) LessThan(other RetentionDuration) bool {
	var t time.Time
	return d.AddToTime(t).Before(other.AddToTime(t))
}

// ColumnRetentionDuration represents an identified retention duration. If ID is nil, it
// represents an inherited or new value. UseDefault set to true means that the duration is
// inherited from a less specific default value. DefaultDuration represents the duration
// that would be inherited if a specific value is not set for the retention duration identifier.
type ColumnRetentionDuration struct {
	DurationType    userstore.DataLifeCycleState `json:"duration_type"`
	ID              uuid.UUID                    `json:"id"`
	ColumnID        uuid.UUID                    `json:"column_id"`
	PurposeID       uuid.UUID                    `json:"purpose_id"`
	Duration        RetentionDuration            `json:"duration"`
	UseDefault      bool                         `json:"use_default"`
	Version         int                          `json:"version"`
	DefaultDuration *RetentionDuration           `json:"default_duration" validate:"allownil"`
	PurposeName     *string                      `json:"purpose_name" validate:"allownil,notempty"`
}

//go:generate genvalidate ColumnRetentionDuration

// UpdateColumnRetentionDurationRequest is is used to update a single retention duration for a column.
// The retention duration must have UseDefault set to false. ID must be nil for a creation request,
// and non-nil for an update request.
type UpdateColumnRetentionDurationRequest struct {
	RetentionDuration ColumnRetentionDuration `json:"retention_duration"`
}

//go:generate genvalidate UpdateColumnRetentionDurationRequest

// UpdateColumnRetentionDurationsRequest is used to update a collection of retention durations
// for a column. If ID for a retention duration is non-nil, that retention duration will be
// updated if UseDefault is set to false, or deleted if UseDefault is set to true.  If ID is nil,
// the associated retention duration will be inserted.
type UpdateColumnRetentionDurationsRequest struct {
	RetentionDurations []ColumnRetentionDuration `json:"retention_durations"`
}

//go:generate genvalidate UpdateColumnRetentionDurationsRequest

// ColumnRetentionDurationResponse is the response to a get or update request for a single
// retention duration.  The retention duration that applies for the request will be returned,
// and will include both the specified and inherited default duration. In addition, a max allowed
// retention duration appropriate for the request parameters will be included. The retention
// duration will have a non-nil ID and have UseDefault set to false if it represents
// a saved value, or a nil ID and UseDefault set to true if it represents an inherited value.
type ColumnRetentionDurationResponse struct {
	MaxDuration       RetentionDuration       `json:"max_duration"`
	RetentionDuration ColumnRetentionDuration `json:"retention_duration"`
}

// ColumnRetentionDurationsResponse is the response to a get or update request for a set of
// retention durations. The set of retention durations that apply for the request will be
// returned, each of which will include a specified and inherited default duration. In addition,
// a max allowed retention duration appropriate for the request parameters will be included. Each
// of the retention durations will have a non-nil ID and have UseDefault set to false if they
// are saved values, or a nil ID and UseDefault set to true if they represent an inherited value.
type ColumnRetentionDurationsResponse struct {
	MaxDuration        RetentionDuration         `json:"max_duration"`
	RetentionDurations []ColumnRetentionDuration `json:"retention_durations"`
}

// CreateColumnRetentionDurationForPurpose creates a column retention duration
// for the specified duration type and purpose, failing if a retention duration
// already exists and returning the derived retention duration upon success.
func (c *Client) CreateColumnRetentionDurationForPurpose(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	purposeID uuid.UUID,
	crd ColumnRetentionDuration,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForPurpose(purposeID).Build()

	req := UpdateColumnRetentionDurationRequest{RetentionDuration: crd}

	var resp ColumnRetentionDurationResponse
	if err := c.client.Post(ctx, path, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// CreateColumnRetentionDurationForTenant creates a column retention duration
// for the specified duration type and tenant, failing if a retention duration
// already exists and returning the derived retention duration upon success.
func (c *Client) CreateColumnRetentionDurationForTenant(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	crd ColumnRetentionDuration,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForTenant().Build()

	req := UpdateColumnRetentionDurationRequest{RetentionDuration: crd}

	var resp ColumnRetentionDurationResponse
	if err := c.client.Post(ctx, path, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DeleteColumnRetentionDurationForColumn deletes the specified column retention duration
func (c *Client) DeleteColumnRetentionDurationForColumn(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	columnID uuid.UUID,
	durationID uuid.UUID,
) error {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForColumn(columnID).ForDuration(durationID).Build()
	return ucerr.Wrap(c.client.Delete(ctx, path, nil))
}

// DeleteColumnRetentionDurationForPurpose deletes the specified purpose retention duration
func (c *Client) DeleteColumnRetentionDurationForPurpose(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	purposeID uuid.UUID,
	durationID uuid.UUID,
) error {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForPurpose(purposeID).ForDuration(durationID).Build()
	return ucerr.Wrap(c.client.Delete(ctx, path, nil))
}

// DeleteColumnRetentionDurationForTenant deletes the specified tenant retention duration
func (c *Client) DeleteColumnRetentionDurationForTenant(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	durationID uuid.UUID,
) error {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForTenant().ForDuration(durationID).Build()
	return ucerr.Wrap(c.client.Delete(ctx, path, nil))
}

// GetColumnRetentionDurationsForColumn returns the derived column and purpose retention durations for the specified column and duration type
func (c *Client) GetColumnRetentionDurationsForColumn(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	columnID uuid.UUID,
) (*ColumnRetentionDurationsResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForColumn(columnID).Build()

	var resp ColumnRetentionDurationsResponse
	if err := c.client.Get(ctx, path, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// GetColumnRetentionDurationForPurpose returns the derived purpose retention duration for the specified purpose and duration type
func (c *Client) GetColumnRetentionDurationForPurpose(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	purposeID uuid.UUID,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForPurpose(purposeID).Build()

	var resp ColumnRetentionDurationResponse
	if err := c.client.Get(ctx, path, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// GetColumnRetentionDurationForTenant returns the derived tenant retention duration for the specified duration type
func (c *Client) GetColumnRetentionDurationForTenant(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForTenant().Build()

	var resp ColumnRetentionDurationResponse
	if err := c.client.Get(ctx, path, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// GetSpecificColumnRetentionDurationForColumn gets the specified column retention duration
func (c *Client) GetSpecificColumnRetentionDurationForColumn(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	columnID uuid.UUID,
	durationID uuid.UUID,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForColumn(columnID).ForDuration(durationID).Build()

	var resp ColumnRetentionDurationResponse
	if err := c.client.Get(ctx, path, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// GetSpecificColumnRetentionDurationForPurpose gets the specified purpose retention duration
func (c *Client) GetSpecificColumnRetentionDurationForPurpose(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	purposeID uuid.UUID,
	durationID uuid.UUID,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForPurpose(purposeID).ForDuration(durationID).Build()

	var resp ColumnRetentionDurationResponse
	if err := c.client.Get(ctx, path, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// GetSpecificColumnRetentionDurationForTenant gets the specified tenant retention duration
func (c *Client) GetSpecificColumnRetentionDurationForTenant(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	durationID uuid.UUID,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForTenant().ForDuration(durationID).Build()

	var resp ColumnRetentionDurationResponse
	if err := c.client.Get(ctx, path, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// UpdateColumnRetentionDurationsForColumn updates the column retention durations
// for the specified column and duration type, returning the updated set
// of retention durations for the column and duration type.
func (c *Client) UpdateColumnRetentionDurationsForColumn(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	columnID uuid.UUID,
	req UpdateColumnRetentionDurationsRequest,
) (*ColumnRetentionDurationsResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForColumn(columnID).Build()

	var resp ColumnRetentionDurationsResponse
	if err := c.client.Post(ctx, path, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// UpdateSpecificColumnRetentionDurationForColumn updates the specific column
// retention duration for the specified column and duration type, returning the updated
// retention duration upon success.
func (c *Client) UpdateSpecificColumnRetentionDurationForColumn(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	columnID uuid.UUID,
	durationID uuid.UUID,
	crd ColumnRetentionDuration,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForColumn(columnID).ForDuration(durationID).Build()

	req := UpdateColumnRetentionDurationRequest{RetentionDuration: crd}

	var resp ColumnRetentionDurationResponse
	if err := c.client.Put(ctx, path, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// UpdateSpecificColumnRetentionDurationForPurpose updates the specific column
// retention duration for the specified purpose and duration type, returning the updated
// retention duration upon success.
func (c *Client) UpdateSpecificColumnRetentionDurationForPurpose(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	crd ColumnRetentionDuration,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForPurpose(purposeID).ForDuration(durationID).Build()

	req := UpdateColumnRetentionDurationRequest{RetentionDuration: crd}

	var resp ColumnRetentionDurationResponse
	if err := c.client.Put(ctx, path, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// UpdateSpecificColumnRetentionDurationForTenant updates the specific column
// retention duration for the tenant and specified duration type, returning the updated
// retention duration upon success.
func (c *Client) UpdateSpecificColumnRetentionDurationForTenant(
	ctx context.Context,
	dlcs userstore.DataLifeCycleState,
	durationID uuid.UUID,
	crd ColumnRetentionDuration,
) (*ColumnRetentionDurationResponse, error) {
	path := paths.NewRetentionPath(dlcs.IsLive()).ForTenant().ForDuration(durationID).Build()

	req := UpdateColumnRetentionDurationRequest{RetentionDuration: crd}

	var resp ColumnRetentionDurationResponse
	if err := c.client.Put(ctx, path, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// CreateAccessorRequest is the request body for creating a new accessor
type CreateAccessorRequest struct {
	Accessor userstore.Accessor `json:"accessor"`
}

//go:generate genvalidate CreateAccessorRequest

// CreateAccessor creates a new accessor for the associated tenant
func (c *Client) CreateAccessor(ctx context.Context, fa userstore.Accessor, opts ...Option) (*userstore.Accessor, error) {

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateAccessorRequest{
		Accessor: fa,
	}

	var resp userstore.Accessor
	if options.ifNotExists {
		exists, existingID, err := c.client.CreateIfNotExists(ctx, paths.CreateAccessorPath, req, &resp)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if exists {
			resp = req.Accessor
			resp.ID = existingID
		}
	} else {
		if err := c.client.Post(ctx, paths.CreateAccessorPath, req, &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &resp, nil
}

// DeleteAccessor deletes the accessor specified by the accessor ID for the associated tenant
func (c *Client) DeleteAccessor(ctx context.Context, accessorID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.DeleteAccessorPath(accessorID), nil))
}

// GetAccessor returns the accessor specified by the accessor ID for the associated tenant
func (c *Client) GetAccessor(ctx context.Context, accessorID uuid.UUID) (*userstore.Accessor, error) {
	var resp userstore.Accessor
	if err := c.client.Get(ctx, paths.GetAccessorPath(accessorID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// GetAccessorByVersion returns the version of an accessor specified by the accessor ID and version for the associated tenant
func (c *Client) GetAccessorByVersion(ctx context.Context, accessorID uuid.UUID, version int) (*userstore.Accessor, error) {
	var resp userstore.Accessor
	if err := c.client.Get(ctx, paths.GetAccessorByVersionPath(accessorID, version), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListAccessorsResponse is the paginated response from listing accessors.
type ListAccessorsResponse struct {
	Data []userstore.Accessor `json:"data"`
	pagination.ResponseFields
}

// ListAccessors lists all the available accessors for the associated tenant
func (c *Client) ListAccessors(ctx context.Context, versioned bool, opts ...Option) (*ListAccessorsResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}
	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()
	query.Add("versioned", strconv.FormatBool(versioned))

	var res ListAccessorsResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.ListAccessorsPath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdateAccessorRequest is the request body for updating an accessor
type UpdateAccessorRequest struct {
	Accessor userstore.Accessor `json:"accessor"`
}

//go:generate genvalidate UpdateAccessorRequest

// UpdateAccessor updates the accessor specified by the accessor ID with the specified data for the associated tenant
func (c *Client) UpdateAccessor(ctx context.Context, accessorID uuid.UUID, updatedAccessor userstore.Accessor) (*userstore.Accessor, error) {
	req := UpdateAccessorRequest{
		Accessor: updatedAccessor,
	}

	var resp userstore.Accessor
	if err := c.client.Put(ctx, paths.UpdateAccessorPath(accessorID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// CreateMutatorRequest is the request body for creating a new mutator
type CreateMutatorRequest struct {
	Mutator userstore.Mutator `json:"mutator"`
}

//go:generate genvalidate CreateMutatorRequest

// CreateMutator creates a new mutator for the associated tenant
func (c *Client) CreateMutator(ctx context.Context, fa userstore.Mutator, opts ...Option) (*userstore.Mutator, error) {

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateMutatorRequest{
		Mutator: fa,
	}

	var resp userstore.Mutator
	if options.ifNotExists {
		exists, existingID, err := c.client.CreateIfNotExists(ctx, paths.CreateMutatorPath, req, &resp)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if exists {
			resp = req.Mutator
			resp.ID = existingID
		}
	} else {
		if err := c.client.Post(ctx, paths.CreateMutatorPath, req, &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &resp, nil
}

// DeleteMutator deletes the mutator specified by the mutator ID for the associated tenant
func (c *Client) DeleteMutator(ctx context.Context, mutatorID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.DeleteMutatorPath(mutatorID), nil))
}

// GetMutator returns the mutator specified by the mutator ID for the associated tenant
func (c *Client) GetMutator(ctx context.Context, mutatorID uuid.UUID) (*userstore.Mutator, error) {
	var resp userstore.Mutator
	if err := c.client.Get(ctx, paths.GetMutatorPath(mutatorID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// GetMutatorByVersion returns the version of an mutator specified by the mutator ID and version for the associated tenant
func (c *Client) GetMutatorByVersion(ctx context.Context, mutatorID uuid.UUID, version int) (*userstore.Mutator, error) {
	var resp userstore.Mutator
	if err := c.client.Get(ctx, paths.GetMutatorByVersionPath(mutatorID, version), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListMutatorsResponse is the paginated response from listing mutators.
type ListMutatorsResponse struct {
	Data []userstore.Mutator `json:"data"`
	pagination.ResponseFields
}

// ListMutators lists all the available mutators for the associated tenant
func (c *Client) ListMutators(ctx context.Context, versioned bool, opts ...Option) (*ListMutatorsResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}
	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	query := pager.Query()
	query.Add("versioned", strconv.FormatBool(versioned))

	var res ListMutatorsResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.ListMutatorsPath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdateMutatorRequest is the request body for updating a mutator
type UpdateMutatorRequest struct {
	Mutator userstore.Mutator `json:"mutator"`
}

//go:generate genvalidate UpdateMutatorRequest

// UpdateMutator updates the mutator specified by the mutator ID with the specified data for the associated tenant
func (c *Client) UpdateMutator(ctx context.Context, mutatorID uuid.UUID, updatedMutator userstore.Mutator) (*userstore.Mutator, error) {
	req := UpdateMutatorRequest{
		Mutator: updatedMutator,
	}

	var resp userstore.Mutator
	if err := c.client.Put(ctx, paths.UpdateMutatorPath(mutatorID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ExecuteAccessorRequest is the request body for accessing user data
type ExecuteAccessorRequest struct {
	AccessorID          uuid.UUID                    `json:"accessor_id" validate:"notnil"` // the accessor that specifies what data to access
	Context             policy.ClientContext         `json:"context"`                       // context that is provided to the accessor Access Policy
	SelectorValues      userstore.UserSelectorValues `json:"selector_values"`               // the values to use for the selector
	Region              region.DataRegion            `json:"region"`                        // only return users in this data region
	AccessPrimaryDBOnly bool                         `json:"access_primary_db_only"`        // whether to read from primary db only
	Debug               bool                         `json:"debug,omitempty"`               // whether to include debug information in the response
}

//go:generate genvalidate ExecuteAccessorRequest

// ExecuteAccessorResponse is the response body for accessing user data
type ExecuteAccessorResponse struct {
	Data  []string       `json:"data"`
	Debug map[string]any `json:"debug,omitempty"`
	// TODO: Truncated will need to be added to our python SDK if we keep it
	Truncated bool `json:"truncated" description:"Will be true if an incomplete set of results could be returned for the query"`
	pagination.ResponseFields
}

// ExecuteAccessor accesses a column via an accessor for the associated tenant
func (c *Client) ExecuteAccessor(ctx context.Context, accessorID uuid.UUID, clientContext policy.ClientContext, selectorValues userstore.UserSelectorValues, opts ...Option) (*ExecuteAccessorResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	query := pager.Query()

	req := ExecuteAccessorRequest{
		AccessorID:          accessorID,
		Context:             clientContext,
		SelectorValues:      selectorValues,
		Region:              options.dataRegion,
		Debug:               options.debug,
		AccessPrimaryDBOnly: options.accessPrimaryDBOnly,
	}

	var res ExecuteAccessorResponse
	if err := c.client.Post(ctx, fmt.Sprintf("%s?%s", paths.ExecuteAccessorPath, query.Encode()), req, &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// MutatorColumnDefaultValue is a special value that can be used to set a column to its default value
const MutatorColumnDefaultValue = "UCDEF-7f55f479-3822-4976-a8a9-b789d5c6f152"

// MutatorColumnCurrentValue is a special value that can be used to set a column to its current value
const MutatorColumnCurrentValue = "UCCUR-7f55f479-3822-4976-a8a9-b789d5c6f152"

// ValueAndPurposes is a tuple for specifying the value and the purpose to store for a user column
type ValueAndPurposes struct {
	Value            any                    `json:"value"`
	ValueAdditions   any                    `json:"value_additions"`
	ValueDeletions   any                    `json:"value_deletions"`
	PurposeAdditions []userstore.ResourceID `json:"purpose_additions"`
	PurposeDeletions []userstore.ResourceID `json:"purpose_deletions"`
}

// ExecuteMutatorRequest is the request body for modifying data in the userstore
type ExecuteMutatorRequest struct {
	MutatorID      uuid.UUID                    `json:"mutator_id" validate:"notnil"` // the mutator that specifies what columns to edit
	Context        policy.ClientContext         `json:"context"`                      // context that is provided to the mutator's Access Policy
	SelectorValues userstore.UserSelectorValues `json:"selector_values"`              // the values to use for the selector
	RowData        map[string]ValueAndPurposes  `json:"row_data"`                     // the values to use for the users table row
	Region         region.DataRegion            `json:"region"`                       // restrict mutations to users in this data region
}

//go:generate genvalidate ExecuteMutatorRequest

// ExecuteMutatorResponse is the response body for modifying data in the userstore
type ExecuteMutatorResponse struct {
	UserIDs []uuid.UUID `json:"user_ids"`
}

// ExecuteMutator modifies columns in userstore via a mutator for the associated tenant
func (c *Client) ExecuteMutator(ctx context.Context, mutatorID uuid.UUID, clientContext policy.ClientContext, selectorValues userstore.UserSelectorValues, rowData map[string]ValueAndPurposes, opts ...Option) (*ExecuteMutatorResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := ExecuteMutatorRequest{
		MutatorID:      mutatorID,
		Context:        clientContext,
		SelectorValues: selectorValues,
		RowData:        rowData,
		Region:         options.dataRegion,
	}

	var resp ExecuteMutatorResponse
	if err := c.client.Post(ctx, paths.ExecuteMutatorPath, req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// CreatePurposeRequest is the request body for creating a new purpose
type CreatePurposeRequest struct {
	Purpose userstore.Purpose `json:"purpose"`
}

//go:generate genvalidate CreatePurposeRequest

// CreatePurpose creates a new purpose for the associated tenant
func (c *Client) CreatePurpose(ctx context.Context, purpose userstore.Purpose, opts ...Option) (*userstore.Purpose, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreatePurposeRequest{
		Purpose: purpose,
	}

	var resp userstore.Purpose
	if options.ifNotExists {
		exists, existingID, err := c.client.CreateIfNotExists(ctx, paths.CreatePurposePath, req, &resp)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if exists {
			resp = req.Purpose
			resp.ID = existingID
		}
	} else {
		if err := c.client.Post(ctx, paths.CreatePurposePath, req, &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &resp, nil
}

// GetPurpose gets a purpose by ID
func (c *Client) GetPurpose(ctx context.Context, purposeID uuid.UUID) (*userstore.Purpose, error) {
	var resp userstore.Purpose
	if err := c.client.Get(ctx, paths.GetPurposePath(purposeID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListPurposesResponse is the paginated response struct for listing purposes
type ListPurposesResponse struct {
	Data []userstore.Purpose `json:"data"`
	pagination.ResponseFields
}

// ListPurposes lists all purposes for the associated tenant
func (c *Client) ListPurposes(ctx context.Context, opts ...Option) (*ListPurposesResponse, error) {

	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	query := pager.Query()

	var res ListPurposesResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.ListPurposesPath, query.Encode()), &res); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &res, nil
}

// UpdatePurposeRequest is the request body for updating a purpose
type UpdatePurposeRequest struct {
	Purpose userstore.Purpose `json:"purpose"`
}

//go:generate genvalidate UpdatePurposeRequest

// UpdatePurpose updates a purpose for the associated tenant
func (c *Client) UpdatePurpose(ctx context.Context, purpose userstore.Purpose) (*userstore.Purpose, error) {
	req := UpdatePurposeRequest{
		Purpose: purpose,
	}

	var resp userstore.Purpose
	if err := c.client.Put(ctx, paths.UpdatePurposePath(purpose.ID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// DeletePurpose deletes a purpose by ID
func (c *Client) DeletePurpose(ctx context.Context, purposeID uuid.UUID) error {
	if err := c.client.Delete(ctx, paths.DeletePurposePath(purposeID), nil); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// Userstore User methods

// CreateUserWithMutatorRequest is the request body for creating a new user with a mutator
type CreateUserWithMutatorRequest struct {
	ID             uuid.UUID                   `json:"id"`              // ID of the user to create (optional)
	MutatorID      uuid.UUID                   `json:"mutator_id"`      // ID of the mutator that specifies what columns to edit
	Context        policy.ClientContext        `json:"context"`         // context that is provided to the mutator's Access Policy
	RowData        map[string]ValueAndPurposes `json:"row_data"`        // the values to use for the users table row
	OrganizationID uuid.UUID                   `json:"organization_id"` // the organization ID to use for the user
	Region         region.DataRegion           `json:"region"`          // the region to use for the user
}

//go:generate genvalidate CreateUserWithMutatorRequest

// CreateUserWithMutator creates a new user and initializes the user's data with the given mutator
func (c *Client) CreateUserWithMutator(ctx context.Context, mutatorID uuid.UUID, clientContext policy.ClientContext, rowData map[string]ValueAndPurposes, opts ...Option) (uuid.UUID, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateUserWithMutatorRequest{
		ID:             options.userID,
		MutatorID:      mutatorID,
		Context:        clientContext,
		RowData:        rowData,
		OrganizationID: options.organizationID,
		Region:         options.dataRegion,
	}

	var res uuid.UUID
	if err := c.client.Post(ctx, paths.CreateUserWithMutatorPath, req, &res); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	return res, nil
}

// GetConsentedPurposesForUserRequest is the request body for getting the purposes that are consented for a user
type GetConsentedPurposesForUserRequest struct {
	UserID  uuid.UUID              `json:"user_id"`
	Columns []userstore.ResourceID `json:"columns"`
}

//go:generate genvalidate GetConsentedPurposesForUserRequest

// ColumnConsentedPurposes is a tuple for specifying the column and the purposes that are consented for that column
type ColumnConsentedPurposes struct {
	Column            userstore.ResourceID   `json:"column"`
	ConsentedPurposes []userstore.ResourceID `json:"consented_purposes"`
}

// GetConsentedPurposesForUserResponse is the response body for getting the purposes that are consented for a user
type GetConsentedPurposesForUserResponse struct {
	Data []ColumnConsentedPurposes `json:"data"`
}

// GetConsentedPurposesForUser gets the purposes that are consented for a user
func (c *Client) GetConsentedPurposesForUser(ctx context.Context, userID uuid.UUID, columns []userstore.ResourceID) (GetConsentedPurposesForUserResponse, error) {
	req := GetConsentedPurposesForUserRequest{
		UserID:  userID,
		Columns: columns,
	}

	var resp GetConsentedPurposesForUserResponse
	if err := c.client.Post(ctx, paths.GetConsentedPurposesForUserPath, req, &resp); err != nil {
		return resp, ucerr.Wrap(err)
	}

	return resp, nil
}

// DownloadGolangSDK downloads the generated Golang SDK for this tenant's userstore configuration
func (c *Client) DownloadGolangSDK(ctx context.Context) (string, error) {
	path := paths.DownloadGolangSDKPath

	var sdk string
	rawBodyDecoder := func(ctx context.Context, body io.ReadCloser) error {
		buf := &strings.Builder{}
		if _, err := io.Copy(buf, body); err != nil {
			return ucerr.Wrap(err)
		}
		sdk = buf.String()
		return nil
	}

	if err := c.client.Get(ctx, path, nil, jsonclient.CustomDecoder(rawBodyDecoder)); err != nil {
		return "", ucerr.Wrap(err)
	}

	return sdk, nil
}

// DownloadPythonSDK downloads the generated Python SDK for this tenant's userstore configuration
func (c *Client) DownloadPythonSDK(ctx context.Context) (string, error) {
	path := paths.DownloadPythonSDKPath

	var sdk string
	rawBodyDecoder := func(ctx context.Context, body io.ReadCloser) error {
		buf := &strings.Builder{}
		if _, err := io.Copy(buf, body); err != nil {
			return ucerr.Wrap(err)
		}
		sdk = buf.String()
		return nil
	}

	if err := c.client.Get(ctx, path, nil, jsonclient.CustomDecoder(rawBodyDecoder)); err != nil {
		return "", ucerr.Wrap(err)
	}

	return sdk, nil
}

// GetExternalOIDCIssuers returns the list of external OIDC issuers for JWT tokens for the tenant
func (c *Client) GetExternalOIDCIssuers(ctx context.Context) ([]string, error) {
	var resp []string
	if err := c.client.Get(ctx, paths.ExternalOIDCIssuersPath, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return resp, nil
}

// UpdateExternalOIDCIssuers updates the list of external OIDC issuers for JWT tokens for the tenant
func (c *Client) UpdateExternalOIDCIssuers(ctx context.Context, issuers []string) error {
	if err := c.client.Put(ctx, paths.ExternalOIDCIssuersPath, issuers, nil); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// CreateUserSearchIndexRequest is the request body for creating a new user search index
type CreateUserSearchIndexRequest struct {
	Index search.UserSearchIndex `json:"index"`
}

// CreateUserSearchIndex creates a new user search index for the associated tenant
func (c *Client) CreateUserSearchIndex(ctx context.Context, index search.UserSearchIndex, opts ...Option) (*search.UserSearchIndex, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	req := CreateUserSearchIndexRequest{
		Index: index,
	}

	var resp search.UserSearchIndex
	if options.ifNotExists {
		exists, existingID, err := c.client.CreateIfNotExists(ctx, paths.SearchIndexPath, req, &resp)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if exists {
			resp = req.Index
			resp.ID = existingID
		}
	} else {
		if err := c.client.Post(ctx, paths.SearchIndexPath, req, &resp); err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &resp, nil
}

// DeleteUserSearchIndex deletes the user search index specified by the index ID for the associated tenant
func (c *Client) DeleteUserSearchIndex(ctx context.Context, indexID uuid.UUID) error {
	return ucerr.Wrap(c.client.Delete(ctx, paths.SearchIndexPathForID(indexID), nil))
}

// GetUserSearchIndex returns the user search index specified by the index ID for the associated tenant
func (c *Client) GetUserSearchIndex(ctx context.Context, indexID uuid.UUID) (*search.UserSearchIndex, error) {
	var resp search.UserSearchIndex
	if err := c.client.Get(ctx, paths.SearchIndexPathForID(indexID), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// ListUserSearchIndicesResponse is the paginated response struct for listing user search indices
type ListUserSearchIndicesResponse struct {
	Data []search.UserSearchIndex `json:"data"`
	pagination.ResponseFields
}

// ListUserSearchIndices lists all user search indices for the associated tenant
func (c *Client) ListUserSearchIndices(ctx context.Context, opts ...Option) (*ListUserSearchIndicesResponse, error) {
	options := c.options
	for _, opt := range opts {
		opt.apply(&options)
	}

	pager, err := pagination.ApplyOptions(options.paginationOptions...)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	query := pager.Query()

	var resp ListUserSearchIndicesResponse
	if err := c.client.Get(ctx, fmt.Sprintf("%s?%s", paths.SearchIndexPath, query.Encode()), &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// UpdateUserSearchIndexRequest is the request body for updating a user search index
type UpdateUserSearchIndexRequest struct {
	Index search.UserSearchIndex `json:"index"`
}

// UpdateUserSearchIndex updates the user search index specified by the index ID with the specified data for the associated tenant
func (c *Client) UpdateUserSearchIndex(ctx context.Context, indexID uuid.UUID, updatedIndex search.UserSearchIndex) (*search.UserSearchIndex, error) {
	req := UpdateUserSearchIndexRequest{
		Index: updatedIndex,
	}

	var resp search.UserSearchIndex
	if err := c.client.Put(ctx, paths.SearchIndexPathForID(indexID), req, &resp); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return &resp, nil
}

// RemoveAccessorUserSearchIndexRequest is the request body for removing an accessor user search index association
type RemoveAccessorUserSearchIndexRequest struct {
	AccessorID uuid.UUID `json:"accessor_id"`
}

// RemoveAccessorUserSearchIndex disassociates the specified accessor from any user search index
func (c *Client) RemoveAccessorUserSearchIndex(ctx context.Context, accessorID uuid.UUID) error {
	req := RemoveAccessorUserSearchIndexRequest{
		AccessorID: accessorID,
	}

	return ucerr.Wrap(c.client.Post(ctx, paths.RemoveAccessorSearchIndexPath, req, nil))
}

// SetAccessorUserSearchIndexRequest is the request body for updating an accessor user search index association
type SetAccessorUserSearchIndexRequest struct {
	AccessorID uuid.UUID        `json:"accessor_id"`
	IndexID    uuid.UUID        `json:"index_id"`
	QueryType  search.QueryType `json:"query_type"`
}

// SetAccessorUserSearchIndex associates the specified accessor with the specified user search index and query type
func (c *Client) SetAccessorUserSearchIndex(ctx context.Context, accessorID uuid.UUID, indexID uuid.UUID, queryType search.QueryType) error {
	req := SetAccessorUserSearchIndexRequest{
		AccessorID: accessorID,
		IndexID:    indexID,
		QueryType:  queryType,
	}

	return ucerr.Wrap(c.client.Post(ctx, paths.SetAccessorSearchIndexPath, req, nil))
}
