package userstore

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dlclark/regexp2"
	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/idp"
	"userclouds.com/idp/events"
	"userclouds.com/idp/helpers"
	"userclouds.com/idp/internal"
	"userclouds.com/idp/internal/constants"
	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/tokenizer"
	"userclouds.com/idp/internal/userstore/sqlparse"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/auth"
	"userclouds.com/internal/sqlshim"
	"userclouds.com/internal/tenantmap"
	logServerClient "userclouds.com/logserver/client"
	"userclouds.com/worker"
)

// ProxyHandlerFactory creates a new IdpSQLQueryHandler
type ProxyHandlerFactory struct{}

// NewProxyHandler implements sqlshim.HandlerFactory
func (f ProxyHandlerFactory) NewProxyHandler(ctx context.Context, databaseID uuid.UUID, ts *tenantmap.TenantState, azc *authz.Client, wc workerclient.Client, lgsc *logServerClient.Client, jwtVerifier auth.Verifier) sqlshim.Observer {
	return NewIdpPsqlQueryHandler(ctx, databaseID, ts, azc, wc, lgsc, jwtVerifier)
}

// IdpSQLQueryHandler handles queries from the sqlshim proxy
type IdpSQLQueryHandler struct {
	databaseID  uuid.UUID
	ts          *tenantmap.TenantState
	azc         *authz.Client
	wc          workerclient.Client
	lgsc        *logServerClient.Client
	s           *storage.Storage
	jwtVerifier auth.Verifier
}

// NewIdpPsqlQueryHandler creates a new IdpSQLQueryHandler
func NewIdpPsqlQueryHandler(ctx context.Context, databaseID uuid.UUID, ts *tenantmap.TenantState, azc *authz.Client, wc workerclient.Client, lgsc *logServerClient.Client, jwtVerifier auth.Verifier) *IdpSQLQueryHandler {
	return &IdpSQLQueryHandler{
		databaseID:  databaseID,
		ts:          ts,
		azc:         azc,
		wc:          wc,
		lgsc:        lgsc,
		s:           storage.NewFromTenantState(ctx, ts),
		jwtVerifier: jwtVerifier,
	}
}

type columnTransformInfo struct {
	column              storage.Column
	table               string
	transformer         *storage.Transformer
	tokenAccessPolicyID uuid.UUID
}

type columnTransformInfoByLowercase map[string]*columnTransformInfo

func (ctibl columnTransformInfoByLowercase) get(name string) (*columnTransformInfo, bool) {
	cti, found := ctibl[strings.ToLower(name)]
	return cti, found
}

func (ctibl *columnTransformInfoByLowercase) put(name string, cti *columnTransformInfo) {
	(*ctibl)[strings.ToLower(name)] = cti
}

type columnsByLowercase map[string]*storage.Column

func (cbl columnsByLowercase) get(name string) (*storage.Column, bool) {
	c, found := cbl[strings.ToLower(name)]
	return c, found
}

func (cbl *columnsByLowercase) put(name string, c *storage.Column) {
	(*cbl)[strings.ToLower(name)] = c
}

type transformerAndTokenAccessPolicyID struct {
	transformer         *storage.Transformer
	tokenAccessPolicyID uuid.UUID
}

type transformersByLowercase struct {
	transformerMap     map[string]transformerAndTokenAccessPolicyID
	defaultTransformer *storage.Transformer
}

func newTransformersByLowercase(defaultTransformer *storage.Transformer) transformersByLowercase {
	return transformersByLowercase{
		transformerMap:     map[string]transformerAndTokenAccessPolicyID{},
		defaultTransformer: defaultTransformer,
	}
}

func (tbl transformersByLowercase) get(ctx context.Context, name string) transformerAndTokenAccessPolicyID {
	transformedName := strings.ToLower(name)
	if t, found := tbl.transformerMap[transformedName]; found {
		return t
	}

	uclog.Debugf(
		ctx,
		"no transformer for column %s; possible cause is schema mismatch",
		transformedName,
	)
	return transformerAndTokenAccessPolicyID{transformer: tbl.defaultTransformer}
}

