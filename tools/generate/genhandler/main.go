package genhandler

import (
	"bytes"
	"context"
	"fmt"
	"go/ast"
	"go/printer"
	"go/token"
	"go/types"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"golang.org/x/tools/go/packages"

	"userclouds.com/infra/codegen"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/tools/generate"
)

// First set of structs are for the handler template
type handlerData struct {
	HandlerName        string
	HandlerTypeName    string
	CollectionBaseName string
	IsNested           bool
	MethodBaseName     string
	Method             string

	Found                bool
	GenOverrideFound     bool
	Description          string
	Summary              string
	Tags                 []string
	RequestTypeName      string
	QueryParameters      map[string]string
	QueryArrayParameters map[string]string
	ReturnTypeName       string
	ReturnTypePrefix     string
	SuccessCodes         []string
	ErrorCodes           []string
}

type collectionData struct {
	BaseName   string
	Authorizer string
	Path       string

	ParentBaseName string
	HasNested      bool

	GetHandlerName       string
	PostHandlerName      string
	DeleteHandlerName    string
	GetOneHandlerName    string
	PutOneHandlerName    string
	DeleteOneHandlerName string
}

type methodData struct {
	BaseName string
	Path     string

	GetHandlerName    string
	PostHandlerName   string
	PutHandlerName    string
	DeleteHandlerName string
}

type singleMethodData struct {
	MethodType  string
	HandlerName string
	Path        string
	Found       bool
}

type handlerTemplateData struct {
	HandlerTypeName   string
	Package           string
	Imports           string
	Handlers          []handlerData
	Collections       []collectionData
	NestedCollections []collectionData
	Methods           []methodData
	SingleMethods     []singleMethodData
}

// Next set of structs are for the openAPI template
type operationData struct {
	Path                    string
	Method                  string
	Summary                 string
	Description             string
	TagsString              string
	RequestTypeName         string
	RequestWithPathTypeName string
	SecondID                bool
	RequestObjectString     string
	ReturnObjectString      string
	SuccessCodes            []string
	ErrorCodes              []string
}

type openAPITemplateData struct {
	Package    string
	Imports    string
	Operations []operationData
}

// builderData is used to collect data about the handlers, collections and methods
type builderData struct {
	Collections   map[string]*collectionData
	Methods       map[string]*methodData
	SingleMethods map[string]*singleMethodData
}

