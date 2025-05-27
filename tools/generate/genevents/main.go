package genevents

import (
	"context"
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/repopath"
	"userclouds.com/tools/generate"
)

type data struct {
	EventCodes  string
	EventMap    string
	ServiceName string
}

type event struct {
	mapName     string
	eventName   string
	code        string
	service     string // this is a SelectorExpr (service.Tokenizer) rather than a service.Service therefore string
	ignore      bool
	url         string
	typ         string
	subcategory string
}

// this encoding design means that we can never use genevents inside uchttp or jsonapi, which I think is totally fine
// (since we always assume a SelectorExpr to match). The int is the arg number (0-indexed) that is the error string.
var supportedErrorFunctions = map[string]int{
	"uchttp.ErrorL":             4,
	"jsonapi.MarshalErrorL":     3,
	"jsonapi.MarshalJSONErrorL": 3,
	"jsonapi.MarshalSQLErrorL":  3,
	"ucerr.WrapWithName":        1,
}

// Run runs the genevents tool
func Run(ctx context.Context, path string, svcName string) {
	// TODO: more elegant way to do this?
	if svcName == string(service.IDP) {
		svcName = "IDP"
	} else {
		var caser = cases.Title(language.English) // our code is always in english
		svcName = caser.String(svcName)
	}

	svcConst := fmt.Sprintf("service.%s", svcName)
	// TODO
	if svcConst == "service.Authz" {
		svcConst = "service.AuthZ"
	}

	codes, events, maxEventCode := parseExistingEvents(ctx, svcName, svcConst)

	parseDir(ctx, svcName, svcConst, events, path)

	// if there are any codes that are used by events but not defined, define them
	start := len(codes)
	for _, e := range events {
		if _, ok := codes[e.code]; !ok {
			maxEventCode++ // increment before assigning since we kept the max, not first-free
			uclog.Debugf(ctx, "[%v] event %s uses code %s which is not defined, assigning %d", svcName, e.eventName, e.code, maxEventCode)
			codes[e.code] = maxEventCode
		}
	}

	if addedEvents := len(codes) - start; addedEvents > 0 {
		uclog.Infof(ctx, "[%v] had %v events, added %v new events", svcName, start, addedEvents)
	}

	// pre-allocate these arrays since we know their length
	// the initial 0 makes the slice length 0, but backed by an array of len(codes)
	// so that append starts at index 0, but won't ever need to reallocate
	codeLines := make([]string, 0, len(codes))
	for n, v := range codes {
		codeLines = append(codeLines, fmt.Sprintf("%s uclog.EventCode = %d", n, v))
	}

	eventLines := make([]string, 0, len(events))
	for _, e := range events {
		eventLines = append(eventLines, writeEventLine(e))
	}

	// sort for consistent generation
	sort.Strings(codeLines)
	sort.Strings(eventLines)

	d := data{
		EventCodes:  strings.Join(codeLines, "\n"),
		EventMap:    strings.Join(eventLines, "\n"),
		ServiceName: strings.ToLower(svcName),
	}

	fn := filepath.Join(repopath.BaseDir(), fmt.Sprintf("logserver/events/%s_generated.go", d.ServiceName))
	generate.WriteFileIfChanged(ctx, fn, tmpString, d)
}

func parseDir(ctx context.Context, svcName, svcConst string, events map[string]event, base string) {
	// first we parse files in this directory
	_, files, err := generate.LoadDir(ctx, base, 0)
	if err != nil {
		uclog.Fatalf(ctx, "failed to load dir: %v", err)
	}

	var fnName string
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			// TODO: we should really keep track of when we come out of this function and reset fnName,
			// because there's a chance that we could have nested function declarations, but checking n == nil
			// is insufficient because that occurs with all code blocks, not just function declarations

			if fn, ok := n.(*ast.FuncDecl); ok {
				fnName = fn.Name.Name
				parseFunction(fn, svcName, svcConst, events)
			} else if call, ok := n.(*ast.CallExpr); ok {
				parseCall(ctx, svcName, svcConst, fnName, call, events)
			}
			// TODO (sgarrity 7/23): for completeness, we should handle *ast.AssignStmts here too someday
			// (where an anoymous function is named and then linked to a handler path, instead of defined
			// inline in the HandleFunc call)

			return true
		})
	}

	// then we enumerate subdirectories and parse them recursively
	dirs, err := os.ReadDir(base)
	if err != nil {
		uclog.Fatalf(ctx, "failed to read dir: %v", err)
	}

	for _, dir := range dirs {
		if !dir.IsDir() {
			continue
		}
		parseDir(ctx, svcName, svcConst, events, filepath.Join(base, dir.Name()))
	}
}

