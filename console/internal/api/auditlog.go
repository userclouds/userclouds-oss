package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/auditlog"
	"userclouds.com/plex/manager"
)

// augment audit log entries with actor names for console
type consoleAuditLogEntry struct {
	auditlog.Entry
	ActorName string `json:"actor_name"`
}

// further augment data access log entries with accessor details
type consoleDataAccessLogEntry struct {
	consoleAuditLogEntry
	AccessorID      uuid.UUID `json:"accessor_id"`
	AccessorName    string    `json:"accessor_name"`
	AccessorVersion int       `json:"accessor_version"`
	Columns         string    `json:"columns"`
	Purposes        string    `json:"purposes"`
	Masked          string    `json:"masked"`
	Completed       bool      `json:"completed"`
	Rows            int       `json:"rows"`
}

func (h *handler) addActorNamesToEntries(ctx context.Context, tenantID uuid.UUID, tenantIDPClient *idp.ManagementClient, entries *[]consoleAuditLogEntry) error {
	actorIDSet := set.NewUUIDSet()
	for _, entry := range *entries {
		actorID, err := uuid.FromString(entry.Actor)
		if err != nil {
			continue
		}
		actorIDSet.Insert(actorID)
	}

	if actorIDSet.Size() == 0 {
		return nil
	}

	actorIDs := actorIDSet.Items()
	actorMap := map[uuid.UUID]string{}

	// pull user profiles from the tenant's idp
	tenantProfiles, err := tenantIDPClient.GetUserBaseProfiles(ctx, actorIDs)
	if err != nil {
		return ucerr.Wrap(err)
	}
	for _, profile := range tenantProfiles {
		actorID, err := uuid.FromString(profile.ID)
		if err != nil {
			return ucerr.Wrap(err)
		}
		actorMap[actorID] = profile.Email
	}

	// pull user profiles from the console tenant
	consoleProfiles, err := h.consoleIDPClient.GetUserBaseProfiles(ctx, actorIDs)
	if err != nil {
		return ucerr.Wrap(err)
	}
	for _, profile := range consoleProfiles {
		actorID, err := uuid.FromString(profile.ID)
		if err != nil {
			return ucerr.Wrap(err)
		}
		actorMap[actorID] = profile.Email
	}

	// pull login apps from the tenant's plex config
	tenantDB, err := h.tenantCache.GetTenantDB(ctx, tenantID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	plexManager := manager.NewFromDB(tenantDB, h.cacheConfig)
	loginApps, err := plexManager.GetLoginApps(ctx, tenantID, uuid.Nil)
	if err != nil {
		return ucerr.Wrap(err)
	}
	for _, app := range loginApps {
		actorMap[app.ID] = app.Name
	}

	// add actor emails / login app names to entries
	for i, entry := range *entries {
		if actorID, err := uuid.FromString(entry.Actor); err == nil {
			if actorIdentifier, ok := actorMap[actorID]; ok {
				(*entries)[i].ActorName = actorIdentifier
			}
		}

	}

	return nil
}

const configChangesFilter = "((('type',LK,'Create%'),OR,('type',LK,'Update%'),OR,('type',LK,'Delete%')),AND,('type',NE,'CreateToken'),AND,('type',NE,'DeleteToken'))"

type listAuditLogEntriesResponse struct {
	Data []consoleAuditLogEntry `json:"data"`
	pagination.ResponseFields
}

func (h *handler) listAuditLogEntries(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// Create audit log and idp clients specifically for this tenant.
	// TODO - this pattern calls for caching otherwise we do extra roundtrips to POST /oidc/token every time the user navigates to this page
	auditLogClient, err := h.newAuditLogClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	idpMgmtClient, err := h.newIDPMgmtClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// check if we need to apply the config-changes only filter
	query := pagination.QueryParamsFromRequest(r)
	if r.URL.Query().Get("config_changes") != "" {
		filter := configChangesFilter
		if query.Filter != nil && *query.Filter != "" {
			filter = fmt.Sprintf("(%s,AND,%s)", *query.Filter, filter)
		}
		query.Filter = &filter
	}

	pager, err := pagination.NewPaginatorFromQuery(query)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	res, err := auditLogClient.ListEntries(ctx, auditlog.Pagination(pager.GetOptions()...))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var entries []consoleAuditLogEntry
	for i := range res.Data {
		entries = append(entries, consoleAuditLogEntry{Entry: res.Data[i]})
	}
	if err := h.addActorNamesToEntries(ctx, id, idpMgmtClient, &entries); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	jsonapi.Marshal(w, listAuditLogEntriesResponse{Data: entries, ResponseFields: res.ResponseFields})
}

const dataAccessLogBaseFilter = "('type',EQ,'ExecuteAccessor')"

type listDataAccessLogEntriesResponse struct {
	Data []consoleDataAccessLogEntry `json:"data"`
	pagination.ResponseFields
}

func (h *handler) listDataAccessLogEntries(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// Create audit log and idp clients specifically for this tenant.
	// TODO - this pattern calls for caching otherwise we do extra roundtrips to POST /oidc/token every time the user navigates to this page
	auditLogClient, err := h.newAuditLogClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	idpMgmtClient, err := h.newIDPMgmtClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	idpClient := idpMgmtClient.GetClient()

	resp, err := idpClient.ListAccessors(ctx, true, idp.Pagination(pagination.Limit(pagination.MaxLimit)))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	accessorMap := map[string]userstore.Accessor{}
	for i := range resp.Data {
		accessorMap[fmt.Sprintf("%s-%d", resp.Data[i].ID, resp.Data[i].Version)] = resp.Data[i]
	}

	query := pagination.QueryParamsFromRequest(r)
	filter := dataAccessLogBaseFilter

	// apply column filter
	if r.URL.Query().Get("column_id") != "" {
		accessorFilters := []string{}
		for _, accessor := range accessorMap {
			for _, columnConfig := range accessor.Columns {
				if columnConfig.Column.ID.String() == r.URL.Query().Get("column_id") {
					accessorFilters = append(accessorFilters, fmt.Sprintf("(('payload->>ID',EQ,'%s'),AND,('payload->>Version',EQ,'%d'))", accessor.ID, accessor.Version))
					break
				}
			}
		}
		if len(accessorFilters) == 1 {
			filter = fmt.Sprintf("(%s,AND,%s)", filter, accessorFilters[0])
		} else if len(accessorFilters) > 1 {
			accessorFiltersStr := "(" + strings.Join(accessorFilters, ",OR,") + ")"
			filter = fmt.Sprintf("(%s,AND,%s)", filter, accessorFiltersStr)
		} else {
			filter = fmt.Sprintf("(%s,AND,%s)", filter, "('type',NE,'ExecuteAccessor')") // no matches, so create contradictory filter
		}
	}

	// apply accessor filter
	if r.URL.Query().Get("accessor_id") != "" {
		filter = fmt.Sprintf("(%s,AND,('payload->>ID',EQ,'%s'))", filter, r.URL.Query().Get("accessor_id"))
	}

	// apply actor filter
	if r.URL.Query().Get("actor_id") != "" {
		filter = fmt.Sprintf("(%s,AND,('actor_id',EQ,'%s'))", filter, r.URL.Query().Get("actor_id"))
	}

	// apply selector filter
	if r.URL.Query().Get("selector_value") != "" {
		filter = fmt.Sprintf("(%s,AND,('payload->SelectorValues',HAS,'%s'))", filter, r.URL.Query().Get("selector_value"))
	}
	query.Filter = &filter

	pager, err := pagination.NewPaginatorFromQuery(query)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
		return
	}

	res, err := auditLogClient.ListEntries(ctx, auditlog.Pagination(pager.GetOptions()...))
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var entries []consoleAuditLogEntry
	for i := range res.Data {
		entries = append(entries, consoleAuditLogEntry{Entry: res.Data[i]})
	}
	if err := h.addActorNamesToEntries(ctx, id, idpMgmtClient, &entries); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	var dataAccessLogEntries []consoleDataAccessLogEntry
	for _, entry := range entries {
		dataAccessLogEntry := consoleDataAccessLogEntry{
			consoleAuditLogEntry: entry,
			AccessorName:         "<deleted>",
			Columns:              "<unknown>",
			Purposes:             "<unknown>",
		}

		if accessorIDString, ok := entry.Payload["ID"].(string); ok {
			if accessorID, err := uuid.FromString(accessorIDString); err == nil {
				if version, ok := entry.Payload["Version"].(float64); ok {
					accessorKey := fmt.Sprintf("%s-%d", accessorID, int(version))
					accessor, ok := accessorMap[accessorKey]
					if ok {
						h.addAccessorDetailsToEntry(&dataAccessLogEntry, &accessor)
					}
				}
			}
		}

		if rows, ok := entry.Payload["RowsReturned"].(float64); ok {
			dataAccessLogEntry.Rows = int(rows)
		}
		if completed, ok := entry.Payload["Succeeded"]; ok {
			dataAccessLogEntry.Completed = completed == true
		}

		dataAccessLogEntries = append(dataAccessLogEntries, dataAccessLogEntry)
	}

	jsonapi.Marshal(w, listDataAccessLogEntriesResponse{Data: dataAccessLogEntries, ResponseFields: res.ResponseFields})
}