func (tbl *transformersByLowercase) put(name string, t *storage.Transformer, tokenAccessPolicyID uuid.UUID) {
	tbl.transformerMap[strings.ToLower(name)] = transformerAndTokenAccessPolicyID{transformer: t, tokenAccessPolicyID: tokenAccessPolicyID}
}

type transformInfo struct {
	s         *storage.Storage
	dtm       *storage.DataTypeManager
	queryType sqlparse.QueryType

	accessor  *storage.Accessor
	startTime time.Time

	ctis columnTransformInfoByLowercase

	accessPolicy        *policy.AccessPolicy
	apContext           policy.AccessPolicyContext
	transformerExecutor *tokenizer.TransformerExecutor

	maxResults int
}

func (ti transformInfo) shouldTransform() bool {
	// only transform rows from SELECT queries
	return ti.queryType == sqlparse.QueryTypeSelect
}

func (ti transformInfo) prepareRow(
	ctx context.Context,
	colNames []string,
	values [][]byte,
) (bool, *policy.AccessPolicyContext, []transformableValue, []tokenizer.ExecuteTransformerParameters) {
	if len(values) != len(colNames) ||
		len(values) != len(ti.ctis) {
		// TODO: log an error but do not cause the proxy to fail; see issue #4563
		uclog.Debugf(
			ctx,
			"number of values (%d), columns (%d), transformers (%d) should all match; possible cause is schema mismatch",
			len(values),
			len(colNames),
			len(ti.ctis),
		)
		return false, nil, nil, nil
	}

	apContext := ti.apContext
	apContext.RowData = make(map[string]string, len(colNames))
	transformableValues := make([]transformableValue, 0, len(colNames))
	var transformerParams []tokenizer.ExecuteTransformerParameters

	for i, colName := range colNames {
		cti, found := ti.ctis.get(colName)
		if !found {
			uclog.Debugf(
				ctx,
				"could not find columnTransformInfo for column %s",
				colName,
			)
			return false, nil, nil, nil
		}

		transformedColName := strings.ToLower(fmt.Sprintf("%s.%s", cti.table, colName))

		apContext.RowData[transformedColName] = string(values[i])

		transformer := cti.transformer

		var value any
		if values[i] != nil {
			value = string(values[i])
		}

		tv, err := newTransformableOutputValue(
			ctx,
			ti.dtm,
			cti.column,
			*transformer,
			cti.column.IsArray,
			value,
		)
		if err != nil {
			uclog.Debugf(
				ctx,
				"could not get transformable output value for column %s: %v",
				transformedColName,
				err,
			)
			return false, nil, nil, nil
		}

		if tv.shouldTransform {
			inputs, err := tv.getTransformableInputs(ctx)
			if err != nil {
				uclog.Debugf(
					ctx,
					"could not get transformable inputs for column %s: %v",
					transformedColName,
					err,
				)
				return false, nil, nil, nil
			}

			for _, input := range inputs {
				transformerParams = append(
					transformerParams,
					tokenizer.ExecuteTransformerParameters{
						Transformer:         transformer,
						TokenAccessPolicyID: cti.tokenAccessPolicyID,
						Data:                input,
					},
				)
				tv.addValueIndex(len(transformerParams) - 1)
			}
		}

		if err := tv.Validate(); err != nil {
			uclog.Debugf(
				ctx,
				"validation failed for column %s transformableValue: %v",
				transformedColName,
				err,
			)
			return false, nil, nil, nil
		}

		transformableValues = append(transformableValues, *tv)
	}

	return true, &apContext, transformableValues, transformerParams
}

