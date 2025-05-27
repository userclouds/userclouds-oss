package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"userclouds.com/internal/companyconfig"
	"userclouds.com/internal/logdb"
	"userclouds.com/internal/tenantdb"
)

// SQLAnalyzer checks things that look like SQL statements for deleted clauses
// TODO: bonus points for someone to make this model-aware and not complain if the model itself doesn't have eg. Deleted
var SQLAnalyzer = &analysis.Analyzer{
	Name:     "sql",
	Doc:      "Checks SQL SELECT statements to make sure they check deleted. Suggest /* lint-deleted */ to bypass if needed.",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runSQL,
}

func runSQL(pass *analysis.Pass) (any, error) {
	// don't bother to run this on samples, it's annoying
	if strings.Contains(pass.Pkg.Path(), "samples.userclouds") ||
		strings.Contains(pass.Pkg.Path(), "userclouds/samples") {
		return nil, nil
	}

	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	nodeFilter := []ast.Node{
		(*ast.BasicLit)(nil),
		(*ast.ValueSpec)(nil),
		(*ast.AssignStmt)(nil),
	}

	var mu sync.Mutex
	queries := make(map[token.Pos]string)

	insp.Preorder(nodeFilter, func(n ast.Node) {
		// don't need to validate these in test files
		fn := pass.Fset.Position(n.Pos()).Filename
		if strings.HasSuffix(fn, "_test.go") {
			return
		}

		if s, ok := n.(*ast.BasicLit); ok {
			checkLiteral(pass, s)
		}

		if v, ok := n.(*ast.ValueSpec); ok {
			checkValue(pass, v, queries, &mu)
		}

		if a, ok := n.(*ast.AssignStmt); ok {
			checkAssign(pass, a)
		}
	})

	nodeFilter = []ast.Node{
		(*ast.CallExpr)(nil),
	}

	insp.Preorder(nodeFilter, func(n ast.Node) {
		if c, ok := n.(*ast.CallExpr); ok {
			checkCall(pass, c, queries)
		}
	})

	return nil, nil
}

var sqlValuesRegex = regexp.MustCompile(`.*\((.*)\)[^\(]*values[^\(]*\((.*)\)[^\)]*$`)
var sqlParamsRegex = regexp.MustCompile(`\$[0-9]+`)

var usedColumns = map[string][]string{}

func init() {
	for _, m := range []map[string][]string{
		logdb.UsedColumns,
		tenantdb.UsedColumns,
		companyconfig.UsedColumns,
	} {
		for k, v := range m {
			if t, ok := usedColumns[k]; ok {
				if !reflect.DeepEqual(t, v) {
					// this is fragile but I think will be ok for a while ...
					panic(fmt.Sprintf("tools/analyzer/sql doesn't currently support conflicting table names (%s) across DB schemas", k))
				}
			}
			usedColumns[k] = v
		}
	}
}

