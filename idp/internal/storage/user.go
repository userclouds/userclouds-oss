package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/userstore"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/uuidarray"
)

// BaseUser represents a single user account in our system.
type BaseUser struct {
	ucdb.VersionBaseModel
	OrganizationID uuid.UUID         `db:"organization_id" json:"organization_id"`
	Region         region.DataRegion `db:"region" json:"region"`
}

func (BaseUser) getPaginationKeys() pagination.KeyTypes {
	return pagination.KeyTypes{
		"organization_id": pagination.UUIDKeyType,
	}
}

//go:generate genpageable BaseUser

//go:generate genvalidate BaseUser

//go:generate genorm --nosave --nodelete --storageclassprefix --accessprimarydboption BaseUser users tenantdb User

// ConsentedPurpose represents a single consented purpose, and
// includes a purpose id and retention timeout, after which the
// consent is no longer valid.
type ConsentedPurpose struct {
	Purpose          uuid.UUID `json:"purpose"`
	RetentionTimeout time.Time `json:"retention_timeout"`
}

func (cp ConsentedPurpose) isDefault() bool {
	return cp.Purpose.IsNil()
}

// GetFriendlyDescription returns a user-friendly description of the consented purpose
func (cp ConsentedPurpose) GetFriendlyDescription() string {
	if cp.isDefault() {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ID:'%v'", cp.Purpose)
	if cp.RetentionTimeout != userstore.GetRetentionTimeoutIndefinite() {
		fmt.Fprintf(&b, ", Expires:'%v'", cp.RetentionTimeout)
	}
	return b.String()
}

// ColumnConsentedValue represents a single column consented value,
// including a unique ID, the column name, the value (the type
// of which is column-specific), a ordering field, and a list of
// consented purposes.  For system columns, the ID is the user ID,
// while for non-system columns, the ID identifies the specific
// value row in the underlying user column values table.
type ColumnConsentedValue struct {
	ID                uuid.UUID          `json:"id"`
	Version           int                `json:"version"`
	ColumnName        string             `json:"column_name"`
	Value             any                `json:"value"`
	Ordering          int                `json:"ordering"`
	ConsentedPurposes []ConsentedPurpose `json:"consented_purposes"`
}

func (ccv ColumnConsentedValue) hasConsentedPurposes() bool {
	return len(ccv.ConsentedPurposes) > 0 && !ccv.ConsentedPurposes[0].isDefault()
}

// GetPurposeIDs returns the list of purpose ids that have been consented
func (ccv ColumnConsentedValue) GetPurposeIDs() []uuid.UUID {
	var purposeIDs []uuid.UUID
	for _, consentedPurpose := range ccv.ConsentedPurposes {
		if !consentedPurpose.isDefault() {
			purposeIDs = append(purposeIDs, consentedPurpose.Purpose)
		}
	}

	return purposeIDs
}

// GetFriendlyDescription returns a user-friendly description of the ColumnConsentedValue
func (ccv ColumnConsentedValue) GetFriendlyDescription() string {
	var b strings.Builder
	fmt.Fprintf(&b, "(Column:'%s', Value:'%v'", ccv.ColumnName, ccv.Value)
	if ccv.hasConsentedPurposes() {
		b.WriteString(", Purposes:[")
		for _, cp := range ccv.ConsentedPurposes {
			fmt.Fprintf(&b, " %s", cp.GetFriendlyDescription())
		}
		b.WriteString(" ]")
	}
	b.WriteString(")")
	return b.String()
}

// ColumnConsentedValues is a map from column name to ColumnConsentedValue
// ID to ColumnConsentedValue
type ColumnConsentedValues map[string]map[uuid.UUID]ColumnConsentedValue

// ConsentedPurposeIDs is a list of purpose UUIDs associated with a column in a row of user data
type ConsentedPurposeIDs uuidarray.UUIDArray

// User represents a single user account in our system, including profile and consented purpose info
//
//go:generate easyjson -local_prefix=userclouds.com .
//easyjson:json
type User struct {
	BaseUser

	// The profile contains custom fields defined per tenant in the userstore
	Profile                    userstore.Record                 `db:"-" json:"profile"`
	ProfileConsentedPurposeIDs map[string][]ConsentedPurposeIDs `db:"-" json:"profile_consented_purpose_ids"`
	ColumnValues               ColumnConsentedValues            `db:"-" json:"column_values"`
}

//go:generate genvalidate User

//go:generate genorm --noget --nolist --storageclassprefix User users tenantdb User

// MarkUserUpdated will update the updated time and version for the specified user
func (s *UserStorage) MarkUserUpdated(ctx context.Context, u *BaseUser) error {
	const q = "UPDATE users SET updated=NOW(), _version=$3 WHERE id=$1 AND _version=$2 AND deleted='0001-01-01 00:00:00'::TIMESTAMP;"

	newVersion := u.Version + 1
	if _, err := s.db.ExecContext(ctx, "MarkUserUpdated", q, u.ID, u.Version, newVersion); err != nil {
		return ucerr.Wrap(err)
	}
	return nil
}

// DeletePartiallyCreatedUser is used to delete a user for which creation via a mutator partially failed.
// This can happen when a user is created via a mutator, since the creation involves inserting a record
// into users and potentially some records into user_column_pre_delete_values, and any of those inserts
// can fail. Since the user is not in a valid state at this point, we eschew the normal mutator flow and
// directly remove any user_column_pre_delete_values entries, since permissions do not matter and we
// do not need or want to retain any of the deleted data.
func (s *UserStorage) DeletePartiallyCreatedUser(ctx context.Context, userID uuid.UUID) error {
	// first delete any user_column_pre_delete_values records for the user ID

	const q = "DELETE FROM user_column_pre_delete_values WHERE user_id=$1;"

	if _, err := s.db.ExecContext(ctx, "DeletePartiallyCreatedUser", q, userID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(err)
		}
	}
	// finally soft-delete the users record for the user ID
	if err := s.DeleteUser(ctx, userID); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return ucerr.Wrap(err)
		}
	}

	return nil
}