func createAllowAllAccessPolicy(ctx context.Context, s *storage.Storage, azc *authz.Client, lgsc *logServerClient.Client, tenantID uuid.UUID, name string) (*storage.AccessPolicy, error) {
	ap := &storage.AccessPolicy{
		SystemAttributeBaseModel: ucdb.NewSystemAttributeBase(),
	}
	ap.Name = name
	ap.PolicyType = storage.InternalPolicyTypeFromClient(policy.PolicyTypeCompositeAnd)
	ap.IsAutogenerated = true
	ap.ComponentIDs = append(ap.ComponentIDs, policy.AccessPolicyAllowAll.ID)
	ap.ComponentParameters = append(ap.ComponentParameters, "")
	ap.ComponentTypes = append(ap.ComponentTypes, int32(storage.AccessPolicyComponentTypePolicy))

	if err := tokenizer.SaveAccessPolicyWithAuthz(ctx, s, azc, ap); err != nil {
		return nil, ucerr.Wrap(err)
	}

	// Create event types for the new access policy without blocking the response
	if lgsc != nil {
		go func() {
			e := events.GetEventsForAccessPolicy(ap.ID, ap.Version)

			if _, err := lgsc.CreateEventTypesForTenant(context.Background(), "tokenizer", uuid.Nil, tenantID, &e); err != nil {
				uclog.Errorf(ctx, "Failed to create event types for access policy with ID %s: %v", ap.ID, err)
			}
		}()
	}
	return ap, nil
}

var accessorNameInvalidChars = regexp.MustCompile(`[^a-zA-Z0-9_\-]+`)
var commentRegex = regexp.MustCompile(`/\*[^*]*\*+(?:[^/*][^*]*\*+)*/`)

// keyValueRegex is a regex that matches key-value pairs in comments
// ([^=\s,]+) is the key (anything other than =, whitespace, or ,s)
// \s*=\s* is the = sign, possibly surrounded by whitespace
// (["'])? is an optional quote character
// ((?:(?!\2)(?(2).|[^\s]))*) is the value and possibly empty
//
//	(?:(?!\2) is a negative lookahead to ensure the value is not the opening quote (if present)
//	(?(2).|[^\s]) is a conditional to allow whitespace in the value iff the opening quote is present
//
// (?(2)\2|) is an optional closing quote character matching the opening quote with backreference
var keyValueRegex = regexp2.MustCompile(`([^=\s,]+)\s*=\s*(["'])?((?:(?!\2)(?(2).|[^\s]))*)(?(2)\2|)`, regexp2.None)
var stringLiteralsRegex = regexp.MustCompile(`'(?:[^']*(?:'')*)*'`)

// CleanupTransformerExecution cleans up any resources associated with transformer execution
func (h *IdpSQLQueryHandler) CleanupTransformerExecution(t any) {
	ti := t.(transformInfo)
	ti.transformerExecutor.CleanupExecution()
}