// Run implements the generator interface
func Run(ctx context.Context, p *packages.Package, path string, args ...string) {

	builderData := builderData{Collections: map[string]*collectionData{}, Methods: map[string]*methodData{}, SingleMethods: map[string]*singleMethodData{}}
	handlerMap := map[string]*handlerData{}

	// Usage: genhandler <service_path> <handler_info>*
	// <handler_info> = { collection,[name],[authorizer],[path] | nestedcollection,[name],[authorizer],[path],[parent_name] | method,[name],[path] | [action],[name],[path] }
	// action = { GET | POST | PUT | DELETE }
	servicePath := args[1]

	// TODO (sgarrity 3/25): this is a quick hack to support public handler types,
	// which we need to unwind some circular dependencies in plex/internal/oidc
	// This should really be picked up automatically from the AST instead of with an arg,
	// but with that much work I could probably even find a better way to break the test dependency :)
	start := 2
	handlerTypeName := "handler"
	if len(args) > 2 && args[2] == "--public" {
		start = 3
		handlerTypeName = "Handler"
	}

	for i := start; i < len(args); i++ {
		argParts := strings.Split(args[i], ",")
		if len(argParts) < 3 || len(argParts) > 5 {
			uclog.Fatalf(ctx, "invalid handler info: %s", args[i])
		}

		if argParts[0] == "collection" {

			if len(argParts) != 4 {
				uclog.Fatalf(ctx, "invalid handler info: %s", args[i])
			}

			// TODO: this map could probably live / be reused in `uchttp`
			hm := map[string]string{
				"list":      "GET",
				"create":    "POST",
				"deleteAll": "DELETE",
				"get":       "GETONE",
				"update":    "PUTONE",
				"delete":    "DELETEONE",
			}

			for k, v := range hm {
				handlerName := fmt.Sprintf("%s%s", k, argParts[1])
				if k == "list" {
					handlerName = fmt.Sprintf("%s%s", k, generate.GetPluralName(argParts[1]))
				}
				handlerMap[handlerName] = &handlerData{
					HandlerName:          handlerName,
					HandlerTypeName:      handlerTypeName,
					CollectionBaseName:   argParts[1],
					Method:               v,
					QueryParameters:      map[string]string{},
					QueryArrayParameters: map[string]string{}}
			}

			collectionData := collectionData{
				BaseName:   argParts[1],
				Authorizer: argParts[2],
				Path:       argParts[3],
			}
			builderData.Collections[argParts[1]] = &collectionData

		} else if argParts[0] == "nestedcollection" {

			if len(argParts) != 5 {
				uclog.Fatalf(ctx, "invalid handler info: %s", args[i])
			}

			baseName := fmt.Sprintf("%sOn%s", argParts[1], argParts[4])

			hm := map[string]string{
				"list":      "GETONE",
				"create":    "POSTONE",
				"deleteAll": "DELETEONE",
				"get":       "GETNESTED",
				"update":    "PUTNESTED",
				"delete":    "DELETENESTED",
			}

			for k, v := range hm {
				var n string
				if k == "list" || k == "deleteAll" {
					n = fmt.Sprintf("%s%ssOn%s", k, argParts[1], argParts[4])
				} else {
					n = fmt.Sprintf("%s%sOn%s", k, argParts[1], argParts[4])
				}

				handlerMap[n] = &handlerData{
					HandlerName:          n,
					HandlerTypeName:      handlerTypeName,
					CollectionBaseName:   baseName,
					Method:               v,
					IsNested:             true,
					QueryParameters:      map[string]string{},
					QueryArrayParameters: map[string]string{},
				}
			}

			collectionData := collectionData{
				BaseName:       baseName,
				Authorizer:     argParts[2],
				Path:           argParts[3],
				ParentBaseName: argParts[4],
			}

			builderData.Collections[baseName] = &collectionData
			builderData.Collections[argParts[4]].HasNested = true

		} else if argParts[0] == "method" {

			if len(argParts) != 3 {
				uclog.Fatalf(ctx, "invalid handler info: %s", args[i])
			}

			hm := map[string]string{
				"get":    "GET",
				"create": "POST",
				"update": "PUT",
				"delete": "DELETE",
			}

			for k, v := range hm {
				n := fmt.Sprintf("%s%s", k, argParts[1])
				handlerMap[n] = &handlerData{
					HandlerName:          n,
					HandlerTypeName:      handlerTypeName,
					MethodBaseName:       argParts[1],
					Method:               v,
					QueryParameters:      map[string]string{},
					QueryArrayParameters: map[string]string{}}
			}

			methodData := methodData{
				BaseName: argParts[1],
				Path:     argParts[2],
			}
			builderData.Methods[argParts[1]] = &methodData

		} else {

			if len(argParts) != 3 {
				uclog.Fatalf(ctx, "invalid handler info: %s", args[i])
			}

			if argParts[0] != "GET" && argParts[0] != "POST" && argParts[0] != "PUT" && argParts[0] != "DELETE" {
				uclog.Fatalf(ctx, "invalid handler info: %s", args[i])
			}

			handlerMap[argParts[1]] = &handlerData{
				HandlerName:          argParts[1],
				HandlerTypeName:      handlerTypeName,
				Method:               argParts[0],
				QueryParameters:      map[string]string{},
				QueryArrayParameters: map[string]string{},
			}

			singleMethodData := singleMethodData{
				MethodType:  argParts[0],
				HandlerName: argParts[1],
				Path:        argParts[2],
			}
			builderData.SingleMethods[argParts[1]] = &singleMethodData

		}
	}

	imports := codegen.NewImports()

	// extract request and return types for each of the handlers from p.TypesInfo
	if err := extractTypes(p.TypesInfo, imports, p.PkgPath, handlerMap, &builderData); err != nil {
		uclog.Fatalf(ctx, "error extracting types: %v", err)
	}

	// parse the functions for each of the handlers to get their potential success and error codes
	if err := extractCodes(ctx, path, handlerMap); err != nil {
		uclog.Fatalf(ctx, "error extracting codes: %v", err)
	}

	// generate the handlers
	handlerTemplateData := handlerTemplateData{
		Package:         p.Name,
		HandlerTypeName: handlerTypeName,
	}

	// populate the handlerTemplateData with the handlers, collections and methods, and sort the arrays so that codegen is consistent
	includeUUIDPackage := false
	operationsFound := false
	for _, hd := range handlerMap {
		if hd.Found {
			handlerTemplateData.Handlers = append(handlerTemplateData.Handlers, *hd)

			if hd.Method == "GETONE" || hd.Method == "PUTONE" || hd.Method == "DELETEONE" {
				includeUUIDPackage = true
			}

			operationsFound = true
		}
	}

	dataImports := imports.Copy()
	dataImports.Add("userclouds.com/infra/uchttp/builder")
	if operationsFound {
		dataImports.Add("net/http")
		dataImports.Add("userclouds.com/infra/jsonapi")
		dataImports.Add("userclouds.com/internal/auditlog")
	}
	if includeUUIDPackage {
		dataImports.Add("github.com/gofrs/uuid")
	}
	handlerTemplateData.Imports = dataImports.String()

	sort.Slice(handlerTemplateData.Handlers, func(i, j int) bool {
		if handlerTemplateData.Handlers[i].CollectionBaseName == handlerTemplateData.Handlers[j].CollectionBaseName &&
			handlerTemplateData.Handlers[i].MethodBaseName == handlerTemplateData.Handlers[j].MethodBaseName {
			return handlerTemplateData.Handlers[i].HandlerName < handlerTemplateData.Handlers[j].HandlerName
		} else if handlerTemplateData.Handlers[i].CollectionBaseName == handlerTemplateData.Handlers[j].CollectionBaseName {
			return handlerTemplateData.Handlers[i].MethodBaseName < handlerTemplateData.Handlers[j].MethodBaseName
		}
		return handlerTemplateData.Handlers[i].CollectionBaseName < handlerTemplateData.Handlers[j].CollectionBaseName
	})

	for _, cd := range builderData.Collections {
		if cd.GetHandlerName != "" || cd.PostHandlerName != "" || cd.DeleteHandlerName != "" || cd.GetOneHandlerName != "" || cd.PutOneHandlerName != "" || cd.DeleteOneHandlerName != "" {
			if cd.ParentBaseName != "" {
				handlerTemplateData.NestedCollections = append(handlerTemplateData.NestedCollections, *cd)
			} else {
				handlerTemplateData.Collections = append(handlerTemplateData.Collections, *cd)
			}
		}
	}
	sort.Slice(handlerTemplateData.Collections, func(i, j int) bool {
		return handlerTemplateData.Collections[i].BaseName < handlerTemplateData.Collections[j].BaseName
	})
	sort.Slice(handlerTemplateData.NestedCollections, func(i, j int) bool {
		return handlerTemplateData.NestedCollections[i].BaseName < handlerTemplateData.NestedCollections[j].BaseName
	})

	for _, md := range builderData.Methods {
		if md.GetHandlerName != "" || md.PostHandlerName != "" || md.PutHandlerName != "" || md.DeleteHandlerName != "" {
			handlerTemplateData.Methods = append(handlerTemplateData.Methods, *md)
		}
	}
	sort.Slice(handlerTemplateData.Methods, func(i, j int) bool {
		return handlerTemplateData.Methods[i].BaseName < handlerTemplateData.Methods[j].BaseName
	})

	for _, smd := range builderData.SingleMethods {
		if smd.Found {
			handlerTemplateData.SingleMethods = append(handlerTemplateData.SingleMethods, *smd)
		}
	}
	sort.Slice(handlerTemplateData.SingleMethods, func(i, j int) bool {
		return handlerTemplateData.SingleMethods[i].Path < handlerTemplateData.SingleMethods[j].Path
	})

	fn := filepath.Join(path, "handlers_generated.go")
	generate.WriteFileIfChanged(ctx, fn, handlerTemplateString, handlerTemplateData)

	// generate the openAPI spec
	openAPIImports := imports.Copy()
	openAPIImports.Add("context")
	openAPIImports.Add("github.com/swaggest/openapi-go/openapi3")
	openAPIImports.Add("github.com/swaggest/jsonschema-go")
	openAPIImports.Add("github.com/gofrs/uuid")
	if operationsFound {
		openAPIImports.Add("net/http")
		openAPIImports.Add("github.com/swaggest/openapi-go")
		openAPIImports.Add("userclouds.com/infra/uclog")
	}

	openAPITemplateData := openAPITemplateData{
		Package: p.Name,
		Imports: openAPIImports.String(),
	}

	// populate the openAPITemplateData with the operations, and sort the array so that codegen is consistent
	for _, hd := range handlerMap {
		if hd.Found {

			var path string
			if hd.CollectionBaseName != "" {
				if hd.IsNested {
					path = builderData.Collections[builderData.Collections[hd.CollectionBaseName].ParentBaseName].Path + "/{id}" + builderData.Collections[hd.CollectionBaseName].Path
				} else {
					path = builderData.Collections[hd.CollectionBaseName].Path
				}
			} else if hd.MethodBaseName != "" {
				path = builderData.Methods[hd.MethodBaseName].Path
			} else {
				path = builderData.SingleMethods[hd.HandlerName].Path
			}

			returnObjectString := "nil"
			if hd.ReturnTypeName != "" {
				if hd.ReturnTypePrefix == "[]" {
					returnObjectString = fmt.Sprintf("new([]%s)", hd.ReturnTypeName)
				} else {
					returnObjectString = fmt.Sprintf("new(%s)", hd.ReturnTypeName)
				}
			}

			requestObjectString := "nil"
			requestWithPathTypeName := ""
			secondID := false

			var method string

			if hd.IsNested {
				switch hd.Method {
				case "GETONE":
					method = "GET"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
				case "POSTONE":
					method = "POST"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
				case "DELETEONE":
					method = "DELETE"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
				case "GETNESTED":
					method = "GET"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
					path += "/{id2}"
					secondID = true
				case "PUTNESTED":
					method = "PUT"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
					path += "/{id2}"
					secondID = true
				case "DELETENESTED":
					method = "DELETE"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
					path += "/{id2}"
					secondID = true
				}
			} else {
				switch hd.Method {
				case "GET":
					method = "GET"
					if hd.RequestTypeName != "" {
						requestObjectString = fmt.Sprintf("new(%s)", hd.RequestTypeName)
					}
				case "POST":
					method = "POST"
					requestObjectString = fmt.Sprintf("new(%s)", hd.RequestTypeName)
				case "DELETE":
					method = "DELETE"
					if hd.RequestTypeName != "" {
						requestObjectString = fmt.Sprintf("new(%s)", hd.RequestTypeName)
					}
				case "GETONE":
					method = "GET"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
					path += "/{id}"
				case "PUTONE":
					method = "PUT"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
					path += "/{id}"
				case "DELETEONE":
					method = "DELETE"
					requestWithPathTypeName = makeRequestWithPathTypeName(hd.RequestTypeName)
					requestObjectString = fmt.Sprintf("new(%s)", requestWithPathTypeName)
					path += "/{id}"
				}
			}

			successCodes := hd.SuccessCodes
			if len(successCodes) == 0 {
				successCodes = []string{"200"}
			}

			errorCodes := hd.ErrorCodes

			tagsString := ""
			if hd.Tags != nil {
				tagsString = `"` + strings.Join(hd.Tags, `","`) + `"`
			}
			openAPITemplateData.Operations = append(openAPITemplateData.Operations, operationData{
				Path:                    servicePath + path,
				Method:                  fmt.Sprintf("http.Method%s", cases.Title(language.English).String(method)),
				Summary:                 hd.Summary,
				Description:             hd.Description,
				TagsString:              tagsString,
				RequestTypeName:         hd.RequestTypeName,
				RequestWithPathTypeName: requestWithPathTypeName,
				SecondID:                secondID,
				RequestObjectString:     requestObjectString,
				ReturnObjectString:      returnObjectString,
				SuccessCodes:            successCodes,
				ErrorCodes:              errorCodes,
			})
		}
	}
	sort.Slice(openAPITemplateData.Operations, func(i, j int) bool {
		if openAPITemplateData.Operations[i].Path == openAPITemplateData.Operations[j].Path {
			return openAPITemplateData.Operations[i].Method < openAPITemplateData.Operations[j].Method
		}
		return openAPITemplateData.Operations[i].Path < openAPITemplateData.Operations[j].Path
	})

	fn = filepath.Join(path, "openapi_generated.go")
	generate.WriteFileIfChanged(ctx, fn, openAPITemplateString, openAPITemplateData)
}

