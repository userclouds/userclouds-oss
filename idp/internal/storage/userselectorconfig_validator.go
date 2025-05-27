package storage

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage/column"
	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
)

type userSelectorConfigFormatter struct {
	cm                      *ColumnManager
	dtm                     *DataTypeManager
	whereParts              []string
	forValidation           bool
	checkUserSelectorValues bool
	totalSelectorValues     int
}

func newExecutionSelectorConfigFormatter(
	cm *ColumnManager,
	dtm *DataTypeManager,
	usc userstore.UserSelectorConfig,
	totalSelectorValues int,
) userSelectorConfigFormatter {
	return userSelectorConfigFormatter{
		cm:                      cm,
		dtm:                     dtm,
		whereParts:              strings.Split(usc.WhereClause+" ", "?"),
		checkUserSelectorValues: true,
		totalSelectorValues:     totalSelectorValues,
	}
}

func newValidationSelectorConfigFormatter(
	cm *ColumnManager,
	dtm *DataTypeManager,
	usc userstore.UserSelectorConfig,
) userSelectorConfigFormatter {
	return userSelectorConfigFormatter{
		cm:            cm,
		dtm:           dtm,
		whereParts:    strings.Split(usc.WhereClause+" ", "?"),
		forValidation: true,
	}
}

func (uscf userSelectorConfigFormatter) format() (whereClause string, anySubFields bool, err error) {
	if uscf.checkUserSelectorValues {
		if totalWhereParts := len(uscf.whereParts) - 1; totalWhereParts != uscf.totalSelectorValues {
			return "",
				false,
				ucerr.Friendlyf(
					nil,
					"number of UserSelectorValues (%d) != number of parameters in WhereClause (%d)",
					uscf.totalSelectorValues,
					totalWhereParts,
				)
		}
	}

	// convert the where clause parameters from ? to $1, $2, etc.
	whereClause = uscf.whereParts[0]
	for i := 1; i < len(uscf.whereParts); i++ {
		whereClause += fmt.Sprintf("$%d%s", i, uscf.whereParts[i])
	}

	// add parentheses around each sub-clause

	orRE := regexp.MustCompile(`(?i) OR `)
	andRE := regexp.MustCompile(`(?i) AND `)
	whereClause = orRE.ReplaceAllString(whereClause, ") OR (")
	whereClause = andRE.ReplaceAllString(whereClause, ") AND (")
	whereClause = fmt.Sprintf("(%s)", whereClause)

	// ensure all columns are recognized and format any subfield queries appropriately

	anySubFields = false

	for _, columnMatch := range constants.ReferencedColumnRE.FindAllStringSubmatch(whereClause, -1) {
		c := uscf.cm.GetUserColumnByName(columnMatch[1])
		if c == nil {
			return "", false, ucerr.Friendlyf(nil, "'%s' is not a recognized column name", columnMatch[1])
		}

		if columnMatch[2] != "" {
			subFieldQuery := fmt.Sprintf("{%s}%s", columnMatch[1], columnMatch[2])
			subFieldParts := strings.Split(columnMatch[2], "'")
			if len(subFieldParts) != 3 {
				return "", false, ucerr.Friendlyf(nil, "'%s' is an invalid query parameter", subFieldQuery)
			}
			subFieldName := subFieldParts[1]

			dt := uscf.dtm.GetDataTypeByID(c.DataTypeID)
			if dt == nil {
				return "", false, ucerr.Friendlyf(nil, "column '%s' has an unrecognized data type '%v'", c.Name, c.DataTypeID)
			}

			subFields := uscf.getSelectorSubFields(dt)
			if len(subFields) == 0 {
				return "", false, ucerr.Friendlyf(nil, "'%s' is not a JSONB column", columnMatch[1])
			}
			subFieldDataTypeID, found := subFields[subFieldName]
			if !found {
				return "",
					false,
					ucerr.Friendlyf(
						nil,
						"'%s' is not a supported sub-field of JSONB column '%s'",
						subFieldName,
						columnMatch[1],
					)
			}

			subFieldColumnName, subFieldSuffix, err := uscf.getSubFieldInfo(subFieldDataTypeID)
			if err != nil {
				return "", false, ucerr.Wrap(err)
			}

			whereClause = strings.ReplaceAll(whereClause, subFieldQuery, fmt.Sprintf("(%s)::%s", subFieldQuery, subFieldSuffix))
			if uscf.forValidation {
				whereClause = strings.ReplaceAll(whereClause, subFieldQuery, fmt.Sprintf("ucv.%s", subFieldColumnName))
			}
			anySubFields = true
		}
	}

	return whereClause, anySubFields, nil
}