// HandleQuery handles a query from the sqlshim proxy
// we name return values here in order to log the reason in deferred logUnhandledQuery
// (they are carefully named to be unused in the function body to avoid confusion)
// STG (2/25): I don't love duplicating the reason string (vs error), but there are
// a few cases where there is not actually an error, it's a design choice, and I
// didn't want to override/confuse the issue. There's also an issue of UCError vs Friendly
func (h *IdpSQLQueryHandler) HandleQuery(
	ctx context.Context,
	dbt sqlshim.DatabaseType,
	queryString string,
	tableSchema string,
	connectionID uuid.UUID,
) (responseType sqlshim.HandleQueryResponse, transformInfoStruct any, reason string, returnError error) {
	startTime := time.Now().UTC()

	uclog.DebugfPII(ctx, "sqlshim query: %s", queryString)
	logUnhandledQuery := true
	keyValuePairs := map[string]any{}

	defer func() {
		if logUnhandledQuery {
			logPassthroughQuery(ctx, queryString, keyValuePairs, reason)
		}
	}()

	// Capture key value pairs from comments in the query
	comments := commentRegex.FindAllStringSubmatch(queryString, -1)
	token := ""
	for _, c := range comments {
		comment := strings.TrimSuffix(strings.TrimPrefix(c[0], "/*"), "*/")
		match, err := keyValueRegex.FindStringMatch(comment)
		if err != nil {
			uclog.Warningf(ctx, "failed to find key value pairs in comment %s: %v", comment, err)
			continue
		}
		for match != nil {
			val := match.GroupByNumber(3).String()
			if key := match.GroupByNumber(1).String(); key == "token" {
				token = val
			} else if key == "refresh_schemas" {
				if err := helpers.IngestSqlshimDatabaseSchemas(ctx, h.ts, h.databaseID); err != nil {
					uclog.Errorf(ctx, "failed to ingest sqlshim database schemas for db %d: %v", h.databaseID, err)
				}
			} else {
				keyValuePairs[key] = val
			}
			match, err = keyValueRegex.FindNextMatch(match)
			if err != nil {
				uclog.Warningf(ctx, "failed to find next key value pair in comment %s: %v", comment, err)
				break
			}
		}
	}
	if token != "" {
		ctxToken, err := auth.AddTokenToContext(ctx, token, h.jwtVerifier, false)
		if err != nil {
			return sqlshim.Passthrough, nil, "token error", ucerr.Wrap(err)
		}
		ctx = ctxToken
	}

	queryToParse := queryString
	if dbt == sqlshim.DatabaseTypeMySQL {
		// equivalent of ` in MySQL is " in PostGres (for refering to database references),
		// and " and ' are both valid in MySQL but only ' is valid in PostGres (for referring to string literals)
		queryToParse = strings.ReplaceAll(strings.ReplaceAll(queryString, "\"", "'"), "`", "\"")
	}
	sanitizedQuery := stringLiteralsRegex.ReplaceAllString(commentRegex.ReplaceAllString(queryToParse, ""), "'?'")
	query, err := sqlparse.ParseQuery(queryToParse)
	if err != nil {
		uclog.DebugfPII(ctx, `failed to parse query "%v": %v`, queryString, err)
		return sqlshim.Passthrough, nil, "did not parse query", nil // ignore errors, let the proxy handle them
	}
	if tableSchema != "" {
		for i, c := range query.Columns {
			query.Columns[i].Table = tableSchema + "." + c.Table
		}
	}

	s := storage.NewFromTenantState(ctx, h.ts)

	apContext := tokenizer.BuildBaseAPContext(ctx, keyValuePairs, policy.ActionExecute)
	apContext.Query = map[string]string{
		"type": string(query.Type),
	}
	apContext.ConnectionID = connectionID

	switch query.Type {
	case sqlparse.QueryTypeUpdate, sqlparse.QueryTypeInsert, sqlparse.QueryTypeDelete:
		globalAP, _, thresholdAP, err :=
			s.GetAccessPolicies(
				ctx,
				h.ts.ID,
				policy.AccessPolicyGlobalMutatorID,
				policy.AccessPolicyAllowAll.ID,
			)
		if err != nil {
			return sqlshim.Passthrough, nil, "failed to get access policies", ucerr.Wrap(err)
		}

		// sqlshim mutator execution is identified for rate limiting via a well-known sentinel UUID
		sentinelSQLShimMutationID := uuid.Must(uuid.FromString("7581cba6-a98c-416a-a412-e29c66b7c6be"))

		allowed, err := thresholdAP.CheckRateThreshold(ctx, s, apContext, sentinelSQLShimMutationID)
		if err != nil {
			return sqlshim.Passthrough, nil, "failed to check rate threshold", ucerr.Wrap(err)
		}

		if allowed {
			allowed, _, err = tokenizer.ExecuteAccessPolicy(ctx, globalAP, apContext, h.azc, s)
			if err != nil {
				return sqlshim.Passthrough, nil, "failed to execute access policy", ucerr.Wrap(err)
			}
		}

		if !allowed {
			return sqlshim.AccessDenied, &transformInfo{queryType: query.Type}, "access policy denied", nil
		}

		return sqlshim.Passthrough, nil, "Update/Insert/Delete query", nil
	}

	dtm, err := storage.NewDataTypeManager(ctx, s)
	if err != nil {
		return sqlshim.Passthrough, nil, "failed to create data type manager", ucerr.Wrap(err)
	}

	cm, err := storage.NewColumnManager(ctx, s, h.databaseID)
	if err != nil {
		return sqlshim.Passthrough, nil, "failed to create column manager", ucerr.Wrap(err)
	}

	// Expand star columns
	columns := []sqlparse.Column{}
	for _, c := range query.Columns {
		if c.Name == "*" {
			if c.Table == "" {
				return sqlshim.Passthrough, nil, "star column must be qualified with a table name", ucerr.Friendlyf(nil, "star column must be qualified with a table name")
			}
			tableColumns := cm.GetColumnsByTable(c.Table)
			if len(tableColumns) == 0 {
				return sqlshim.Passthrough, nil, "table not found", ucerr.Friendlyf(nil, "table not found: %s", c.Table)
			}
			for _, col := range tableColumns {
				columns = append(columns, sqlparse.Column{
					Table: c.Table,
					Name:  col.Name,
				})
			}
		} else {
			columns = append(columns, c)
		}
	}

	var accessor *storage.Accessor
	accessorName, ok := keyValuePairs["accessor_name"].(string)
	var randomizeName bool
	if ok {
		accessor, err = s.GetAccessorByName(ctx, accessorName)
		if err != nil {
			accessor = nil
		} else if !accessorSignatureMatches(accessor, columns, query.Selector, cm) {
			return sqlshim.Passthrough, nil, "named accessor does not match query signature", ucerr.Friendlyf(nil, "accessor '%s' does not match query signature", accessorName)
		}
	} else {
		// unnamed accessor -- generate a name based on the query
		columnSet := set.NewStringSet()
		tableSet := set.NewStringSet()
		for _, col := range columns {
			columnSet.Insert(col.Name)
			tableSet.Insert(col.Table)
		}
		accessorName = "SELECT_" + strings.Join(columnSet.Items(), "-") + "_FROM_" + strings.Join(tableSet.Items(), "-") + "_WHERE_" + query.Selector
		accessorName = strings.ReplaceAll(accessorName, "!", "NOT")
		accessorName = accessorNameInvalidChars.ReplaceAllString(accessorName, "")
		if len(accessorName) > 256 {
			accessorName = accessorName[:256]
		}

		// check to see if an accessor already exists based on the signature of this query
		pager, err := storage.NewAccessorPaginatorFromOptions()
		if err != nil {
			return sqlshim.Passthrough, nil, "failed to create accessor paginator", ucerr.Wrap(err)
		}
		for {
			accessors, pr, err := s.GetLatestAccessors(ctx, *pager)
			if err != nil {
				return sqlshim.Passthrough, nil, "failed to get latest accessors", ucerr.Wrap(err)
			}

			for _, a := range accessors {
				if accessorSignatureMatches(&a, columns, query.Selector, cm) {
					uclog.Infof(ctx, "found existing accessor: %v", a)
					accessor = &a
					break
				}

				// if we got here, signature did not match
				if a.Name == accessorName {
					// if we don't find another accessor with a signature match, we're just going
					// to 409 on accessor creation, so we're going to remind ourselves here to
					// randomize the name later if needed
					randomizeName = true
				}
			}

			// needed because we're inside a double for loop above
			if accessor != nil {
				break
			}

			if !pager.AdvanceCursor(*pr) {
				break
			}
		}
	}

	// Create a new accessor if one with this name or signature doesn't exist yet
	if accessor == nil {
		if randomizeName {
			if len(accessorName) > 252 {
				accessorName = accessorName[:251]
			}
			accessorName = accessorName + "_" + uuid.Must(uuid.NewV4()).String()[:4]
		}

		uclog.Infof(ctx, "creating new accessor")
		mm := storage.NewMethodManager(ctx, s)

		newAccessor := &userstore.Accessor{
			ID:                 uuid.Must(uuid.NewV4()),
			Name:               accessorName,
			Description:        "Accessor generated from query: " + sanitizedQuery,
			Version:            1,
			DataLifeCycleState: userstore.DataLifeCycleStateLive,
			SelectorConfig: userstore.UserSelectorConfig{
				WhereClause: query.Selector,
			},
			Purposes:        []userstore.ResourceID{{ID: constants.OperationalPurposeID}},
			Columns:         []userstore.ColumnOutputConfig{},
			IsAuditLogged:   true,
			IsAutogenerated: true,
		}

		for _, c := range columns {
			col := cm.GetColumnByTableAndName(c.Table, c.Name)
			if col == nil {
				cols := cm.GetColumnsByTable(c.Table)
				var colNames []string
				for _, col := range cols {
					colNames = append(colNames, col.Name)
				}
				err := ucerr.Errorf("column %s not found, columns in table: %v", c, colNames)

				return sqlshim.Passthrough, nil, "column not found", ucerr.Friendlyf(err, "column not found: %s, columns in table %s: %v", c, c.Table, colNames)
			}
			newAccessor.Columns = append(newAccessor.Columns, userstore.ColumnOutputConfig{
				Column: userstore.ResourceID{ID: col.ID},
			})
		}

		ap, err := createAllowAllAccessPolicy(ctx, s, h.azc, h.lgsc, h.ts.ID, "AccessPolicyForAccessor_"+newAccessor.ID.String())
		if err != nil {
			return sqlshim.Passthrough, nil, "failed to create access policy", ucerr.Wrap(err)
		}

		newAccessor.AccessPolicy = userstore.ResourceID{ID: ap.ID}
		if err := newAccessor.Validate(); err != nil {
			if err := tokenizer.DeleteAccessPolicyWithAuthz(ctx, s, h.azc, ap); err != nil {
				uclog.Errorf(ctx, "failed to delete access policy: %v", err)
			}
			return sqlshim.Passthrough, nil, "failed to validate accessor", ucerr.Wrap(err)
		}

		if _, err := mm.CreateAccessorFromClient(ctx, newAccessor); err != nil {
			if err := tokenizer.DeleteAccessPolicyWithAuthz(ctx, s, h.azc, ap); err != nil {
				uclog.Errorf(ctx, "failed to delete access policy: %v", err)
			}
			return sqlshim.Passthrough, nil, "failed to create accessor", ucerr.Wrap(err)
		}

		// Create event types for the new accessor without blocking the response
		if h.lgsc != nil {
			go func() {
				e := events.GetEventsForAccessor(newAccessor.ID, newAccessor.Version)

				if _, err := h.lgsc.CreateEventTypesForTenant(context.Background(), "idp", uuid.Nil, h.ts.ID, &e); err != nil {
					uclog.Errorf(ctx, "failed to create event types for accessor %v: %v", newAccessor.ID, err)
				}
			}()
		}

		accessor, err = s.GetLatestAccessor(ctx, newAccessor.ID)
		if err != nil {
			return sqlshim.Passthrough, nil, "failed to get latest accessor", ucerr.Wrap(err)
		}
	}

	// Prepare a transformer map for the query to be returned
	defaultTransformer, err := s.GetLatestTransformer(ctx, policy.TransformerPassthrough.ID)
	if err != nil {
		return sqlshim.Passthrough, nil, "failed to get latest transformer", ucerr.Wrap(err)
	}
	transformerMap := newTransformersByLowercase(defaultTransformer)

	columnMap := columnsByLowercase{}
	columnAccessPolicyRIDs := make([]userstore.ResourceID, 0, len(accessor.ColumnIDs))
	for i, cID := range accessor.ColumnIDs {
		col := cm.GetColumnByID(cID)
		if col == nil {
			return sqlshim.Passthrough, nil, "column not found", ucerr.Friendlyf(nil, "column not found: %s", cID)
		}
		transformerID := accessor.TransformerIDs[i]
		tokenAccessPolicyID := accessor.TokenAccessPolicyIDs[i]
		if transformerID.IsNil() {
			transformerID = col.DefaultTransformerID
			tokenAccessPolicyID = col.DefaultTokenAccessPolicyID
		}
		transformer, err := s.GetLatestTransformer(ctx, transformerID)
		if err != nil {
			return sqlshim.Passthrough, nil, "failed to get latest transformer", ucerr.Wrap(err)
		}
		fullColName := col.FullName()
		columnMap.put(fullColName, col)
		transformerMap.put(fullColName, transformer, tokenAccessPolicyID)

		if col.AccessPolicyID != policy.AccessPolicyAllowAll.ID {
			columnAccessPolicyRIDs = append(columnAccessPolicyRIDs, userstore.ResourceID{ID: col.AccessPolicyID})
		}
	}

	// Prepare transformer info for the query to be returned, expanding star columns
	ctis := columnTransformInfoByLowercase{}
	for _, c := range query.Columns {
		if c.Name != "*" {
			queryColumnName := c.String()
			storageColumn, _ := columnMap.get(queryColumnName)
			transformerAndAPID := transformerMap.get(ctx, queryColumnName)
			ctis.put(
				c.Name,
				&columnTransformInfo{
					column:              *storageColumn,
					table:               c.Table,
					transformer:         transformerAndAPID.transformer,
					tokenAccessPolicyID: transformerAndAPID.tokenAccessPolicyID,
				},
			)
		} else {
			for _, tableColumn := range cm.GetColumnsByTable(c.Table) {
				transformerAndAPID := transformerMap.get(ctx, tableColumn.FullName())
				ctis.put(
					tableColumn.Name,
					&columnTransformInfo{
						column:              tableColumn,
						table:               tableColumn.Table,
						transformer:         transformerAndAPID.transformer,
						tokenAccessPolicyID: transformerAndAPID.tokenAccessPolicyID,
					},
				)
			}
		}
	}

	globalAP, accessorAP, thresholdAP, err :=
		s.GetAccessPolicies(
			ctx,
			h.ts.ID,
			policy.AccessPolicyGlobalAccessorID,
			accessor.AccessPolicyID,
		)
	if err != nil {
		return sqlshim.Passthrough, nil, "failed to get access policies", ucerr.Wrap(err)
	}

	allowed, err := thresholdAP.CheckRateThreshold(ctx, s, apContext, accessor.ID)
	if err != nil {
		return sqlshim.Passthrough, nil, "failed to check rate threshold", ucerr.Wrap(err)
	}

	clientAP := &policy.AccessPolicy{
		PolicyType: policy.PolicyTypeCompositeAnd,
		Components: []policy.AccessPolicyComponent{{Policy: &userstore.ResourceID{ID: globalAP.ID}},
			{Policy: &userstore.ResourceID{ID: accessorAP.ID}},
		},
	}
	if !accessor.AreColumnAccessPoliciesOverridden {
		for _, accessPolicyRID := range columnAccessPolicyRIDs {
			clientAP.Components = append(clientAP.Components, policy.AccessPolicyComponent{Policy: &accessPolicyRID})
		}
	}

	te := tokenizer.NewTransformerExecutor(s, h.azc)
	logUnhandledQuery = false
	ti := transformInfo{
		s:                   s,
		dtm:                 dtm,
		accessor:            accessor,
		startTime:           startTime,
		queryType:           query.Type,
		ctis:                ctis,
		accessPolicy:        clientAP,
		apContext:           apContext,
		transformerExecutor: &te,
		maxResults:          thresholdAP.GetResultThreshold(),
	}

	if !allowed {
		return sqlshim.AccessDenied, ti, "final access policy denied", nil
	}

	return sqlshim.TransformResponse, ti, "", nil
}