var queryRegExp = regexp.MustCompile(`query:"([^"]*)"`)
var descriptionRegExp = regexp.MustCompile(`OpenAPI Description:(.*)`)
var summaryRegExp = regexp.MustCompile(`OpenAPI Summary:(.*)`)
var tagsRegExp = regexp.MustCompile(`OpenAPI Tags:(.*)`)

// looks for functions that match the names of the handlers specified and extracts the request and return types, marks them as found
func extractTypes(typesInfo *types.Info, imports *codegen.Imports, pkgPath string, handlerMap map[string]*handlerData, builderData *builderData) error {

	for id, obj := range typesInfo.Defs {

		if hd, ok := handlerMap[id.Name]; ok {

			hd.Found = true
			processQueryParams := false
			if hd.CollectionBaseName != "" {
				collectionData := builderData.Collections[hd.CollectionBaseName]

				if hd.IsNested {
					switch hd.Method {
					case "GETONE":
						collectionData.GetHandlerName = id.Name
						processQueryParams = true
					case "POSTONE":
						collectionData.PostHandlerName = id.Name
					case "DELETEONE":
						collectionData.DeleteHandlerName = id.Name
						processQueryParams = true
					case "GETNESTED":
						collectionData.GetOneHandlerName = id.Name
						processQueryParams = true
					case "PUTNESTED":
						collectionData.PutOneHandlerName = id.Name
					case "DELETENESTED":
						collectionData.DeleteOneHandlerName = id.Name
						processQueryParams = true
					}
				} else {
					switch hd.Method {
					case "GET":
						collectionData.GetHandlerName = id.Name
						processQueryParams = true
					case "POST":
						collectionData.PostHandlerName = id.Name
					case "DELETE":
						collectionData.DeleteHandlerName = id.Name
						processQueryParams = true
					case "GETONE":
						collectionData.GetOneHandlerName = id.Name
						processQueryParams = true
					case "PUTONE":
						collectionData.PutOneHandlerName = id.Name
					case "DELETEONE":
						collectionData.DeleteOneHandlerName = id.Name
						processQueryParams = true
					}
				}
			} else if hd.MethodBaseName != "" {
				methodData := builderData.Methods[hd.MethodBaseName]
				switch hd.Method {
				case "GET":
					methodData.GetHandlerName = id.Name
					processQueryParams = true
				case "POST":
					methodData.PostHandlerName = id.Name
				case "PUT":
					methodData.PutHandlerName = id.Name
				case "DELETE":
					methodData.DeleteHandlerName = id.Name
					processQueryParams = true
				}
			} else {
				singleMethodData := builderData.SingleMethods[hd.HandlerName]
				singleMethodData.Found = true
				if singleMethodData.MethodType == "DELETE" || singleMethodData.MethodType == "GET" || singleMethodData.MethodType == "POST" {
					processQueryParams = true
				}
			}

			f := obj.(*types.Func)
			sig := f.Type().(*types.Signature)

			params := sig.Params()
			if params.Len() < 2 {
				return ucerr.Errorf("expected at least 2 params for %s, got %v", id.Name, params)
			}
			if params.At(0).Type().String() != "context.Context" {
				return ucerr.Errorf("expected first param of %s to be context.Context, got %s", id.Name, params.At(0).Type().String())
			}

			reqParam := params.At(params.Len() - 1)
			packageName, typeName, _ := parseType(pkgPath, reqParam.Type().String())
			if typeName != "url.Values" {
				hd.RequestTypeName = typeName
				if packageName != "" {
					imports.Add(packageName)
				}

				// Find all parameters in the request struct with query tags and store them in the QueryParameters map
				if processQueryParams {
					if s, ok := reqParam.Type().Underlying().(*types.Struct); ok {
						for i := range s.NumFields() {

							field := s.Field(i)
							if field.Embedded() {

								// If this is an embedded struct, we need to look at the fields of the embedded type (we only go one layer deep)
								if embedded, ok := field.Type().Underlying().(*types.Struct); ok {
									for j := range embedded.NumFields() {
										tags := embedded.Tag(j)
										if queryMatches := queryRegExp.FindStringSubmatch(tags); queryMatches != nil {
											if embedded.Field(j).Type().String() == "[]string" {
												hd.QueryArrayParameters[embedded.Field(j).Name()] = queryMatches[1]
											} else {
												hd.QueryParameters[embedded.Field(j).Name()] = queryMatches[1]
											}
										}
									}
								}

							} else {

								tags := s.Tag(i)
								if queryMatches := queryRegExp.FindStringSubmatch(tags); queryMatches != nil {
									if s.Field(i).Type().String() == "[]string" {
										hd.QueryArrayParameters[s.Field(i).Name()] = queryMatches[1]
									} else {
										hd.QueryParameters[s.Field(i).Name()] = queryMatches[1]
									}
								}

							}
						}
					}
				}
			}

			results := sig.Results()
			if results.Len() != 3 && results.Len() != 4 {
				return ucerr.Errorf("expected 3 or 4 results for %s, got %v", id.Name, results)
			}
			if results.At(results.Len()-3).Type().String() != "int" {
				return ucerr.Errorf("expected third to last result of %s to be int, got %s", id.Name, results.At(results.Len()-3).Type().String())
			}
			if results.At(results.Len()-1).Type().String() != "error" {
				return ucerr.Errorf("expected last result of %s to be error, got %s", id.Name, results.At(results.Len()-1).Type().String())
			}
			if results.Len() == 4 {
				packageName, typeName, prefix := parseType(pkgPath, results.At(0).Type().String())
				if packageName != "" {
					imports.Add(packageName)
				}
				hd.ReturnTypeName = typeName
				hd.ReturnTypePrefix = prefix
			}
		} else if strings.HasSuffix(id.Name, "GeneratedOverride") {
			handlerName := strings.TrimSuffix(id.Name, "GeneratedOverride")
			if hd, ok := handlerMap[handlerName]; ok {
				hd.GenOverrideFound = true
			}
		}
	}

	return nil
}

