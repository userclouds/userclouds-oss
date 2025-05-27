package userstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/internal/userstore/validation"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/auditlog"
)

func verifyColumnExists(ctx context.Context, s *storage.Storage, columnID uuid.UUID) (*storage.Column, int, error) {
	c, err := s.GetColumn(ctx, columnID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusNotFound, ucerr.Friendlyf(err, "column '%v' does not exist", columnID)
		}
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}
	return c, http.StatusOK, nil
}

func verifyPurposeExists(ctx context.Context, s *storage.Storage, purposeID uuid.UUID) (*storage.Purpose, int, error) {
	p, err := s.GetPurpose(ctx, purposeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, http.StatusNotFound, ucerr.Friendlyf(err, "purpose '%v' does not exist", purposeID)
		}
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}
	return p, http.StatusOK, nil
}

func deleteRetentionDurationForColumn(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	columnID uuid.UUID,
	durationID uuid.UUID,
) (int, error) {
	_, code, err := getRetentionDurationForColumn(ctx, durationType, columnID, durationID)
	if err != nil {
		return code, ucerr.Wrap(err)
	}

	s := storage.MustCreateStorage(ctx)

	if err := s.DeleteColumnValueRetentionDuration(ctx, durationID); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil
}

func deleteRetentionDurationForPurpose(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	purposeID uuid.UUID,
	durationID uuid.UUID,
) (int, error) {
	_, code, err := getRetentionDurationForPurpose(ctx, durationType, purposeID, durationID, true)
	if err != nil {
		return code, ucerr.Wrap(err)
	}

	s := storage.MustCreateStorage(ctx)

	if err := s.DeleteColumnValueRetentionDuration(ctx, durationID); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil
}

