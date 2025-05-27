package storage

import (
	"context"
	"regexp"
	"strings"

	"userclouds.com/idp/userstore"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

const maxSearchTermLength = 10

// SearchParameters describes the parameters used for a user lookup using the search index
type SearchParameters struct {
	Column    *Column
	Operator  string
	Term      string
	TermFull  string
	Truncated bool
}

// TODO: for now, we only support ILIKE queries since the index is lowercased
// var searchSelectorRegexp = regexp.MustCompile(`{([a-zA-Z0-9_-]+)\} (LIKE|like|ILIKE|ilike) (.+)`)
var searchSelectorRegexp = regexp.MustCompile(`{([a-zA-Z0-9_-]+)\} (ILIKE|ilike) (.+)`)

var searchTermUnsupportedCharRegexp = regexp.MustCompile(`[\\\*?]`)
var searchTermWildcardRegexp = regexp.MustCompile(`[%]`)

type searchParametersBuilder struct {
	cm                   *ColumnManager
	requireSearchIndexed bool
	selectorParts        [][]string
	selectorValues       userstore.UserSelectorValues
}

func newSearchParametersBuilder(
	cm *ColumnManager,
	requireSearchIndexed bool,
	selectorConfig userstore.UserSelectorConfig,
	selectorValues ...any,
) searchParametersBuilder {
	return searchParametersBuilder{
		cm:                   cm,
		requireSearchIndexed: requireSearchIndexed,
		selectorParts:        searchSelectorRegexp.FindAllStringSubmatch(selectorConfig.WhereClause, -1),
		selectorValues:       selectorValues,
	}
}

func (spb searchParametersBuilder) convertTerm(ctx context.Context, term string) (string, string, bool) {
	if term == "?" {
		uclog.Debugf(ctx, "term is a placeholder (?)")
		return term, term, false
	}

	// we do not allow some characters in the unmodified search term
	// we require a trailing %, which will be converted to * for search
	// we optionally allow a leading %, which will also be converted to *
	// no other % may be present in the unmodified search term

	// if the term contains any of the unsupported characters (\, *, ?), we will not allow it
	if searchTermUnsupportedCharRegexp.MatchString(term) {
		uclog.Debugf(ctx, "term contains unsupported characters.")
		return "", "", false
	}

	// TODO: for now, we will only support search queries that have wildcard matches
	//       for both the prefix and suffix
	if !strings.HasPrefix(term, "%") || !strings.HasSuffix(term, "%") {
		uclog.Debugf(ctx, "term does not have wildcard matches.")
		return "", "", false
	}

	term = strings.TrimPrefix(term, "%")
	term = strings.TrimSuffix(term, "%")

	if searchTermWildcardRegexp.MatchString(term) {
		uclog.Debugf(ctx, "term contains wildcard matches.")
		return "", "", false
	}

	// Truncate the term to max characters supported by the search index
	truncated := false
	fullTerm := term
	if len(term) > maxSearchTermLength {
		term = term[:maxSearchTermLength]
		truncated = true
	}
	return term, fullTerm, truncated
}

func (spb searchParametersBuilder) extractTerm(applySelectorValues bool) (string, error) {
	term := strings.TrimSpace(spb.selectorParts[0][3])
	if term == "?" {
		if applySelectorValues {
			if len(spb.selectorValues) != 1 {
				return "",
					ucerr.Friendlyf(
						nil,
						"number of UserSelectorValues (%d) != number of parameters in WhereClause (1)",
						len(spb.selectorValues),
					)
			}

			s, ok := spb.selectorValues[0].(string)
			if !ok {
				return "",
					ucerr.Friendlyf(
						nil,
						"selector argument '%v' is not a string, it is: '%T'",
						spb.selectorValues[0],
						spb.selectorValues[0],
					)
			}

			term = s
		}
	} else if applySelectorValues {
		if len(spb.selectorValues) != 0 {
			return "",
				ucerr.Friendlyf(
					nil,
					"number of UserSelectorValues (%d) != number of parameters in WhereClause (0)",
					len(spb.selectorValues),
				)
		}
	}

	return term, nil
}

func (spb searchParametersBuilder) getSearchParameters(ctx context.Context, applySelectorValues bool) (*SearchParameters, error) {
	if len(spb.selectorParts) != 1 || len(spb.selectorParts[0]) != 4 {
		return nil, nil
	}
	colName := spb.selectorParts[0][1]
	col := spb.cm.GetUserColumnByName(colName)
	if col == nil {
		return nil, ucerr.Friendlyf(nil, "'%s' is not a recognized column name", colName)
	}

	if spb.requireSearchIndexed && !col.SearchIndexed {
		uclog.Debugf(ctx, "column '%s' is not indexed for search", colName)
		return nil, nil
	}

	term, err := spb.extractTerm(applySelectorValues)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	uclog.DebugfPII(ctx, "term: %v", term)
	termTrunc, termFull, truncated := spb.convertTerm(ctx, term)
	if termTrunc == "" {
		return nil, nil
	}

	return &SearchParameters{
		Column:    col,
		Operator:  spb.selectorParts[0][2],
		Term:      termTrunc,
		TermFull:  termFull,
		Truncated: truncated,
	}, nil
}

// GetAdjustedSelectorValues returns an adjusted set of user selector values for a given
// user selector config and user selector values
func GetAdjustedSelectorValues(
	ctx context.Context,
	cm *ColumnManager,
	selectorConfig userstore.UserSelectorConfig,
	selectorValues userstore.UserSelectorValues,
) (userstore.UserSelectorValues, error) {
	spb := newSearchParametersBuilder(cm, false, selectorConfig, selectorValues...)
	sp, err := spb.getSearchParameters(ctx, true)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	if sp != nil && sp.Column.DataTypeID == datatype.Email.ID {
		// NOTE: this assumes search is only supported for ilike selectors with
		//       a single selector term and an infix style query (wildcards before
		//       and after the term
		//
		//       we strip the domain from the selector value if the column has
		//       an email data type, since including the domain adversely affects
		//       the performance of a tri-gram query for the address
		emailParts := strings.Split(sp.TermFull, "@")
		if len(emailParts) == 2 {
			return []any{"%" + emailParts[0] + "@%"}, nil
		}
	}

	return selectorValues, nil
}

// GetExecutableSearchParameters returns valid search parameters with selector values applied
func GetExecutableSearchParameters(
	ctx context.Context,
	cm *ColumnManager,
	selectorConfig userstore.UserSelectorConfig,
	selectorValues userstore.UserSelectorValues,
) (*SearchParameters, error) {
	spb := newSearchParametersBuilder(cm, true, selectorConfig, selectorValues...)
	sp, err := spb.getSearchParameters(ctx, true)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return sp, nil
}

// getSearchParameters returns valid search parameters
func getSearchParameters(ctx context.Context, cm *ColumnManager, selectorConfig userstore.UserSelectorConfig) (*SearchParameters, error) {
	spb := newSearchParametersBuilder(cm, true, selectorConfig)
	sp, err := spb.getSearchParameters(ctx, false)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return sp, nil
}