func makeRequestWithPathTypeName(typeName string) string {
	if typeName == "" {
		return "RequestAndPath"
	}

	parts := strings.Split(typeName, ".")
	ret := parts[len(parts)-1]
	if len(parts) == 1 {
		return ret + "AndPath"
	}
	return ret
}

// parses a type string into a package name and type name
func parseType(curPackage string, fullTypeName string) (packageName string, typeName string, prefix string) {
	if strings.HasPrefix(fullTypeName, "*") {
		prefix = "*"
	}
	if strings.HasPrefix(fullTypeName, "[]") {
		prefix = "[]"
	}
	fullTypeParts := strings.Split(strings.ReplaceAll(strings.ReplaceAll(fullTypeName, "*", ""), "[]", ""), "/")
	typeName = fullTypeParts[len(fullTypeParts)-1]

	typeParts := strings.Split(fullTypeParts[len(fullTypeParts)-1], ".")
	if len(fullTypeParts) > 1 {
		packageName = strings.Join(fullTypeParts[:len(fullTypeParts)-1], "/") + "/" + typeParts[0]
	}

	if packageName == curPackage {
		packageName = ""
		typeName = typeParts[1]
	}

	return
}

// parses the functions for each of the handlers to get their potential success and error codes
func extractCodes(ctx context.Context, dir string, handlerMap map[string]*handlerData) error {
	fset, files, err := generate.LoadDir(ctx, dir, 0)
	if err != nil {
		return ucerr.Errorf("failed to load dir %s: %v", dir, err)
	}

	for _, file := range files {
		// Create an ast.CommentMap from the ast.File's comments.
		commentMap := ast.NewCommentMap(fset, file, file.Comments)
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok {
				return true
			}

			hd, ok := handlerMap[fn.Name.Name]
			if !ok {
				return false
			}

			comment, ok := commentMap[fn]
			if ok {
				for _, c := range comment {
					if descriptionMatches := descriptionRegExp.FindStringSubmatch(c.Text()); descriptionMatches != nil {
						hd.Description = strings.TrimSpace(descriptionMatches[1])
					}
					if summaryMatches := summaryRegExp.FindStringSubmatch(c.Text()); summaryMatches != nil {
						hd.Summary = strings.TrimSpace(summaryMatches[1])
					}
					if tagMatches := tagsRegExp.FindStringSubmatch(c.Text()); tagMatches != nil {
						tags := strings.SplitSeq(tagMatches[1], ",")
						for tag := range tags {
							hd.Tags = append(hd.Tags, strings.TrimSpace(tag))
						}
					}
				}
			}

			successCodes := set.NewStringSet()
			errorCodes := set.NewStringSet()
			ast.Inspect(fn, func(n ast.Node) bool {
				ret, ok := n.(*ast.ReturnStmt)
				if !ok {
					return true
				}

				l := len(ret.Results)
				if l != 3 && l != 4 {
					uclog.Fatalf(ctx, "expected 3 or 4 results in %s, got %d", fn.Name.Name, len(ret.Results))
				}
				errResult, err := exprToString(fset, ret.Results[l-1])
				if err != nil {
					uclog.Fatalf(ctx, "failed to print return result: %v", err)
				}

				codeResult, err := exprToString(fset, ret.Results[l-3])
				if err != nil {
					uclog.Fatalf(ctx, "failed to print return result: %v", err)
				}

				if errResult == "nil" {
					successCodes.Insert(codeResult)
				} else {
					// Recognize the error code helper and insert types it can return
					if strings.HasPrefix(codeResult, "uchttp.SQLReadErrorMapper") {
						errorCodes.Insert([]string{"http.StatusNotFound", "http.StatusInternalServerError"}...)
					} else if strings.HasPrefix(codeResult, "uchttp.SQLDeleteErrorMapper") {
						errorCodes.Insert([]string{"http.StatusNotFound", "http.StatusInternalServerError"}...)
					} else if strings.HasPrefix(codeResult, "uchttp.SQLWriteErrorMapper") {
						errorCodes.Insert([]string{"http.StatusConflict", "http.StatusInternalServerError"}...)
					} else {
						errorCodes.Insert(codeResult)
					}
				}
				return false
			})

			hd.SuccessCodes = successCodes.Items()
			hd.ErrorCodes = errorCodes.Items()
			return false
		})
	}
	return nil
}

