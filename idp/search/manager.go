package search

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/search"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/worker"
)

// Manager manages the lifecycle of the search indices configured for a tenant
type Manager struct {
	ctx          context.Context
	idpClient    *idp.Client
	tenantDB     *ucdb.DB
	tenantID     uuid.UUID
	workerClient workerclient.Client
}

// NewManager creates a new search manager
func NewManager(
	ctx context.Context,
	tenantID uuid.UUID,
	tenantDB *ucdb.DB,
	idpClient *idp.Client,
	workerClient workerclient.Client,
) (*Manager, error) {
	return &Manager{
		ctx:          ctx,
		idpClient:    idpClient,
		tenantDB:     tenantDB,
		tenantID:     tenantID,
		workerClient: workerClient,
	}, nil
}

// CreateIndex creates the index for the specified parameters
func (m *Manager) CreateIndex(
	name string,
	description string,
	indexType search.IndexType,
	indexSettings search.IndexSettings,
	columnIDs ...uuid.UUID,
) (uuid.UUID, error) {
	index := search.UserSearchIndex{
		Name:               name,
		Description:        description,
		DataLifeCycleState: userstore.DataLifeCycleStateLive,
		Type:               indexType,
		Settings:           indexSettings,
	}

	for _, columnID := range columnIDs {
		index.Columns = append(
			index.Columns,
			userstore.ResourceID{ID: columnID},
		)
	}

	createdIndex, err := m.idpClient.CreateUserSearchIndex(m.ctx, index)
	if err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	return createdIndex.ID, nil
}

// DeleteIndex deletes the target index
func (m *Manager) DeleteIndex(indexID uuid.UUID) error {
	if err := m.idpClient.DeleteUserSearchIndex(m.ctx, indexID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// DisableIndex marks the target index as disabled
func (m *Manager) DisableIndex(indexID uuid.UUID) error {
	index, err := m.GetIndex(indexID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	index.Disable()

	if _, err := m.idpClient.UpdateUserSearchIndex(m.ctx, index.ID, *index); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// EnableIndex sends a message to the worker to enable the target index
func (m *Manager) EnableIndex(indexID uuid.UUID) error {
	if _, err := m.GetIndex(indexID); err != nil {
		return ucerr.Wrap(err)
	}

	msg := worker.ProvisionTenantOpenSearchIndexMessage(m.tenantID, indexID)
	if err := m.workerClient.Send(m.ctx, msg); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// GetIndex returns the target index
func (m Manager) GetIndex(indexID uuid.UUID) (*search.UserSearchIndex, error) {
	index, err := m.idpClient.GetUserSearchIndex(m.ctx, indexID)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return index, nil
}

// ListIndices returns all indices
func (m Manager) ListIndices() ([]search.UserSearchIndex, error) {
	resp, err := m.idpClient.ListUserSearchIndices(m.ctx, idp.Pagination(pagination.Limit(pagination.MaxLimit)))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if resp.HasNext {
		uclog.Warningf(m.ctx, "there are more than %d configured user search indices", pagination.MaxLimit)
	}

	return resp.Data, nil
}

// ContinueIndexBootstrap continues the index bootstrap where it last left
// off, which can be useful if it stalls.
func (m *Manager) ContinueIndexBootstrap(indexID uuid.UUID, r region.DataRegion, batchSize int) error {
	index, err := m.GetIndex(indexID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	if index.Enabled.IsZero() {
		return ucerr.Friendlyf(nil, "index '%v' is not enabled", indexID)
	}

	if !index.Bootstrapped.IsZero() {
		return ucerr.Friendlyf(nil, "index '%v' is already bootstrapped", indexID)
	}

	msg := worker.BootstrapTenantOpenSearchIndexMessage(m.tenantID, indexID, uuid.Nil, r, batchSize)
	if err := m.workerClient.Send(m.ctx, msg); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// MarkIndexSearchable sets the searchable timestamp for the specified
// index to the current time. The index must be bootstrapped.
func (m *Manager) MarkIndexSearchable(indexID uuid.UUID) error {
	index, err := m.GetIndex(indexID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	index.Searchable = time.Now().UTC()

	if _, err := m.idpClient.UpdateUserSearchIndex(m.ctx, index.ID, *index); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// MarkIndexUnsearchable clears the searchable timestamp for the specified
// index. The index cannot be in use by any accessors.
func (m *Manager) MarkIndexUnsearchable(indexID uuid.UUID) error {
	index, err := m.GetIndex(indexID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	index.Searchable = time.Time{}

	if _, err := m.idpClient.UpdateUserSearchIndex(m.ctx, index.ID, *index); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// RemoveAccessorIndex disassociates the specified accessor from its search index.
func (m *Manager) RemoveAccessorIndex(accessorID uuid.UUID) error {
	if err := m.idpClient.RemoveAccessorUserSearchIndex(m.ctx, accessorID); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// SetAccessorIndex associates the specified accessor with the specified search
// index. The index must be searchable and compatible with the search column
// associated with the index.
func (m *Manager) SetAccessorIndex(
	accessorID uuid.UUID,
	indexID uuid.UUID,
	queryType search.QueryType,
) error {
	if err := m.idpClient.SetAccessorUserSearchIndex(
		m.ctx,
		accessorID,
		indexID,
		queryType,
	); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// SetIndexColumnIDs updates the columns supported by the specified index. This
// can only be performed if the index is disabled.
func (m *Manager) SetIndexColumnIDs(
	indexID uuid.UUID,
	columnIDs ...uuid.UUID,
) error {
	index, err := m.GetIndex(indexID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	index.Columns = []userstore.ResourceID{}
	for _, columnID := range columnIDs {
		index.Columns = append(
			index.Columns,
			userstore.ResourceID{ID: columnID},
		)
	}

	if _, err := m.idpClient.UpdateUserSearchIndex(m.ctx, index.ID, *index); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// SetIndexDescription updates the description for the specified index.
func (m *Manager) SetIndexDescription(
	indexID uuid.UUID,
	description string,
) error {
	index, err := m.GetIndex(indexID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	index.Description = description

	if _, err := m.idpClient.UpdateUserSearchIndex(m.ctx, index.ID, *index); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// SetIndexName updates the name of the specified index. The name must
// be unique and cannot be changed while the index is enabled.
func (m *Manager) SetIndexName(
	indexID uuid.UUID,
	name string,
) error {
	index, err := m.GetIndex(indexID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	index.Name = name

	if _, err := m.idpClient.UpdateUserSearchIndex(m.ctx, index.ID, *index); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// SetIndexType updates the type and associated settings of the specified
// index. The type cannot be updated while the index is enabled.
func (m *Manager) SetIndexType(
	indexID uuid.UUID,
	indexType search.IndexType,
	indexSettings search.IndexSettings,
) error {
	index, err := m.GetIndex(indexID)
	if err != nil {
		return ucerr.Wrap(err)
	}

	index.Type = indexType
	index.Settings = indexSettings

	if _, err := m.idpClient.UpdateUserSearchIndex(m.ctx, index.ID, *index); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