func checkParamType(t *ast.Field, pkg, name string) bool {
	t1, ok := t.Type.(*ast.SelectorExpr)
	if !ok {
		st, ok := t.Type.(*ast.StarExpr)
		if !ok {
			return false
		}

		t1, ok = st.X.(*ast.SelectorExpr)
		if !ok {
			return false
		}
	}

	x1, ok := t1.X.(*ast.Ident)
	if !ok {
		return false
	}

	if x1.Name != pkg || t1.Sel.Name != name {
		return false
	}

	return true
}

func isHandlerFunction(fn *ast.FuncDecl) bool {
	// min 2, max 4
	if len(fn.Type.Params.List) < 2 || len(fn.Type.Params.List) > 4 {
		return false
	}

	// first param is always http.ResponseWriter
	if !checkParamType(fn.Type.Params.List[0], "http", "ResponseWriter") {
		return false
	}

	// second param is always *http.Request
	if !checkParamType(fn.Type.Params.List[1], "http", "Request") {
		return false
	}

	// if we get 3-4, they must (currently) be uuid.UUID for uchttp
	if len(fn.Type.Params.List) > 2 {
		if !checkParamType(fn.Type.Params.List[2], "uuid", "UUID") {
			return false
		}
	}

	if len(fn.Type.Params.List) > 3 {
		if !checkParamType(fn.Type.Params.List[3], "uuid", "UUID") {
			return false
		}
	}

	return true
}

func generateName(functionName string) string {
	// we want to translate a function name like "GetUser" to "Get User" for the name
	// but not "GetJSONKey" to "Get J S O N Key" (we want "Get JSON Key")
	// we also want "GetIDForUser" to become "Get ID For User" and not "Get Id For User"...
	var name string
	for i, l := range functionName {
		// first letter is always upper, and never a space
		if i == 0 {
			name += string(unicode.ToUpper(l))
			continue
		}

		if unicode.IsUpper(l) {
			// if we're on an upper case letter
			if !unicode.IsUpper(rune(functionName[i-1])) {
				// and the last letter was not upper case, we need to add a space
				name += " "
			} else if i+1 < len(functionName) && !unicode.IsUpper(rune(functionName[i+1])) {
				// or if the next letter is not upper case, we need to add a space
				name += " "
			}
		}
		name += string(l)
	}

	return name
}

func parseFunction(fn *ast.FuncDecl, svcName, svcConst string, events map[string]event) {
	// we're only interested in handler functions, but that can include nested handlers
	if !isHandlerFunction(fn) {
		return
	}

	funcName := strings.TrimSuffix(fn.Name.Name, "Generated")
	name := generateName(funcName)

	countEvent := event{
		mapName:     fmt.Sprintf(`%s.%s-fm.Count`, strings.ToLower(svcName), funcName),
		eventName:   name,
		service:     svcConst,
		url:         "", // we can't auto-detect this yet, so this will need to be filled in manually
		code:        fmt.Sprintf("Event%s%s", svcName, strings.ReplaceAll(name, " ", "")),
		typ:         "uclog.EventCategoryCall",
		subcategory: "function",
	}
	addEvent(events, countEvent)

	durationEvent := event{
		mapName:     fmt.Sprintf(`%s.%s-fm.Duration`, strings.ToLower(svcName), funcName),
		eventName:   name,
		service:     svcConst,
		url:         countEvent.url,
		code:        countEvent.code + "Duration",
		typ:         "uclog.EventCategoryDuration",
		subcategory: "function",
	}
	addEvent(events, durationEvent)
	addDBEvents(funcName, svcName, svcConst, name, events, "DBSelect")
	addDBEvents(funcName, svcName, svcConst, name, events, "DBGet")
	addDBEvents(funcName, svcName, svcConst, name, events, "DBWrite")

}
func addDBEvents(funcName string, svcName, svcConst, name string, events map[string]event, dbOperation string) {
	dbCountEvent := event{
		mapName:     fmt.Sprintf(`%s.%s-fm.%sCount`, strings.ToLower(svcName), funcName, dbOperation),
		eventName:   name,
		service:     svcConst,
		url:         "", // we can't auto-detect this yet, so this will need to be filled in manually
		code:        fmt.Sprintf("Event%s%s%s", svcName, strings.ReplaceAll(name, " ", ""), dbOperation),
		typ:         "uclog.EventCategoryCount",
		subcategory: "db",
	}
	addEvent(events, dbCountEvent)

	dbDurationEvent := event{
		mapName:     fmt.Sprintf(`%s.%s-fm.%sDuration`, strings.ToLower(svcName), funcName, dbOperation),
		eventName:   name,
		service:     svcConst,
		url:         dbCountEvent.url,
		code:        dbCountEvent.code + "Duration",
		typ:         "uclog.EventCategoryDuration",
		subcategory: "db",
	}
	addEvent(events, dbDurationEvent)
}

