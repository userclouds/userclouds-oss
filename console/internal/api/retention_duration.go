package api

import (
	"net/http"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
)

// GetColumnRetentionDurationsRequest is a console request for retrieving all column purpose
// retention durations for a column
type GetColumnRetentionDurationsRequest struct {
	ColumnID     uuid.UUID                    `json:"column_id"`
	DurationType userstore.DataLifeCycleState `json:"duration_type"`
}

// PurposeRetentionDuration represents a single purpose retention duration
type PurposeRetentionDuration struct {
	ID              uuid.UUID             `json:"id"`
	Version         int                   `json:"version"`
	PurposeID       uuid.UUID             `json:"purpose_id"`
	PurposeName     string                `json:"purpose_name"`
	Duration        idp.RetentionDuration `json:"duration"`
	UseDefault      bool                  `json:"use_default"`
	DefaultDuration idp.RetentionDuration `json:"default_duration"`
}

// ColumnRetentionDurationsResponse represents a set of purpose retention durations for a column
type ColumnRetentionDurationsResponse struct {
	ColumnID                  uuid.UUID                    `json:"column_id"`
	DurationType              userstore.DataLifeCycleState `json:"duration_type"`
	MaxDuration               idp.RetentionDuration        `json:"max_duration"`
	SupportedDurationUnits    []idp.DurationUnit           `json:"supported_duration_units"`
	PurposeRetentionDurations []PurposeRetentionDuration   `json:"purpose_retention_durations"`
}

func (crdr *ColumnRetentionDurationsResponse) setDurationType(durationType userstore.DataLifeCycleState) {
	switch durationType {
	case userstore.DataLifeCycleStateLive:
		crdr.SupportedDurationUnits =
			[]idp.DurationUnit{
				idp.DurationUnitIndefinite,
				idp.DurationUnitYear,
				idp.DurationUnitMonth,
				idp.DurationUnitWeek,
				idp.DurationUnitDay,
				idp.DurationUnitHour,
			}
	case userstore.DataLifeCycleStateSoftDeleted:
		crdr.SupportedDurationUnits =
			[]idp.DurationUnit{
				idp.DurationUnitYear,
				idp.DurationUnitMonth,
				idp.DurationUnitWeek,
				idp.DurationUnitDay,
				idp.DurationUnitHour,
			}
	default:
		// in practice this will never happen
		return
	}
	crdr.DurationType = durationType
}

func (crdr *ColumnRetentionDurationsResponse) fromIDP(
	columnID uuid.UUID,
	durationType userstore.DataLifeCycleState,
	icrdr idp.ColumnRetentionDurationsResponse,
) {
	crdr.ColumnID = columnID
	crdr.setDurationType(durationType)
	crdr.MaxDuration = icrdr.MaxDuration

	crdr.PurposeRetentionDurations = []PurposeRetentionDuration{}
	for _, icrd := range icrdr.RetentionDurations {
		crdr.PurposeRetentionDurations = append(
			crdr.PurposeRetentionDurations,
			PurposeRetentionDuration{
				ID:              icrd.ID,
				Version:         icrd.Version,
				PurposeID:       icrd.PurposeID,
				PurposeName:     *icrd.PurposeName,
				Duration:        icrd.Duration,
				UseDefault:      icrd.UseDefault,
				DefaultDuration: *icrd.DefaultDuration,
			},
		)
	}
}

func (h *handler) getUserStoreColumnRetentionDurations(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req GetColumnRetentionDurationsRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	idpResp, err := idpClient.GetColumnRetentionDurationsForColumn(ctx, req.DurationType, req.ColumnID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var resp ColumnRetentionDurationsResponse
	resp.fromIDP(req.ColumnID, req.DurationType, *idpResp)
	jsonapi.Marshal(w, resp)
}

// UpdateColumnRetentionDurationsRequest is a console request for updating column purpose
// retention durations for a column
type UpdateColumnRetentionDurationsRequest struct {
	ColumnID                  uuid.UUID                    `json:"column_id"`
	DurationType              userstore.DataLifeCycleState `json:"duration_type"`
	PurposeRetentionDurations []PurposeRetentionDuration   `json:"purpose_retention_durations"`
}

func (ucrdr UpdateColumnRetentionDurationsRequest) toIDP() idp.UpdateColumnRetentionDurationsRequest {
	req := idp.UpdateColumnRetentionDurationsRequest{
		RetentionDurations: []idp.ColumnRetentionDuration{},
	}

	for _, prd := range ucrdr.PurposeRetentionDurations {
		req.RetentionDurations = append(
			req.RetentionDurations,
			idp.ColumnRetentionDuration{
				ID:              prd.ID,
				Version:         prd.Version,
				ColumnID:        ucrdr.ColumnID,
				PurposeID:       prd.PurposeID,
				DurationType:    ucrdr.DurationType,
				Duration:        prd.Duration,
				UseDefault:      prd.UseDefault,
				DefaultDuration: &prd.DefaultDuration,
				PurposeName:     &prd.PurposeName,
			},
		)
	}

	return req
}

func (h *handler) updateUserStoreColumnRetentionDurations(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	tenant, err := h.storage.GetTenant(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalSQLError(ctx, w, err)
		return
	}

	isAdmin, err := h.ensureEmployeeAccessToTenant(r, tenant)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if !isAdmin {
		jsonapi.MarshalError(
			ctx,
			w,
			ucerr.Friendlyf(nil, "only tenant admins can update column retention durations"),
			jsonapi.Code(http.StatusForbidden),
		)
		return
	}

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var req UpdateColumnRetentionDurationsRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if _, err = idpClient.UpdateColumnRetentionDurationsForColumn(ctx, req.DurationType, req.ColumnID, req.toIDP()); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	idpResp, err := idpClient.GetColumnRetentionDurationsForColumn(ctx, req.DurationType, req.ColumnID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var resp ColumnRetentionDurationsResponse
	resp.fromIDP(req.ColumnID, req.DurationType, *idpResp)
	jsonapi.Marshal(w, resp)
}