func exprToString(fset *token.FileSet, expr ast.Expr) (string, error) {
	var buf bytes.Buffer
	if err := printer.Fprint(&buf, fset, expr); err != nil {
		return "", ucerr.Wrap(err)
	}
	return buf.String(), nil
}

var handlerTemplateString = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>

import (
<< .Imports >>
)

func handlerBuilder(builder *builder.HandlerBuilder, h *<< .HandlerTypeName >>) {

	<<- range .Collections >>
	<< print "\n" >>
	<<- if .HasNested >> handler<< .BaseName>> := <<- end >> builder.CollectionHandler("<< .Path >>").
	<<- if .GetOneHandlerName >>
		GetOne(h.<< .GetOneHandlerName >>Generated).
	<<- end >>
	<<- if .PostHandlerName >>
		Post(h.<< .PostHandlerName >>Generated).
	<<- end >>
	<<- if .PutOneHandlerName >>
		Put(h.<< .PutOneHandlerName >>Generated).
	<<- end >>
	<<- if .DeleteOneHandlerName >>
		Delete(h.<< .DeleteOneHandlerName >>Generated).
	<<- end >>
	<<- if .GetHandlerName >>
		GetAll(h.<< .GetHandlerName >>Generated).
	<<- end >>
	<<- if .DeleteHandlerName >>
		DeleteAll(h.<< .DeleteHandlerName >>Generated).
	<<- end >>
		WithAuthorizer(<< .Authorizer >>)
	<<- end >>

	<<- range .NestedCollections >>
	<< print "\n" >>
		handler<< .ParentBaseName >>.NestedCollectionHandler("<< .Path >>").
	<<- if .GetOneHandlerName >>
		GetOne(h.<< .GetOneHandlerName >>Generated).
	<<- end >>
	<<- if .PostHandlerName >>
		Post(h.<< .PostHandlerName >>Generated).
	<<- end >>
	<<- if .PutOneHandlerName >>
		Put(h.<< .PutOneHandlerName >>Generated).
	<<- end >>
	<<- if .DeleteOneHandlerName >>
		Delete(h.<< .DeleteOneHandlerName >>Generated).
	<<- end >>
	<<- if .GetHandlerName >>
		GetAll(h.<< .GetHandlerName >>Generated).
	<<- end >>
	<<- if .DeleteHandlerName >>
		DeleteAll(h.<< .DeleteHandlerName >>Generated).
	<<- end >>
		WithAuthorizer(<< .Authorizer >>)
	<<- end >>

	<<- range .Methods >>

	builder.MethodHandler("<< .Path >>").
	<<- if .GetHandlerName >>
		Get(h.<< .GetHandlerName >>Generated).
	<<- end >>
	<<- if .PostHandlerName >>
		Post(h.<< .PostHandlerName >>Generated).
	<<- end >>
	<<- if .PutHandlerName >>
		Put(h.<< .PutHandlerName >>Generated).
	<<- end >>
	<<- if .DeleteHandlerName >>
		Delete(h.<< .DeleteHandlerName >>Generated).
	<<- end >>
		End()

	<<- end >>
	<<- range .SingleMethods >>

	builder.MethodHandler("<< .Path >>").<<- if eq .MethodType "GET" >>Get<<- else if eq .MethodType "POST" >>Post<<- else if eq .MethodType "PUT" >> Put<<- else if eq .MethodType "DELETE">>Delete<<- end >>(h.<< .HandlerName >>Generated)

	<<- end >>

}
<<- range .Handlers >>