func (userSelectorConfigFormatter) getSelectorSubFields(dt *column.DataType) map[string]uuid.UUID {
	if !dt.IsComposite() {
		return nil
	}

	subFields := map[string]uuid.UUID{}
	for _, field := range dt.CompositeAttributes.Fields {
		subFields[field.StructName] = field.DataTypeID
	}
	return subFields
}

func (userSelectorConfigFormatter) getSubFieldInfo(
	subFieldDataType uuid.UUID,
) (columnName string, suffix string, err error) {
	switch subFieldDataType {
	case datatype.String.ID:
		return "varchar_value", "VARCHAR", nil
	case datatype.Boolean.ID:
		return "boolean_value", "BOOLEAN", nil
	case datatype.Integer.ID:
		return "int_value", "INTEGER", nil
	case datatype.Timestamp.ID, datatype.Date.ID:
		return "timestamp_value", "TIMESTAMP", nil
	case datatype.UUID.ID:
		return "uuid_value", "UUID", nil
	default:
		return "", "", ucerr.Errorf("unsupported composite column field type '%v'", subFieldDataType)
	}
}

const userSelectorValidationQueryTemplate = `
/* lint-sql-unsafe-columns bypass-known-table-check */
SELECT DISTINCT
u.id
FROM
users u
%s
WHERE
%s
`

// UserSelectorConfigValidator is used to validate a UserSelectorConfig
type UserSelectorConfigValidator struct {
	ctx  context.Context
	cm   *ColumnManager
	dtm  *DataTypeManager
	s    *Storage
	dlcs column.DataLifeCycleState
}

// NewUserSelectorConfigValidator returns a configured UserSelectorConfigValidator
func NewUserSelectorConfigValidator(
	ctx context.Context,
	s *Storage,
	cm *ColumnManager,
	dtm *DataTypeManager,
	dlcs column.DataLifeCycleState,
) UserSelectorConfigValidator {
	return UserSelectorConfigValidator{
		ctx:  ctx,
		cm:   cm,
		dtm:  dtm,
		s:    s,
		dlcs: dlcs,
	}
}

// Validate validates the passed in UserSelectorConfig
func (uscv UserSelectorConfigValidator) Validate(usc userstore.UserSelectorConfig) error {
	if err := usc.Validate(); err != nil {
		return ucerr.Wrap(err)
	}

	if usc.MatchesAll() {
		return nil
	}

	// format and prepare query for all referenced columns

	whereClause, anyNonSystemColumns, err := newValidationSelectorConfigFormatter(uscv.cm, uscv.dtm, usc).format()
	if err != nil {
		return ucerr.Wrap(err)
	}

	for _, c := range uscv.cm.GetColumns() {
		ci := newColumnInfo(c, uscv.dlcs)
		bracketedColumnName := ci.getBracketedColumnName()
		if !strings.Contains(whereClause, bracketedColumnName) {
			continue
		}

		if ci.isSystemColumn() {
			whereClause = strings.ReplaceAll(whereClause, bracketedColumnName, "u."+ci.getDBColumnName())
		} else {
			userRowColumnName, err := ci.getUserRowColumnName()
			if err != nil {
				return ucerr.Wrap(err)
			}
			whereClause = strings.ReplaceAll(whereClause, bracketedColumnName, "ucv."+userRowColumnName)
			anyNonSystemColumns = true
		}
	}

	joinClause := ""
	if anyNonSystemColumns {
		urti, err := newUserRowsTableInfo(uscv.dlcs)
		if err != nil {
			return ucerr.Wrap(err)
		}

		joinClause = fmt.Sprintf(
			" LEFT JOIN %s ucv ON ucv.user_id = u.id",
			urti.getUserColumnValuesTableName(),
		)
	}

	query := fmt.Sprintf(userSelectorValidationQueryTemplate, joinClause, whereClause)
	statement, err := uscv.s.db.PrepareContext(uscv.ctx, query)
	if err != nil {
		return ucerr.Wrap(err)
	}
	if err := statement.Close(); err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}