// make sure that SQL SELECTs have deleted clauses and don't repeat $assignments
func checkLiteral(pass *analysis.Pass, s *ast.BasicLit) {
	if s.Kind != token.STRING {
		return
	}

	st, err := strconv.Unquote(s.Value)
	if err != nil {
		pass.Reportf(s.Pos(), "error unquoting string literal")
		return
	}

	v := strings.ReplaceAll(strings.TrimSpace(strings.ToLower(st)), "\n", " ")

	// we only currently understand select / insert / upsert
	if !(strings.HasPrefix(v, "select ") || strings.HasPrefix(v, "insert ") || strings.HasPrefix(v, "upsert ")) {
		return
	}

	// extra safety to get rid of things like fmt.Errorf("select foo failed")
	if !strings.Contains(v, " from") && !strings.Contains(v, " into") {
		return
	}

	fn := pass.Fset.File(s.Pos()).Name()
	if strings.Contains(v, "select *") && !strings.Contains(fn, "_test.go") && !strings.Contains(v, "select-star-ok") {
		pass.Reportf(s.Pos(), "avoid using SELECT * because it will fail during a deploy if a column is added")
		return
	}

	// check column usage
	checkColumns(pass, v, s.ValuePos)

	// these checks only make sense on insert/upsert statements
	if strings.HasPrefix(v, "insert") || strings.HasPrefix(v, "upsert") {
		if !strings.Contains(v, "no-match-cols-vals") {
			// does len(cols) == len(vals)?
			// note that the not-paren groups before and after value are to ensure we don't capture eg. NOW() incorrectly
			matches := sqlValuesRegex.FindStringSubmatch(v)
			if len(matches) == 3 {
				if cols, vals := len(strings.Split(matches[1], ",")), len(strings.Split(matches[2], ",")); cols != vals {
					pass.Reportf(s.ValuePos, "SQL insert/upsert doesn't have matching counts for columns (%d) and values (%d)", cols, vals)
				}
			} else if !strings.Contains(v, "lint-sql-ok") {
				pass.Reportf(s.ValuePos, "SQL query doesn't match expected INSERT/UPSERT regex, please update the linter: %v", matches)
			}
		}

		if !strings.Contains(v, "allow-multiple-target-use") {
			used := make(map[string]struct{})
			matches := sqlParamsRegex.FindAllString(v, -1)
			for _, m := range matches {
				if _, ok := used[m]; ok {
					pass.Reportf(s.Pos(), "repeated use of %s in single query", m)
				}
				used[m] = struct{}{}
			}
		}
	}

	// everything should have deleted in it
	if !strings.Contains(v, "deleted") {
		pass.Reportf(s.Pos(), "apparent SQL statement missing deleted clause")
		return
	}

	// and everything that checks deleted=0 should include a timestamp cast
	noSpaces := strings.ReplaceAll(v, " ", "") // this is a bit hacky but we want to catch `deleted = '...'` etc
	if strings.Contains(noSpaces, "deleted=0") && !strings.Contains(noSpaces, "deleted=0::timestamp") {
		pass.Reportf(s.Pos(), "apparent SQL statement with deleted cause missing timestamp cast")
	}
}

// make sure that SQL queries are constants
func checkValue(pass *analysis.Pass, v *ast.ValueSpec, queries map[token.Pos]string, mu *sync.Mutex) {
	for i, val := range v.Values {
		if !literalIsSQL(pass, val) {
			continue
		}

		if v.Names[i].Obj.Kind != ast.Con {
			pass.Reportf(v.Pos(), "SQL query appears to have non-constant type")
		}

		// we need to save the queries here instead of in checkLiteral because the positions won't match with the args
		query, ok := getLiteral(pass, val)
		if !ok {
			panic("got literal above in literalIsSQL but can't get it here?")
		}
		mu.Lock()
		queries[v.Names[i].Obj.Pos()] = query
		mu.Unlock()
	}
}

// make sure that SQL queries are constants
func checkAssign(pass *analysis.Pass, v *ast.AssignStmt) {
	for i, val := range v.Rhs {
		if !literalIsSQL(pass, val) {
			continue
		}

		e, ok := v.Lhs[i].(*ast.Ident)
		if !ok {
			continue
		}

		if e.Obj.Kind != ast.Con {
			pass.Reportf(v.Pos(), "SQL query appears to have non-constant type")
		}
	}
}

func checkCall(pass *analysis.Pass, c *ast.CallExpr, queries map[token.Pos]string) {
	sel, ok := c.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	// Note for future self if we want to do stricter type checking on the call sites
	// to avoid whitelisting a set of SQL operations
	// the most frequent pattern would be
	// c.Fun is a SelectorExpr (sel)
	// sel is another SelectorExpr because we frequently use m.db.Select*
	// sel.X is an Ident, and sel.X.Obj.Decl is a Field (f)
	// f.Type is a StarExpr (s)
	// s.X is an Ident, and s.X.Name == "Manager
	// likewise sel.Sel is the "db" equivalent, which just seems to be an Ident without type info

	op := sel.Sel
	if !isSQLOp(op.Name) {
		return
	}

	// second or third arg should be query
	var i int
	if strings.HasPrefix(op.Name, "Get") || strings.HasPrefix(op.Name, "Select") {
		i = 2
	} else {
		i = 1
	}

	argIdent, ok := c.Args[i].(*ast.Ident)
	if !ok {
		// This occurs currently in migratedb where we're loading the query from an object (vs a constant)
		return
	}

	if argIdent.Obj == nil {
		pass.Reportf(c.Pos(), "this should never happen but I don't know why it is :)")
		return
	}

	// this happens when we have non-constant queries ... we can't check them, which is why we discourge them
	// TODO: would be nice to fix this to be a little safer or alert somehow?
	query, ok := queries[argIdent.Obj.Pos()]
	if !ok {
		return
	}

	// count $Xs vs the remaining variadic args in the call
	matches := sqlParamsRegex.FindAllString(query, -1)

	if !strings.HasPrefix(query, "select") && !strings.Contains(query, "allow-multiple-target-use") {
		if expected := len(c.Args) - i - 1; len(matches) != expected {
			pass.Reportf(c.Pos(), "number of $targets (%d) doesn't match number of params to SQL call (%d)", len(matches), expected)
		}
	}
}