<<- if .GenOverrideFound >>
	<<- if eq .Method "GET" "POST" "DELETE" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request) {
	entries := h.<< .HandlerName >>GeneratedOverride(w, r)
	auditlog.PostMultipleAsync(r.Context(), entries)
}

	<<- else if eq .Method "GETONE" "PUTONE" "DELETEONE" "POSTONE" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	entries := h.<< .HandlerName >>GeneratedOverride(w, r, id)
	auditlog.PostMultipleAsync(r.Context(), entries)
}

	<<- else if eq .Method "GETNESTED" "PUTNESTED" "DELETENESTED" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, id uuid.UUID, id2 uuid.UUID) {
	entries := h.<< .HandlerName >>GeneratedOverride(w, r, id, id2)
	auditlog.PostMultipleAsync(r.Context(), entries)
}

	<<- end >>
<<- else if eq .Method "GET" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	<<- if .RequestTypeName >>

	req := << .RequestTypeName >>{}
		<<- range $k, $v := .QueryParameters >>
	if urlValues.Has("<< $v >>") && urlValues.Get("<< $v >>") != "null" {
		v := urlValues.Get("<< $v >>")
		req.<< $k >> = &v
	}
		<<- end >>
		<<- range $k, $v := .QueryArrayParameters >>
	if urlValues.Has("<< $v >>") {
		req.<< $k >> = urlValues["<< $v >>"]
	}
		<<- end >>

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, req)
	<<- else >>

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, urlValues)
	<<- end >>
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

