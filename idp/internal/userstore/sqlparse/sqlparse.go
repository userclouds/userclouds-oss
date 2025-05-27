package sqlparse

import (
	"fmt"
	"sort"
	"strings"

	pg_query "github.com/pganalyze/pg_query_go/v6"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
)

// QueryType represents the type of a query
type QueryType string

const (
	// QueryTypeSelect represents a SELECT query
	QueryTypeSelect QueryType = "SELECT"

	// QueryTypeInsert represents an INSERT query
	QueryTypeInsert QueryType = "INSERT"

	// QueryTypeUpdate represents an UPDATE query
	QueryTypeUpdate QueryType = "UPDATE"

	// QueryTypeDelete represents a DELETE query
	QueryTypeDelete QueryType = "DELETE"
)

// Query represents a parsed SQL query
type Query struct {
	Type     QueryType
	Columns  []Column
	Selector string
}

// Column represents a column in a table
type Column struct {
	Table string
	Name  string
}

// String returns the string representation of a column
// keep this in sync with idp/internal/storage/models.go:Column.FullName()
func (c Column) String() string {
	if c.Table == "" {
		return c.Name
	}
	return c.Table + "." + c.Name
}

func colSorter(cols []Column) {
	sort.Slice(cols, func(x, y int) bool {
		return cols[x].String() < cols[y].String()
	})
}

// NewColumnSet creates a new set of columns
func NewColumnSet(cols ...Column) set.Set[Column] {
	cs := set.New(colSorter, cols...)
	return cs
}

// ParseQuery parses a SQL query string and returns the table name, column names, and selector (in the convention of userstore)
func ParseQuery(queryString string) (*Query, error) {
	queryString = strings.TrimSpace(queryString)
	tree, err := pg_query.Parse(queryString)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	if len(tree.Stmts) != 1 {
		return nil, ucerr.Friendlyf(nil, "expected exactly one statement, got %d", len(tree.Stmts))
	}

	if selectStmt := tree.Stmts[0].Stmt.GetSelectStmt(); selectStmt != nil {
		return parseSelectStmt(selectStmt)
	}

	if updateStmt := tree.Stmts[0].Stmt.GetUpdateStmt(); updateStmt != nil {
		return parseUpdateStmt(updateStmt)
	}

	if insertStmt := tree.Stmts[0].Stmt.GetInsertStmt(); insertStmt != nil {
		return parseInsertStmt(insertStmt)
	}

	if deleteStmt := tree.Stmts[0].Stmt.GetDeleteStmt(); deleteStmt != nil {
		return parseDeleteStmt(deleteStmt)
	}

	return nil, ucerr.Friendlyf(nil, "unsupported query type")
}

func parseSelectStmt(selectStmt *pg_query.SelectStmt) (*Query, error) {
	fromClause := selectStmt.GetFromClause()
	if len(fromClause) != 1 {
		return nil, ucerr.Friendlyf(nil, "can only handle Select statements with exactly one FROM clause")
	}

	table := ""
	tables := []string{}
	if rangeVar := fromClause[0].GetRangeVar(); rangeVar != nil {
		table = rangeVar.GetRelname()
	}

	if joinExpr := fromClause[0].GetJoinExpr(); joinExpr != nil {
		if lJoin := joinExpr.GetLarg(); lJoin != nil {
			if rangeVar := lJoin.GetRangeVar(); rangeVar != nil {
				tables = append(tables, rangeVar.GetRelname())
			}
		}
		if rJoin := joinExpr.GetRarg(); rJoin != nil {
			if rangeVar := rJoin.GetRangeVar(); rangeVar != nil {
				tables = append(tables, rangeVar.GetRelname())
			}
		}

	}

	targetList := selectStmt.GetTargetList()
	if targetList == nil {
		return nil, ucerr.Friendlyf(nil, "no target columns found in query")
	}

	columns := []Column{}
	for _, target := range targetList {
		col, err := getColumnFromNode(target.GetResTarget().GetVal())
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		if col.Table == "" {
			if table == "" {
				if col.Name == "*" && len(tables) == 2 {
					for _, t := range tables {
						columns = append(columns, Column{Table: t, Name: "*"})
					}
					continue
				} else {
					return nil, ucerr.Friendlyf(nil, "table not able to be determined for column: %s", col.Name)
				}
			}
			col.Table = table
		}
		columns = append(columns, *col)
	}

	selector := "{id} = ANY(?)" // default selector for no where clause
	if whereClause := selectStmt.GetWhereClause(); whereClause != nil {
		var err error
		selector, err = rewriteWhereClauseAsSelector(whereClause, table)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	}

	return &Query{
		Type:     QueryTypeSelect,
		Columns:  columns,
		Selector: selector,
	}, nil
}