// TransformSummary records the summary from a transformed query response
func (h *IdpSQLQueryHandler) TransformSummary(ctx context.Context, t any, numSelectorRows, numReturned, numDenied int) {
	ti := t.(transformInfo)

	// Log the query to the audit log
	aei := newAccessorExecutor(
		ctx,
		ti.s,
		nil,
		idp.ExecuteAccessorRequest{AccessorID: ti.accessor.ID},
		ti.startTime,
		false,
		ti.accessor,
	)
	aei.numSelectorRows = numSelectorRows
	aei.numReturned = numReturned
	aei.numDenied = numDenied
	aei.succeeded = true
	aei.apContext = ti.apContext

	auditlog.PostMultipleAsync(ctx, aei.auditLogInfo())
}

// TransformDataRow transforms a data row from the sqlshim proxy
func (h *IdpSQLQueryHandler) TransformDataRow(
	ctx context.Context,
	colNames []string,
	values [][]byte,
	t any,
	cumulativeRows int,
) (bool, error) {
	ti := t.(transformInfo)

	if !ti.shouldTransform() {
		return false, nil
	}

	if ti.maxResults > 0 && cumulativeRows >= ti.maxResults {
		return false, nil
	}

	canTransform, apContext, transformableValues, transformerParams := ti.prepareRow(ctx, colNames, values)
	if !canTransform {
		return true, nil
	}

	allowed, _, err := tokenizer.ExecuteAccessPolicy(ctx, ti.accessPolicy, *apContext, h.azc, ti.s)
	if err != nil {
		return false, ucerr.Wrap(err)
	}
	if !allowed {
		return false, nil
	}

	// Make any necessary transformations to the query results
	var transformedValues []string
	if len(transformerParams) > 0 {
		var err error
		transformedValues, _, err = ti.transformerExecutor.Execute(ctx, transformerParams...)
		if err != nil {
			return false, ucerr.Wrap(err)
		}
	}

	for i, tv := range transformableValues {
		value, err := tv.getValue(ctx, transformedValues)
		if err != nil {
			return false, ucerr.Wrap(err)
		}
		if value != nil {
			s, ok := value.(string)
			if !ok {
				return false, ucerr.Friendlyf(nil, "transformed value '%v' is of type '%T', not string", value, value)
			}
			values[i] = []byte(s)
		} else {
			values[i] = nil
		}
	}

	return true, nil
}