<<- else if eq .Method "POST" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	<<- if .QueryParameters >>
	urlValues := r.URL.Query()
	<<- end >>

	var req << .RequestTypeName >>
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}
	<<- range $k, $v := .QueryParameters >>
	if urlValues.Has("<< $v >>") && urlValues.Get("<< $v >>") != "null" {
		v := urlValues.Get("<< $v >>")
		req.<< $k >> = &v
	}
	<<- end >>
	<<- range $k, $v := .QueryArrayParameters >>
	if urlValues.Has("<< $v >>") {
		req.<< $k >> = urlValues["<< $v >>"]
	}
	<<- end >>

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

<<- else if eq .Method "GETONE" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	<<- if .RequestTypeName >>

	req := << .RequestTypeName >>{}
		<<- range $k, $v := .QueryParameters >>
	if urlValues.Has("<< $v >>") && urlValues.Get("<< $v >>") != "null" {
		v := urlValues.Get("<< $v >>")
		req.<< $k >> = &v
	}
		<<- end >>
		<<- range $k, $v := .QueryArrayParameters >>
	if urlValues.Has("<< $v >>") {
		req.<< $k >> = urlValues["<< $v >>"]
	}
		<<- end >>

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, id, req)
	<<- else >>

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, id, urlValues)
	<<- end >>
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

<<- else if eq .Method "PUTONE" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()

	var req << .RequestTypeName >>
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}


	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

<<- else if eq .Method "DELETEONE" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	<<- if .RequestTypeName >>

	req := << .RequestTypeName >>{}
		<<- range $k, $v := .QueryParameters >>
	if urlValues.Has("<< $v >>") && urlValues.Get("<< $v >>") != "null" {
		v := urlValues.Get("<< $v >>")
		req.<< $k >> = &v
	}
		<<- end >>
	<<- range $k, $v := .QueryArrayParameters >>
		if urlValues.Has("<< $v >>") {
			req.<< $k >> = urlValues["<< $v >>"]
		}
	<<- end >>

	code, entries, err := h.<< .HandlerName >>(ctx, id, req)
	<<- else >>

	code, entries, err := h.<< .HandlerName >>(ctx, id, urlValues)
	<<- end >>
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