func parseUpdateStmt(updateStmt *pg_query.UpdateStmt) (*Query, error) {
	_ = updateStmt
	return &Query{
		Type: QueryTypeUpdate,
	}, nil
}

func parseInsertStmt(insertStmt *pg_query.InsertStmt) (*Query, error) {
	_ = insertStmt
	return &Query{
		Type: QueryTypeInsert,
	}, nil
}

func parseDeleteStmt(deleteStmt *pg_query.DeleteStmt) (*Query, error) {
	_ = deleteStmt
	return &Query{
		Type: QueryTypeDelete,
	}, nil
}

func rewriteWhereClauseAsSelector(whereClause *pg_query.Node, defaultTable string) (string, error) {

	if b := whereClause.GetBoolExpr(); b != nil {
		var operator string
		switch b.Boolop {
		case pg_query.BoolExprType_AND_EXPR:
			operator = "AND"
		case pg_query.BoolExprType_OR_EXPR:
			operator = "OR"
		case pg_query.BoolExprType_NOT_EXPR:
			operator = "!"
		}

		boolParts := []string{}
		for _, arg := range b.Args {
			selector, err := rewriteWhereClauseAsSelector(arg, defaultTable)
			if err != nil {
				return "", ucerr.Wrap(err)
			}
			boolParts = append(boolParts, selector)
		}

		if len(boolParts) == 1 {
			return fmt.Sprintf("%s %s", operator, boolParts[0]), nil
		}
		return "(" + strings.Join(boolParts, " "+operator+" ") + ")", nil

	} else if a := whereClause.GetAExpr(); a != nil {
		name := a.Name[0].GetString_().Sval
		switch name {
		case "<>":
			name = "!="
		case "~~":
			name = "LIKE"
		case "!~~":
			name = "NOT LIKE"
		case "~~*":
			name = "ILIKE"
		case "!~~*":
			name = "NOT ILIKE"
		}
		var lStr, rStr string
		lColumn, err := getColumnFromNode(a.Lexpr)
		if err != nil {
			lStr = "?"
		} else {
			parts := strings.Split(lColumn.String(), ".")
			lStr = "{" + parts[len(parts)-1] + "}"
		}
		rColumn, err := getColumnFromNode(a.Rexpr)
		if err != nil {
			rStr = "?"
		} else {
			parts := strings.Split(rColumn.String(), ".")
			rStr = "{" + parts[len(parts)-1] + "}"
		}

		return fmt.Sprintf("%s %s %s", lStr, name, rStr), nil
	} else if c := whereClause.GetCaseExpr(); c != nil {
		return "", ucerr.Friendlyf(nil, "case statements not supported")
	} else if n := whereClause.GetNullTest(); n != nil {
		var str string
		column, err := getColumnFromNode(n.Arg)
		if err != nil {
			str = "?"
		} else {
			parts := strings.Split(column.String(), ".")
			str = "{" + parts[len(parts)-1] + "}"
		}
		return fmt.Sprintf("%s IS NULL", str), nil
	}

	return "", ucerr.Friendlyf(nil, "unhandled where clause type")
}

func getColumnFromNode(node *pg_query.Node) (*Column, error) {
	colRef := node.GetColumnRef()
	if colRef == nil {
		return nil, ucerr.Friendlyf(nil, "expected column reference")
	}
	fields := colRef.GetFields()
	if fields == nil {
		return nil, ucerr.Friendlyf(nil, "expected fields in column reference")
	}

	column := &Column{}

	if columnVal := fields[len(fields)-1].GetString_(); columnVal != nil {
		column.Name = columnVal.Sval
	} else if fields[len(fields)-1].GetAStar() != nil {
		column.Name = "*"
	} else {
		return nil, ucerr.Friendlyf(nil, "expected string or * in column reference")
	}

	if len(fields) > 1 {
		if tableVal := fields[len(fields)-2].GetString_(); tableVal != nil {
			column.Table = tableVal.Sval
		} else {
			return nil, ucerr.Friendlyf(nil, "expected string in column reference")
		}
	}

	return column, nil
}