func logPassthroughQuery(ctx context.Context, queryString string, keyValuePairs map[string]any, reason string) {
	entry := auditlog.NewEntryArray(
		auth.GetAuditLogActor(ctx),
		internal.AuditLogEventTypeSqlshimUnhandledQuery,
		auditlog.Payload{
			"Query":         commentRegex.ReplaceAllString(queryString, ""),
			"KeyValuePairs": keyValuePairs,
			"Reason":        reason,
		},
	)
	auditlog.PostMultipleAsync(ctx, entry)
}

func accessorSignatureMatches(accessor *storage.Accessor, columns []sqlparse.Column, selector string, cm *storage.ColumnManager) bool {
	sqlColumnSet := sqlparse.NewColumnSet(columns...)

	if accessor.SelectorConfig.WhereClause == selector {
		tenantColumnSet := sqlparse.NewColumnSet()
		for _, cID := range accessor.ColumnIDs {
			tenantColumn := cm.GetColumnByID(cID)
			if tenantColumn == nil {
				return false
			}
			tenantColumnSet.Insert(sqlparse.Column{
				Table: tenantColumn.Table,
				Name:  tenantColumn.Name,
			})
		}
		if tenantColumnSet.Equal(sqlColumnSet) {
			return true
		}
	}
	return false
}