func parseCall(ctx context.Context, svcName, svcConst, fnName string, call *ast.CallExpr, events map[string]event) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}

	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return
	}

	var found bool
	var fnArgIdx int
	for fn, idx := range supportedErrorFunctions {
		vals := strings.Split(fn, ".")
		if id.Name == vals[0] && sel.Sel.Name == vals[1] {
			found = true
			fnArgIdx = idx
			break
		}
	}
	if !found {
		return
	}

	// the argument number depends on the function that was called, it's the last non-variadic one
	// but easier to encode the index than parse the function signature :)
	str, ok := call.Args[fnArgIdx].(*ast.BasicLit)
	if !ok {
		uclog.Fatalf(ctx, "%s.%s called with last arg that isn't an *ast.BasicLit", id.Name, sel.Sel.Name)
	}
	val := strings.Trim(str.Value, `"`)

	ev := event{
		mapName:     fmt.Sprintf(`%s.%s-fm.%s`, strings.ToLower(svcName), fnName, val),
		eventName:   val,
		service:     svcConst,
		url:         "",                        // look up from parent function if avail?
		typ:         "uclog.EventCategoryTODO", // we can't fill this in automatically yet
		subcategory: "na",
	}
	ev.code = generateCodeConstant(ev, svcName, fnName)

	addEvent(events, ev)
}

// generateCode generates a name for the code constant (a value will be assigned later)
func generateCodeConstant(e event, svcName, fnName string) string {
	// our convention is to use the service name & function name as a unique prefix
	// (error strings need to be unique within a function)
	capFnName := fmt.Sprintf("%s%s", strings.ToUpper(string(fnName[0])), fnName[1:])
	name := fmt.Sprintf("Event%sError%s", svcName, capFnName)

	// use the name without whitespace
	name += strings.ReplaceAll(e.eventName, " ", "")

	// we post-pend "Duration" if it's a duration event
	if e.typ == "uclog.EventCategoryDuration" {
		name += "Duration"
	}

	return name
}

func addEvent(events map[string]event, e event) {
	if _, ok := events[e.mapName]; !ok {
		events[e.mapName] = e
	}
}

func writeEventLine(e event) string {
	normalizedName := strings.ReplaceAll(e.eventName, " ", "")
	firstHalf := fmt.Sprintf(`"%s": {Name: "%s",  NormalizedName: "%s", Code: %s, Service: %s, Subcategory: "%s",`, e.mapName, e.eventName, normalizedName, e.code, e.service, e.subcategory)
	var middle string
	if e.ignore {
		middle = "Ignore: true, "
	}
	secondHalf := fmt.Sprintf(`URL: "%s", Category: %s},`, e.url, e.typ)

	return fmt.Sprintf("%s%s%s", firstHalf, middle, secondHalf)
}

const tmpString = `// auto-generated by genevents
// your changes will be preserved inside both the EventCodes constant block
// and the EventMap block, but names are auto-generated and so won't necessarily
// be used if they are incorrect
package events

import (
	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/uclog"
)

func init() {
	for k, ev := range << .ServiceName >>EventMap {
		RegisterEventType(k, ev)
	}
}

// Event codes
// TODO: get rid of these entirely?
const (
<< .EventCodes >>
)

var << .ServiceName >>EventMap = map[string]uclog.LogEventTypeInfo{
<< .EventMap >>
}
`