func isSQLOp(ident string) bool {
	ops := []string{
		"GetContext",
		"SelectContext",
		"ExecContext",
		"QueryxContext",
	}

	return slices.Contains(ops, ident)
}

func getLiteral(pass *analysis.Pass, val ast.Expr) (string, bool) {
	l, ok := val.(*ast.BasicLit)
	if !ok {
		return "", false
	}

	if l.Kind != token.STRING {
		return "", false
	}

	st, err := strconv.Unquote(l.Value)
	if err != nil {
		pass.Reportf(val.Pos(), "error unquoting string literal")
		return "", false
	}

	return strings.TrimSpace(strings.ToLower(st)), true
}

func literalIsSQL(pass *analysis.Pass, val ast.Expr) bool {
	s, ok := getLiteral(pass, val)
	if !ok {
		return false
	}

	// TODO there must be a better way to classify this as a SQL query or not
	return (strings.HasPrefix(s, "select ") && strings.Contains(s, "from")) ||
		(strings.HasPrefix(s, "update ") && strings.Contains(s, " set ")) ||
		// NB: these aren't collapsed so we don't trigger the linter itself
		strings.HasPrefix(s, "upsert ") && strings.Contains(s, " into ") ||
		strings.HasPrefix(s, "insert ") && strings.Contains(s, " into ") ||
		strings.HasPrefix(s, "delete from")
}

var keywords = []string{
	"select",
	"insert",
	"upsert",
	"into",
	"from",
	"order",
	"by",
	"asc",
	"desc",
	"where",
	"timestamp",
	"and",
	"or",
	"count",
	"max",
}

var delimeters = "(), =/*:';"

// Column List RegExp
var clre = regexp.MustCompile(`select\s+(.*?)\s+from`)