func (h *handler) getDataAccessLogEntry(w http.ResponseWriter, r *http.Request, id uuid.UUID, entryID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, id)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	// Create audit log and idp clients specifically for this tenant.
	// TODO - this pattern calls for caching otherwise we do extra roundtrips to POST /oidc/token every time the user navigates to this page
	auditLogClient, err := h.newAuditLogClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	entry, err := auditLogClient.GetEntry(ctx, entryID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	consoleEntry := consoleAuditLogEntry{Entry: *entry}
	consoleEntries := []consoleAuditLogEntry{consoleEntry}
	idpMgmtClient, err := h.newIDPMgmtClient(ctx, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	idpClient := idpMgmtClient.GetClient()
	if err := h.addActorNamesToEntries(ctx, id, idpMgmtClient, &consoleEntries); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	dataAccessLogEntry := consoleDataAccessLogEntry{
		consoleAuditLogEntry: consoleEntries[0],
	}

	if accessorIDString, ok := entry.Payload["ID"].(string); ok {
		if accessorID, err := uuid.FromString(accessorIDString); err == nil {
			var accessor *userstore.Accessor
			if version, ok := entry.Payload["Version"].(float64); ok {
				accessor, err = idpClient.GetAccessorByVersion(ctx, accessorID, int(version))
				if err != nil {
					jsonapi.MarshalError(ctx, w, err)
					return
				}
			} else {
				accessor, err = idpClient.GetAccessor(ctx, accessorID)
				if err != nil {
					jsonapi.MarshalError(ctx, w, err)
					return
				}
			}
			h.addAccessorDetailsToEntry(&dataAccessLogEntry, accessor)
		}
	}

	if rows, ok := entry.Payload["RowsReturned"].(float64); ok {
		dataAccessLogEntry.Rows = int(rows)
	}
	if completed, ok := entry.Payload["Succeeded"]; ok {
		dataAccessLogEntry.Completed = completed == true
	}

	jsonapi.Marshal(w, dataAccessLogEntry)
}

func (h *handler) addAccessorDetailsToEntry(dataAccessLogEntry *consoleDataAccessLogEntry, accessor *userstore.Accessor) {
	dataAccessLogEntry.AccessorID = accessor.ID
	dataAccessLogEntry.AccessorName = accessor.Name
	dataAccessLogEntry.AccessorVersion = accessor.Version
	columns := []string{}
	for _, columnConfig := range accessor.Columns {
		columns = append(columns, columnConfig.Column.Name)
	}
	dataAccessLogEntry.Columns = strings.Join(columns, ", ")
	purposes := []string{}
	for _, purpose := range accessor.Purposes {
		purposes = append(purposes, purpose.Name)
	}
	dataAccessLogEntry.Purposes = strings.Join(purposes, ", ")
}