<<- else if eq .Method "DELETE" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	<<- if .RequestTypeName >>

	req := << .RequestTypeName >>{}
		<<- range $k, $v := .QueryParameters >>
	if urlValues.Has("<< $v >>") && urlValues.Get("<< $v >>") != "null" {
		v := urlValues.Get("<< $v >>")
		req.<< $k >> = &v
	}
		<<- end >>
		<<- range $k, $v := .QueryArrayParameters >>
	if urlValues.Has("<< $v >>") {
		req.<< $k >> = urlValues["<< $v >>"]
	}
		<<- end >>

	code, entries, err := h.<< .HandlerName >>(ctx, req)
	<<- else >>

	code, entries, err := h.<< .HandlerName >>(ctx, urlValues)
	<<- end >>
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

<<- else if eq .Method "POSTONE" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID) {
	ctx := r.Context()

	var req << .RequestTypeName >>
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, parentID, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

<<- else if eq .Method "GETNESTED" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	<<- if .RequestTypeName >>

	req := << .RequestTypeName >>{}
		<<- range $k, $v := .QueryParameters >>
	if urlValues.Has("<< $v >>") && urlValues.Get("<< $v >>") != "null" {
		v := urlValues.Get("<< $v >>")
		req.<< $k >> = &v
	}
		<<- end >>
		<<- range $k, $v := .QueryArrayParameters >>
	if urlValues.Has("<< $v >>") {
		req.<< $k >> = urlValues["<< $v >>"]
	}
		<<- end >>

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, parentID, id, req)
	<<- else >>

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, parentID, id, urlValues)
	<<- end >>
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

<<- else if eq .Method "PUTNESTED" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()

	var req << .RequestTypeName >>
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res << .ReturnTypePrefix >><< .ReturnTypeName >>
	res, code, entries, err := h.<< .HandlerName >>(ctx, parentID, id, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}

<<- else if eq .Method "DELETENESTED" >>

func (h *<< .HandlerTypeName >>) << .HandlerName >>Generated(w http.ResponseWriter, r *http.Request, parentID uuid.UUID, id uuid.UUID) {
	ctx := r.Context()
	urlValues := r.URL.Query()

	<<- if .RequestTypeName >>

	req := << .RequestTypeName >>{}
		<<- range $k, $v := .QueryParameters >>
	if urlValues.Has("<< $v >>") && urlValues.Get("<< $v >>") != "null" {
		v := urlValues.Get("<< $v >>")
		req.<< $k >> = &v
	}
		<<- end >>
		<<- range $k, $v := .QueryArrayParameters >>
	if urlValues.Has("<< $v >>") {
		req.<< $k >> = urlValues["<< $v >>"]
	}
		<<- end >>

	code, entries, err := h.<< .HandlerName >>(ctx, parentID, id, req)
	<<- else >>

	code, entries, err := h.<< .HandlerName >>(ctx, parentID, id, urlValues)
	<<- end >>
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	if code == http.StatusNoContent {
		w.WriteHeader(code)
		return
	}

	jsonapi.Marshal(w, nil, jsonapi.Code(code))
}

<<- end >>

<<- end >>

`

var openAPITemplateString = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>

import (
<< .Imports >>
)

// BuildOpenAPISpec is a generated helper function for building the OpenAPI spec for this service
func BuildOpenAPISpec(ctx context.Context, reflector *openapi3.Reflector) {

	{
		// Create custom schema mapping for 3rd party type.
		uuidDef := jsonschema.Schema{}
		uuidDef.AddType(jsonschema.String)
		uuidDef.WithFormat("uuid")
		uuidDef.WithExamples("248df4b7-aa70-47b8-a036-33ac447e668d")

		// Map 3rd party type with your own schema.
		reflector.AddTypeMapping(uuid.UUID{}, uuidDef)
	}

	<<- range .Operations >>
	<< $ret := .ReturnObjectString >>
	{
		op, err := reflector.NewOperationContext(<< .Method >>, "<< .Path >>")
		if err != nil {
			uclog.Fatalf(ctx, "failed to creation operation context: %v", err)
		}
		<<- if .Summary >>
		op.SetSummary("<< .Summary >>")
		<<- end >>
		<<- if .Description >>
		op.SetDescription("<< .Description >>")
		<<- end >>
		<<- if .TagsString >>
		op.SetTags(<< .TagsString >>)
		<<- end >>
		<<- if .RequestWithPathTypeName >>
		type << .RequestWithPathTypeName >> struct {
			ID uuid.UUID ` + "`" + `path:"id"` + "`" + `
			<<- if .SecondID >>
			ID2 uuid.UUID ` + "`" + `path:"id2"` + "`" + `
			<<- end >>
			<< .RequestTypeName >>
		}
		<<- end >>
		op.AddReqStructure(<< .RequestObjectString >>)
		<<- range $code := .SuccessCodes >>
		op.AddRespStructure(<< $ret >>, openapi.WithHTTPStatus(<< $code >>))
		<<- end >>
		<<- range $code := .ErrorCodes >>
		op.AddRespStructure(nil, openapi.WithHTTPStatus(<< $code >>))
		<<- end >>
		if err := reflector.AddOperation(op); err != nil {
			uclog.Fatalf(ctx, "failed to add operation: %v", err)
		}
	}

	<<- end >>
}
`