func deleteRetentionDurationForTenant(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	durationID uuid.UUID,
) (int, error) {
	_, code, err := getRetentionDurationForTenant(ctx, durationType, durationID, true)
	if err != nil {
		return code, ucerr.Wrap(err)
	}

	s := storage.MustCreateStorage(ctx)

	if err := s.DeleteColumnValueRetentionDuration(ctx, durationID); err != nil {
		return http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return http.StatusNoContent, nil
}

func getRetentionDurationForColumn(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	columnID uuid.UUID,
	durationID uuid.UUID,
) (*idp.ColumnRetentionDurationResponse, int, error) {
	if err := validation.ValidateDurationID(durationID, true); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	intermediateResp, code, err := getRetentionDurationsForColumn(ctx, durationType, columnID, durationID)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	resp := idp.ColumnRetentionDurationResponse{
		MaxDuration:       intermediateResp.MaxDuration,
		RetentionDuration: intermediateResp.RetentionDurations[0],
	}

	return &resp, http.StatusOK, nil
}

func getRetentionDurationsForColumn(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	columnID uuid.UUID,
	durationID uuid.UUID,
) (*idp.ColumnRetentionDurationsResponse, int, error) {
	s := storage.MustCreateStorage(ctx)

	if _, code, err := verifyColumnExists(ctx, s, columnID); err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	purposes, err := s.ListPurposesNonPaginated(ctx)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	retentionDurationsByPurposeID := map[uuid.UUID]idp.ColumnRetentionDuration{}

	defaultDuration := storage.GetDefaultRetentionDuration(durationType)
	for _, purpose := range purposes {
		retentionDurationsByPurposeID[purpose.ID] = idp.ColumnRetentionDuration{
			DurationType:    durationType.ToClient(),
			ColumnID:        columnID,
			PurposeID:       purpose.ID,
			Duration:        defaultDuration,
			UseDefault:      true,
			DefaultDuration: &defaultDuration,
			PurposeName:     &purpose.Name,
		}
	}

	pager, err := storage.NewColumnValueRetentionDurationPaginatorFromOptions(
		pagination.Filter(
			fmt.Sprintf("(%s,AND,%s)",
				fmt.Sprintf("('duration_type',EQ,'%d')", durationType),
				fmt.Sprintf("(('column_id',EQ,'%v'),OR,('column_id',EQ,'%v'))", columnID, uuid.Nil),
			),
		),
		pagination.SortKey("column_id,purpose_id,id"),
	)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	for {
		retentionDurations, respFields, err := s.ListColumnValueRetentionDurationsPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, ucerr.Wrap(err)
		}

		for _, rd := range retentionDurations {
			if rd.ColumnID.IsNil() {
				if rd.PurposeID.IsNil() {
					for purposeID, retentionDuration := range retentionDurationsByPurposeID {
						retentionDuration.Duration.Unit = rd.DurationUnit.ToClient()
						retentionDuration.Duration.Duration = rd.Duration
						retentionDuration.DefaultDuration = &retentionDuration.Duration
						retentionDurationsByPurposeID[purposeID] = retentionDuration
					}
				} else {
					purposeRetentionDuration, found := retentionDurationsByPurposeID[rd.PurposeID]
					if !found {
						uclog.Errorf(ctx, "encountered unrecognized purpose %v", rd.PurposeID)
						continue
					}
					purposeRetentionDuration.Duration.Unit = rd.DurationUnit.ToClient()
					purposeRetentionDuration.Duration.Duration = rd.Duration
					purposeRetentionDuration.DefaultDuration = &purposeRetentionDuration.Duration
					retentionDurationsByPurposeID[rd.PurposeID] = purposeRetentionDuration
				}
			} else {
				purposeRetentionDuration, found := retentionDurationsByPurposeID[rd.PurposeID]
				if !found {
					uclog.Errorf(ctx, "encountered unrecognized purpose %v", rd.PurposeID)
					continue
				}

				purposeRetentionDuration.ID = rd.ID
				purposeRetentionDuration.Version = rd.Version
				purposeRetentionDuration.Duration.Unit = rd.DurationUnit.ToClient()
				purposeRetentionDuration.Duration.Duration = rd.Duration
				purposeRetentionDuration.UseDefault = false
				retentionDurationsByPurposeID[purposeRetentionDuration.PurposeID] = purposeRetentionDuration
			}
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	retentionDurations := []idp.ColumnRetentionDuration{}
	for _, rd := range retentionDurationsByPurposeID {
		if durationID != uuid.Nil && durationID != rd.ID {
			continue
		}

		retentionDurations = append(retentionDurations, rd)
	}

	if durationID != uuid.Nil && len(retentionDurations) != 1 {
		return nil, http.StatusNotFound, ucerr.Friendlyf(nil, "retention duration '%v' does not exist for column '%v'", durationID, columnID)
	}

	resp := idp.ColumnRetentionDurationsResponse{
		MaxDuration:        storage.RetentionDurationMax,
		RetentionDurations: retentionDurations,
	}
	v := validation.NewRetentionDurationsResponseValidator(durationType)
	if err := v.ValidateColumnResponse(columnID, resp.MaxDuration, resp.RetentionDurations...); err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return &resp, http.StatusOK, nil
}

func getRetentionDurationForPurpose(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	durationIDRequired bool,
) (*idp.ColumnRetentionDurationResponse, int, error) {
	if err := validation.ValidateDurationID(durationID, durationIDRequired); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	s := storage.MustCreateStorage(ctx)

	purpose, code, err := verifyPurposeExists(ctx, s, purposeID)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	defaultDuration := storage.GetDefaultRetentionDuration(durationType)
	retentionDuration := idp.ColumnRetentionDuration{
		DurationType:    durationType.ToClient(),
		PurposeID:       purposeID,
		Duration:        defaultDuration,
		UseDefault:      true,
		DefaultDuration: &defaultDuration,
		PurposeName:     &purpose.Name,
	}

	pager, err := storage.NewColumnValueRetentionDurationPaginatorFromOptions(
		pagination.Filter(
			fmt.Sprintf("(%s,AND,(%s,AND,%s))",
				fmt.Sprintf("('duration_type',EQ,'%d')", durationType),
				fmt.Sprintf("('column_id',EQ,'%v')", uuid.Nil),
				fmt.Sprintf("(('purpose_id',EQ,'%v'),OR,('purpose_id',EQ,'%v'))", purposeID, uuid.Nil),
			),
		),
		pagination.SortKey("purpose_id,id"),
	)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	for {
		retentionDurations, respFields, err :=
			s.ListColumnValueRetentionDurationsPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, ucerr.Wrap(err)
		}

		for _, rd := range retentionDurations {
			if rd.PurposeID.IsNil() {
				retentionDuration.Duration.Unit = rd.DurationUnit.ToClient()
				retentionDuration.Duration.Duration = rd.Duration
				retentionDuration.DefaultDuration = &retentionDuration.Duration
			} else {
				retentionDuration.ID = rd.ID
				retentionDuration.Version = rd.Version
				retentionDuration.Duration.Unit = rd.DurationUnit.ToClient()
				retentionDuration.Duration.Duration = rd.Duration
				retentionDuration.UseDefault = false
			}
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	if durationID != uuid.Nil && durationID != retentionDuration.ID {
		return nil, http.StatusNotFound, ucerr.Friendlyf(nil, "retention duration '%v' does not exist for purpose '%v'", durationID, purposeID)
	}

	resp := idp.ColumnRetentionDurationResponse{
		MaxDuration:       storage.RetentionDurationMax,
		RetentionDuration: retentionDuration,
	}

	v := validation.NewRetentionDurationsResponseValidator(durationType)
	if err := v.ValidatePurposeResponse(purposeID, resp.MaxDuration, resp.RetentionDuration); err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return &resp, http.StatusOK, nil
}

func getRetentionDurationForTenant(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	durationID uuid.UUID,
	durationIDRequired bool,
) (*idp.ColumnRetentionDurationResponse, int, error) {
	if err := validation.ValidateDurationID(durationID, durationIDRequired); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	s := storage.MustCreateStorage(ctx)

	defaultDuration := storage.GetDefaultRetentionDuration(durationType)
	retentionDuration := idp.ColumnRetentionDuration{
		DurationType:    durationType.ToClient(),
		Duration:        defaultDuration,
		UseDefault:      true,
		DefaultDuration: &defaultDuration,
	}

	pager, err := storage.NewColumnValueRetentionDurationPaginatorFromOptions(
		pagination.Filter(
			fmt.Sprintf("(%s,AND,%s)",
				fmt.Sprintf("('duration_type',EQ,'%d')", durationType),
				fmt.Sprintf("(('column_id',EQ,'%v'),AND,('purpose_id',EQ,'%v'))", uuid.Nil, uuid.Nil),
			),
		),
	)
	if err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	for {
		retentionDurations, respFields, err :=
			s.ListColumnValueRetentionDurationsPaginated(ctx, *pager)
		if err != nil {
			return nil, http.StatusInternalServerError, ucerr.Wrap(err)
		}

		for _, rd := range retentionDurations {
			retentionDuration.ID = rd.ID
			retentionDuration.Version = rd.Version
			retentionDuration.Duration.Unit = rd.DurationUnit.ToClient()
			retentionDuration.Duration.Duration = rd.Duration
			retentionDuration.UseDefault = false
		}

		if !pager.AdvanceCursor(*respFields) {
			break
		}
	}

	if durationID != uuid.Nil && durationID != retentionDuration.ID {
		return nil, http.StatusNotFound, ucerr.Friendlyf(nil, "retention duration '%v' does not exist for tenant", durationID)
	}

	resp := idp.ColumnRetentionDurationResponse{
		MaxDuration:       storage.RetentionDurationMax,
		RetentionDuration: retentionDuration,
	}

	v := validation.NewRetentionDurationsResponseValidator(durationType)
	if err := v.ValidateTenantResponse(resp.MaxDuration, resp.RetentionDuration); err != nil {
		return nil, http.StatusInternalServerError, ucerr.Wrap(err)
	}

	return &resp, http.StatusOK, nil
}

func getUpdatedRetentionDurationsForColumn(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	columnID uuid.UUID,
	req idp.UpdateColumnRetentionDurationsRequest,
) (*idp.ColumnRetentionDurationsResponse, int, error) {
	purposeIDs := set.NewUUIDSet()
	for _, rd := range req.RetentionDurations {
		purposeIDs.Insert(rd.PurposeID)
	}

	response, code, err := getRetentionDurationsForColumn(ctx, durationType, columnID, uuid.Nil)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	updatedRetentionDurations := []idp.ColumnRetentionDuration{}
	for _, rd := range response.RetentionDurations {
		if purposeIDs.Contains(rd.PurposeID) {
			updatedRetentionDurations = append(updatedRetentionDurations, rd)
		}
	}
	response.RetentionDurations = updatedRetentionDurations
	return response, http.StatusOK, nil
}

func updateRetentionDurations(
	ctx context.Context,
	durationIDRequired bool,
	retentionDurations ...idp.ColumnRetentionDuration,
) (int, error) {
	s := storage.MustCreateStorage(ctx)

	for _, rd := range retentionDurations {
		if rd.ColumnID != uuid.Nil {
			if _, code, err := verifyColumnExists(ctx, s, rd.ColumnID); err != nil {
				return code, ucerr.Wrap(err)
			}
		}
		if rd.PurposeID != uuid.Nil {
			if _, code, err := verifyPurposeExists(ctx, s, rd.PurposeID); err != nil {
				return code, ucerr.Wrap(err)
			}
		}

		if rd.ID != uuid.Nil {
			if rd.UseDefault {
				if err := s.DeleteColumnValueRetentionDuration(ctx, rd.ID); err != nil {
					if errors.Is(err, sql.ErrNoRows) {
						return http.StatusNotFound, ucerr.Wrap(err)
					}
					return http.StatusInternalServerError, ucerr.Wrap(err)
				}
			} else {
				cvrd, err := s.GetColumnValueRetentionDuration(ctx, rd.ID)
				if err != nil {
					if durationIDRequired {
						if errors.Is(err, sql.ErrNoRows) {
							return http.StatusNotFound, ucerr.Wrap(err)
						}
						return http.StatusInternalServerError, ucerr.Wrap(err)
					} else if !errors.Is(err, sql.ErrNoRows) {
						return http.StatusInternalServerError, ucerr.Wrap(err)
					}

					// we are attempting to create a duration with the specific ID
					cvrd = &storage.ColumnValueRetentionDuration{
						VersionBaseModel: ucdb.NewVersionBaseWithID(rd.ID),
						ColumnID:         rd.ColumnID,
						PurposeID:        rd.PurposeID,
						DurationType:     column.DataLifeCycleStateFromClient(rd.DurationType),
						DurationUnit:     column.DurationUnitFromClient(rd.Duration.Unit),
						Duration:         rd.Duration.Duration,
					}
				}

				if cvrd.DurationType != column.DataLifeCycleStateFromClient(rd.DurationType) {
					return http.StatusBadRequest, ucerr.Friendlyf(nil, "DurationType '%v' does not match for ID '%v'", rd.DurationType, rd.ID)
				}

				if cvrd.ColumnID != rd.ColumnID {
					return http.StatusBadRequest, ucerr.Friendlyf(nil, "ColumnID '%v' does not match for ID '%v'", rd.ColumnID, rd.ID)
				}

				if cvrd.PurposeID != rd.PurposeID {
					return http.StatusBadRequest, ucerr.Friendlyf(nil, "PurposeID '%v' does not match for ID '%v'", rd.PurposeID, rd.ID)
				}

				cvrd.Version = rd.Version
				cvrd.DurationUnit = column.DurationUnitFromClient(rd.Duration.Unit)
				cvrd.Duration = rd.Duration.Duration

				if err := s.SaveColumnValueRetentionDuration(ctx, cvrd); err != nil {
					if ucdb.IsUniqueViolation(err) {
						return http.StatusConflict,
							ucerr.Friendlyf(
								err,
								"A retention duration setting already exists for this type (%v) and scope (ColumnID %v, PurposeID %v)",
								rd.DurationType,
								rd.ColumnID,
								rd.PurposeID,
							)
					}
					return http.StatusInternalServerError, ucerr.Wrap(err)
				}
			}
		} else if !rd.UseDefault {
			cvrd := storage.ColumnValueRetentionDuration{
				VersionBaseModel: ucdb.NewVersionBase(),
				ColumnID:         rd.ColumnID,
				PurposeID:        rd.PurposeID,
				DurationType:     column.DataLifeCycleStateFromClient(rd.DurationType),
				DurationUnit:     column.DurationUnitFromClient(rd.Duration.Unit),
				Duration:         rd.Duration.Duration,
			}
			if err := s.SaveColumnValueRetentionDuration(ctx, &cvrd); err != nil {
				if ucdb.IsUniqueViolation(err) {
					return http.StatusConflict,
						ucerr.Friendlyf(
							err,
							"A retention duration setting already exists for this type (%v) and scope (ColumnID %v, PurposeID %v)",
							rd.DurationType,
							rd.ColumnID,
							rd.PurposeID,
						)
				}
				return http.StatusInternalServerError, ucerr.Wrap(err)
			}
		}
	}

	return http.StatusOK, nil
}

func updateRetentionDurationForColumn(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	columnID uuid.UUID,
	durationID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, error) {
	v := validation.NewRetentionDurationsUpdateValidator(durationType)
	if err := v.ValidateColumnUpdate(columnID, durationID, true, req.RetentionDuration); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	code, err := updateRetentionDurations(ctx, true, req.RetentionDuration)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	response, code, err := getRetentionDurationForColumn(ctx, durationType, columnID, durationID)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	return response, http.StatusOK, nil
}

func updateRetentionDurationsForColumn(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	columnID uuid.UUID,
	req idp.UpdateColumnRetentionDurationsRequest,
) (*idp.ColumnRetentionDurationsResponse, int, error) {
	v := validation.NewRetentionDurationsUpdateValidator(durationType)
	if err := v.ValidateColumnUpdate(columnID, uuid.Nil, false, req.RetentionDurations...); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	code, err := updateRetentionDurations(ctx, false, req.RetentionDurations...)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	response, code, err := getUpdatedRetentionDurationsForColumn(ctx, durationType, columnID, req)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	return response, http.StatusOK, nil
}

func updateRetentionDurationForPurpose(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	durationIDRequired bool,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, error) {
	v := validation.NewRetentionDurationsUpdateValidator(durationType)
	if err := v.ValidatePurposeUpdate(purposeID, durationID, durationIDRequired, req.RetentionDuration); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	code, err := updateRetentionDurations(ctx, durationIDRequired, req.RetentionDuration)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	response, code, err := getRetentionDurationForPurpose(ctx, durationType, purposeID, durationID, durationIDRequired)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	return response, http.StatusOK, nil
}

func updateRetentionDurationForTenant(
	ctx context.Context,
	durationType column.DataLifeCycleState,
	durationID uuid.UUID,
	durationIDRequired bool,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, error) {
	v := validation.NewRetentionDurationsUpdateValidator(durationType)
	if err := v.ValidateTenantUpdate(durationID, durationIDRequired, req.RetentionDuration); err != nil {
		return nil, http.StatusBadRequest, ucerr.Wrap(err)
	}

	code, err := updateRetentionDurations(ctx, durationIDRequired, req.RetentionDuration)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	response, code, err := getRetentionDurationForTenant(ctx, durationType, durationID, durationIDRequired)
	if err != nil {
		return nil, code, ucerr.Wrap(err)
	}

	return response, http.StatusOK, nil
}

// OpenAPI Summary: Create SoftDeleted ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint creates a SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.
func (h *handler) createSoftDeletedRetentionDuration(
	ctx context.Context,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForTenant(ctx, column.DataLifeCycleStateSoftDeleted, req.RetentionDuration.ID, false, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Delete SoftDeleted ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint deletes a specific SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. Once the tenant default retention duration has been deleted, it will fall back to the system default to not retain deleted data.
func (h *handler) deleteSoftDeletedRetentionDuration(
	ctx context.Context,
	durationID uuid.UUID,
	_ url.Values,
) (int, []auditlog.Entry, error) {
	code, err := deleteRetentionDurationForTenant(ctx, column.DataLifeCycleStateSoftDeleted, durationID)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get SoftDeleted ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets a specific SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.
func (h *handler) getSoftDeletedRetentionDuration(
	ctx context.Context,
	durationID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationForTenant(ctx, column.DataLifeCycleStateSoftDeleted, durationID, true)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Default SoftDeleted ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets the default SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. If the retention duration is a user-specified value, id will be non-nil, and use_default will be false.
func (h *handler) listSoftDeletedRetentionDurations(
	ctx context.Context,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, _, err := getRetentionDurationForTenant(ctx, column.DataLifeCycleStateSoftDeleted, uuid.Nil, false)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update SoftDeleted ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint updates a specific SoftDeleted ColumnRetentionDuration for a tenant. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.
func (h *handler) updateSoftDeletedRetentionDuration(
	ctx context.Context,
	durationID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForTenant(ctx, column.DataLifeCycleStateSoftDeleted, durationID, true, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Live ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint creates a Live ColumnRetentionDuration for a tenant.
func (h *handler) createLiveRetentionDuration(
	ctx context.Context,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForTenant(ctx, column.DataLifeCycleStateLive, req.RetentionDuration.ID, false, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Delete Live ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint deletes a specific Live ColumnRetentionDuration for a tenant. Once the tenant default retention duration has been deleted, it will fall back to the system default to retain Live data indefinitely.
func (h *handler) deleteLiveRetentionDuration(
	ctx context.Context,
	durationID uuid.UUID,
	_ url.Values,
) (int, []auditlog.Entry, error) {
	code, err := deleteRetentionDurationForTenant(ctx, column.DataLifeCycleStateLive, durationID)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get Live ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets a specific Live ColumnRetentionDuration for a tenant.
func (h *handler) getLiveRetentionDuration(
	ctx context.Context,
	durationID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationForTenant(ctx, column.DataLifeCycleStateLive, durationID, true)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Default Live ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets the default Live ColumnRetentionDuration for a tenant. If the retention duration is a user-specified value, id will be non-nil, and use_default will be false.
func (h *handler) listLiveRetentionDurations(
	ctx context.Context,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, _, err := getRetentionDurationForTenant(ctx, column.DataLifeCycleStateLive, uuid.Nil, false)
	if err != nil {
		return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Live ColumnRetentionDuration for Tenant
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint updates a specific Live ColumnRetentionDuration for a tenant.
func (h *handler) updateLiveRetentionDuration(
	ctx context.Context,
	durationID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForTenant(ctx, column.DataLifeCycleStateLive, durationID, true, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update SoftDeleted ColumnRetentionDurations for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint updates all specified SoftDeleted column purpose ColumnRetentionDurations for a tenant column. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. For each retention duration, if id is nil and use_default is false, the retention duration will be created; if id is non-nil and use_default is false, the associated retention duration will be updated; or if id is non-nil and use_default is true, the associated retention duration will be deleted. Each column purpose retention duration that has been deleted will fall back to the associated purpose retention duration.
func (h *handler) createSoftDeletedRetentionDurationOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	req idp.UpdateColumnRetentionDurationsRequest,
) (*idp.ColumnRetentionDurationsResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationsForColumn(ctx, column.DataLifeCycleStateSoftDeleted, columnID, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Delete SoftDeleted ColumnRetentionDuration for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint deletes a specific SoftDeleted column purpose ColumnRetentionDuration for a tenant column. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. Once the column purpose retention duration has been deleted, it will fall back to the associated purpose retention duration.
func (h *handler) deleteSoftDeletedRetentionDurationOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	durationID uuid.UUID,
	_ url.Values,
) (int, []auditlog.Entry, error) {
	code, err := deleteRetentionDurationForColumn(ctx, column.DataLifeCycleStateSoftDeleted, columnID, durationID)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get SoftDeleted ColumnRetentionDuration for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets a specific SoftDeleted column purpose ColumnRetentionDuration for a tenant column. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.
func (h *handler) getSoftDeletedRetentionDurationOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	durationID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationForColumn(ctx, column.DataLifeCycleStateSoftDeleted, columnID, durationID)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Default SoftDeleted ColumnRetentionDurations for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets the default SoftDeleted column purpose ColumnRetentionDurations for a tenant column, one for each column purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. For each retention duration, if the retention duration is a user-specified value, id will be non-nil, and use_default will be false.
func (h *handler) listSoftDeletedRetentionDurationsOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationsResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationsForColumn(ctx, column.DataLifeCycleStateSoftDeleted, columnID, uuid.Nil)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update SoftDeleted ColumnRetentionDuration for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint updates a specific SoftDeleted column purpose ColumnRetentionDuration for a tenant column. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.
func (h *handler) updateSoftDeletedRetentionDurationOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	durationID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForColumn(ctx, column.DataLifeCycleStateSoftDeleted, columnID, durationID, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Live ColumnRetentionDurations for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint updates all specified Live column purpose ColumnRetentionDurations for a tenant column. For each retention duration, if id is nil and use_default is false, the retention duration will be created; if id is non-nil and use_default is false, the associated retention duration will be updated; or if id is non-nil and use_default is true, the associated retention duration will be deleted. Each column purpose retention duration that has been deleted will fall back to the associated purpose retention duration.
func (h *handler) createLiveRetentionDurationOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	req idp.UpdateColumnRetentionDurationsRequest,
) (*idp.ColumnRetentionDurationsResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationsForColumn(ctx, column.DataLifeCycleStateLive, columnID, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Delete Live ColumnRetentionDuration for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint deletes a specific Live column purpose ColumnRetentionDuration for a tenant column. Once the column purpose retention duration has been deleted, it will fall back to the associated purpose retention duration.
func (h *handler) deleteLiveRetentionDurationOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	durationID uuid.UUID,
	_ url.Values,
) (int, []auditlog.Entry, error) {
	code, err := deleteRetentionDurationForColumn(ctx, column.DataLifeCycleStateLive, columnID, durationID)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get Live ColumnRetentionDuration for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets a specific Live column purpose ColumnRetentionDuration for a tenant column.
func (h *handler) getLiveRetentionDurationOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	durationID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationForColumn(ctx, column.DataLifeCycleStateLive, columnID, durationID)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Default Live ColumnRetentionDurations for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets the default Live column purpose ColumnRetentionDurations for a tenant column, one for each column purpose. For each retention duration, if the retention duration is a user-specified value, id will be non-nil, and use_default will be false.
func (h *handler) listLiveRetentionDurationsOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationsResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationsForColumn(ctx, column.DataLifeCycleStateLive, columnID, uuid.Nil)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Live ColumnRetentionDuration for Column
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint updates a specific Live column purpose ColumnRetentionDuration for a tenant column.
func (h *handler) updateLiveRetentionDurationOnColumn(
	ctx context.Context,
	columnID uuid.UUID,
	durationID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForColumn(ctx, column.DataLifeCycleStateLive, columnID, durationID, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create SoftDeleted ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint creates a SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.
func (h *handler) createSoftDeletedRetentionDurationOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForPurpose(ctx, column.DataLifeCycleStateSoftDeleted, purposeID, req.RetentionDuration.ID, false, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Delete SoftDeleted ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint deletes a specific SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. Once the purpose default retention duration has been deleted, it will fall back to the tenant retention duration.
func (h *handler) deleteSoftDeletedRetentionDurationOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	_ url.Values,
) (int, []auditlog.Entry, error) {
	code, err := deleteRetentionDurationForPurpose(ctx, column.DataLifeCycleStateSoftDeleted, purposeID, durationID)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get SoftDeleted ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets a specific SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.
func (h *handler) getSoftDeletedRetentionDurationOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationForPurpose(ctx, column.DataLifeCycleStateSoftDeleted, purposeID, durationID, true)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Default SoftDeleted ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets the default SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion. If the retention duration is a user-specified value, id will be non-nil, and use_default will be false.
func (h *handler) listSoftDeletedRetentionDurationsOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationForPurpose(ctx, column.DataLifeCycleStateSoftDeleted, purposeID, uuid.Nil, false)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update SoftDeleted ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint updates a specific SoftDeleted ColumnRetentionDuration for a tenant purpose. SoftDeleted data is data that has been deleted but is retained for a specified purpose and duration after deletion.
func (h *handler) updateSoftDeletedRetentionDurationOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForPurpose(ctx, column.DataLifeCycleStateSoftDeleted, purposeID, durationID, true, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Create Live ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint creates a Live ColumnRetentionDuration for a tenant purpose.
func (h *handler) createLiveRetentionDurationOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForPurpose(ctx, column.DataLifeCycleStateLive, purposeID, req.RetentionDuration.ID, false, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusCreated, nil, nil
}

// OpenAPI Summary: Delete Live ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint deletes a specific Live ColumnRetentionDuration for a tenant purpose. If the purpose default retention duration has been deleted, it will fall back to the tenant retention duration.
func (h *handler) deleteLiveRetentionDurationOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	_ url.Values,
) (int, []auditlog.Entry, error) {
	code, err := deleteRetentionDurationForPurpose(ctx, column.DataLifeCycleStateLive, purposeID, durationID)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return http.StatusNoContent, nil, nil
}

// OpenAPI Summary: Get Live ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets a specific Live ColumnRetentionDuration for a tenant purpose.
func (h *handler) getLiveRetentionDurationOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationForPurpose(ctx, column.DataLifeCycleStateLive, purposeID, durationID, true)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Get Default Live ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint gets the default Live ColumnRetentionDuration for a tenant purpose. If the retention duration is a user-specified value, id will be non-nil, and use_default will be false.
func (h *handler) listLiveRetentionDurationsOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	_ url.Values,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := getRetentionDurationForPurpose(ctx, column.DataLifeCycleStateLive, purposeID, uuid.Nil, false)
	if err != nil {
		switch code {
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}

// OpenAPI Summary: Update Live ColumnRetentionDuration for Purpose
// OpenAPI Tags: ColumnRetentionDurations
// OpenAPI Description: This endpoint updates a specific Live ColumnRetentionDuration for a tenant purpose.
func (h *handler) updateLiveRetentionDurationOnPurpose(
	ctx context.Context,
	purposeID uuid.UUID,
	durationID uuid.UUID,
	req idp.UpdateColumnRetentionDurationRequest,
) (*idp.ColumnRetentionDurationResponse, int, []auditlog.Entry, error) {
	result, code, err := updateRetentionDurationForPurpose(ctx, column.DataLifeCycleStateLive, purposeID, durationID, true, req)
	if err != nil {
		switch code {
		case http.StatusBadRequest:
			return nil, http.StatusBadRequest, nil, ucerr.Wrap(err)
		case http.StatusConflict:
			return nil, http.StatusConflict, nil, ucerr.Wrap(err)
		case http.StatusNotFound:
			return nil, http.StatusNotFound, nil, ucerr.Wrap(err)
		default:
			return nil, http.StatusInternalServerError, nil, ucerr.Wrap(err)
		}
	}

	return result, http.StatusOK, nil, nil
}