func checkColumns(pass *analysis.Pass, v string, pos token.Pos) {
	// bypass this check for places we query the system tables like mysql.Users
	if strings.Contains(v, "lint-system-table") || strings.Contains(v, "bypass-known-table-check") {
		return
	}

	// to avoid parsing the query we're going to tokenize and keep a list
	// of known SQL keywords, anything else should be table name or columns. Right? :)
	ff := func(r rune) bool {
		return strings.Contains(delimeters, string(r))
	}
	tks := strings.FieldsFunc(v, ff)

	// first figure out which table we're using
	// we're making an assumption that set(table_names) !intersect set(column_names)
	// and/or that table name happens earlier in the query
	var tbl string
	for _, tk := range tks {
		for k := range usedColumns {
			if k == tk {
				// this is once again slightly fragile but should hold us for a while
				// we'll ultimately need to pull in a full SQL query parser :/
				if tbl != "" && tbl != k {
					pass.Reportf(pos, "found query with 2+ table names (%s and %s)", tbl, k)
					return
				}
				tbl = k
			}
		}
	}

	if tbl == "" {
		pass.Reportf(pos, "couldn't find known table name for query")
		return
	}

	// is there an alias for this table? used later for column safety
	var alias string
	tblAliasRE := regexp.MustCompile(fmt.Sprintf(`%s\s+as\s+([a-zA-Z]+)\s`, tbl))

	if tblAliasRE.Match([]byte(v)) {
		aliases := tblAliasRE.FindStringSubmatch(v)

		// more than one alias for the table name will require more code
		// aliases[0] is the full match, aliases[1] is the first capture group
		if len(aliases) != 2 {
			pass.Reportf(pos, "found query with multiple aliases for table name (%s - %v) ... congrats, you get to update a linter ", tbl, aliases)
			return
		}
		alias = aliases[1]
	}

	// allow bypassing safe column checks
	if strings.Contains(v, "lint-sql-unsafe-columns") {
		return
	}

	// now check we don't have unknown columns ...
	for _, tk := range tks {
		var found bool

		// if this is the table name, we're good
		if tk == tbl {
			found = true
			break
		}

		// check through all our SQL reserved keywords
		if slices.Contains(keywords, tk) {
			found = true
		}

		// if the columns are selected by table alias, we can live with that and just call it the column name
		if strings.Contains(tk, fmt.Sprintf("%s.", alias)) {
			tk = strings.Split(tk, ".")[1]
		}

		// and check through our known columns for this table
		if slices.Contains(usedColumns[tbl], tk) {
			found = true
		}

		// $1 etc are all legal for query replacement
		if strings.HasPrefix(tk, "$") {
			found = true
			break
		}

		if found {
			continue
		}
		pass.Reportf(pos, "apparent SQL statement for table %s contains unsafe column %s", tbl, tk)
	}

	// this last part only applies to select queries, since we want to make sure our field lists are exhaustive
	if !strings.Contains(v, "select ") {
		return
	}

	// we also don't care to keep this up to date in tests, assuming they'll fail in other ways in the rare case
	// that we handwrite SQL queries in them
	if fn := pass.Fset.File(pos).Name(); strings.HasSuffix(fn, "_test.go") {
		return
	}

	// we also can skip this check with a lint statement, currently only needed for the shadow objects in authz
	if strings.Contains(v, "lint-sql-select-partial-columns") {
		return
	}

	colsList := clre.FindAllStringSubmatch(v, -1)
	if len(colsList) == 0 {
		pass.Reportf(pos, "couldn't find columns list for query %v", v)
		return
	}

	for _, matches := range colsList {
		cols := matches[1] // [0] is the full regex match, [1] is the first (and only) group operator in it

		// if we're using an aggregate function, we can't check the columns well, so either fail, or
		// allow `id, aggregate_function(col)` as a common case
		// TODO we should really check that this case is part of a subquery or JOIN, but not yet
		foundCols := strings.Split(cols, ",")
		var agg bool
		for _, tk := range foundCols {
			tk = strings.TrimSpace(tk)
			if strings.HasPrefix(tk, "max") || strings.HasPrefix(tk, "count") {
				agg = true
				if strings.Contains(v, "lint-sql-allow-multi-column-aggregation") {
					break
				}
				if len(foundCols) > 2 {
					pass.Reportf(pos, "we don't support multiple columns and aggregate functions")
					break
				}
				if len(foundCols) == 2 && foundCols[0] != "id" && foundCols[0] != "name" {
					pass.Reportf(pos, "we only support id, aggregate_function(col) or name, aggregate_function(col) as a valid query")
					break
				}
			}
		}

		// if we're using an aggregate function, but didn't fail above, we're (probably) good
		if agg {
			continue
		}

		if strings.Contains(strings.ToLower(cols), " as ") {
			pass.Reportf(pos, "we don't support AS aliasing in queries today (it will break sqlx common path)")
			continue
		}

		// and that we included all the known ones, specifically so that we don't forget to update
		// handwritten queries when we add a column (since we no longer use SELECT * to make migrations safe)
		for _, k := range usedColumns[tbl] {
			var found bool
			for _, tk := range foundCols {
				// if the columns are selected by table alias, we can live with that
				if strings.Contains(tk, fmt.Sprintf("%s.", alias)) {
					tk = strings.Split(tk, ".")[1]
				}

				if strings.TrimSpace(tk) == strings.TrimSpace(k) {
					found = true
					break
				}
			}

			if found {
				continue
			}
			pass.Reportf(pos, "apparent SQL statement for table %s is missing column %s", tbl, k)
		}
	}
}