const schemaUpdateInterval = 6 * time.Hour
const schemaUpdateRetryInterval = 30 * time.Minute

// NotifySchemaSelected handles when a schema has been selected
func (h *IdpSQLQueryHandler) NotifySchemaSelected(ctx context.Context, schema string) {
	database, err := h.s.GetSQLShimDatabase(ctx, h.databaseID)
	if err != nil {
		uclog.Errorf(ctx, "failed to get database: %v", err)
		return
	}

	doUpdate := false
	if slices.Contains(database.Schemas, schema) {
		now := time.Now().UTC()
		doUpdate = now.Sub(database.SchemasUpdated) > schemaUpdateInterval &&
			now.Sub(database.SchemasUpdateScheduled) > schemaUpdateRetryInterval
	} else {
		doUpdate = true
		database.Schemas = append(database.Schemas, schema)
	}

	if doUpdate {
		database.SchemasUpdateScheduled = time.Now().UTC()
		if err := h.s.SaveSQLShimDatabase(ctx, database); err != nil {
			uclog.Errorf(ctx, "failed to save database: %v", err)
		}

		if h.wc != nil {
			if err := h.wc.Send(ctx, worker.IngestSqlshimDatabaseSchemasMessage(h.ts.ID, h.databaseID)); err != nil {
				uclog.Errorf(ctx, "failed to send message to worker: %v", err)
			}
		} else if err := helpers.IngestSqlshimDatabaseSchemas(ctx, h.ts, h.databaseID); err != nil {
			uclog.Errorf(ctx, "failed to ingest sqlshim database schemas for db %d: %v", h.databaseID, err)
		}
	}
}
