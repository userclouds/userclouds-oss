package genevents

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"userclouds.com/infra/uclog"
	"userclouds.com/internal/repopath"
	"userclouds.com/tools/generate"
)

// returns two maps scoped to the current service (event code constants -> values, and
// event map names -> events), and the max event code across *all of UserClouds* (not just this service)
// so that we can assign new ones as needed
func parseExistingEvents(ctx context.Context, svcName, svcConst string) (map[string]uclog.EventCode, map[string]event, uclog.EventCode) {
	set, files, err := generate.LoadDir(ctx, filepath.Join(repopath.BaseDir(), "logserver/events"), 0)
	if err != nil {
		uclog.Fatalf(ctx, "failed to load events package: %v", err)
	}

	codes := map[string]uclog.EventCode{}
	events := map[string]event{}
	codeDecls := map[uclog.EventCode][]*ast.Ident{}

	var maxEventCode uclog.EventCode
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			decl, ok := n.(*ast.GenDecl)
			if !ok {
				return true
			}

			if decl.Tok == token.CONST {
				if c := parseEventCodes(ctx, svcName, decl, codes, codeDecls, set); c > maxEventCode {
					maxEventCode = c
				}
			} else if decl.Tok == token.VAR {
				parseEventMap(ctx, svcConst, decl, events)
			}

			return true
		})
	}

	// now that we've parsed all the event codes, check for duplicate values by looking at the
	// *ast.Ident declarations (eg. `EventConsoleListTenants uclog.EventCode = 1234`) and finding
	// duplicate *values* with different names.
	for _, v := range codeDecls {
		// if we have more than one declaration for this value, we have a duplicate
		// and need to choose one to re-issue a code for

		// TODO: this will require running twice to handle a triplicate code, but that
		// should be unusual enough right now that I'm not solving immediately

		if len(v) > 1 {
			// loop through all of the lines, git-blame them, and pick the hash that
			// appears latest in our tree (via git --is-ancestor), figuring that is the
			// least likely to have been provisioned etc.
			var latestHash string
			var latestName string
			for _, id := range v {
				fn := set.Position(id.Pos()).Filename
				ln := set.Position(id.Pos()).Line
				hash := parseGitBlame(ctx, fn, ln)
				if latestHash == "" || isGitAncestor(hash, latestHash) {
					latestHash = hash
					latestName = id.Name
				}
			}

			// only update this here if we actually found a duplicate in our current service
			// otherwise another run of genevents will get it
			if _, ok := codes[latestName]; ok {
				maxEventCode++
				uclog.Warningf(ctx, "found %d duplicate event codes for %v", len(v), v)
				uclog.Warningf(ctx, "updating event code %s to %d", latestName, maxEventCode)
				codes[latestName] = maxEventCode
			}
		}
	}

	return codes, events, maxEventCode
}

var gitBlameRE = regexp.MustCompile(`^([0-9a-f]+)\s\(.*?\s(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[\+-]\d{2}:\d{2})\s\d+\).*$`)

// parseGitBlame parses the git blame for the given file and line, returning the commit hash
func parseGitBlame(ctx context.Context, filename string, line int) string {
	cmd := exec.Command("git", "blame", "-L", fmt.Sprintf("%d,+1", line), "--date=iso-strict", filename)
	bs, err := cmd.Output()
	if err != nil {
		uclog.Fatalf(ctx, "failed to run git blame: %v", err)
	}

	str := strings.TrimSpace(string(bs))
	matches := gitBlameRE.FindStringSubmatch(str)
	if len(matches) != 3 {
		uclog.Fatalf(ctx, "failed to parse git blame output (len(matches)=%d): %s", len(matches), str)
	}

	// this turns out not to be useful but leaving it here for future
	if _, err := time.Parse(time.RFC3339, matches[2]); err != nil {
		uclog.Fatalf(ctx, "failed to parse git blame timestamp: %v", err)
	}

	return matches[1]
}

func isGitAncestor(ancestor, hash string) bool {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", hash, ancestor)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

// parseEventCodes parses the event codes from the events package for the given service,
// and also returns the max code it saw across all services (for later assignment if needed)
func parseEventCodes(ctx context.Context,
	svcName string,
	decl *ast.GenDecl,
	codes map[string]uclog.EventCode,
	codeDecls map[uclog.EventCode][]*ast.Ident,
	set *token.FileSet) uclog.EventCode {

	var maxEventCode uclog.EventCode

	for _, spec := range decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}

		// TODO: this will break if you rename the uclog import (please don't),
		// or if you use this code inside uclog (also please don't, but maybe someday)
		typeSel, ok := vs.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		typeSelID, ok := typeSel.X.(*ast.Ident)
		if !ok {
			continue
		}
		if typeSelID.Name != "uclog" && typeSel.Sel.Name != "EventCode" {
			continue
		}

		for i, name := range vs.Names {
			var eci int
			var err error
			v, ok := vs.Values[i].(*ast.BasicLit)
			if !ok {
				// this is a lot of code to handle a single case (-1 EventCodeUnknown)...
				pos := set.Position(vs.Values[i].Pos()).String()
				u, ok := vs.Values[i].(*ast.UnaryExpr)
				if !ok {
					uclog.Fatalf(ctx, "found a uclog.EventCode const that isn't a BasicLit or UnaryExpr: %v at %v", vs.Values[i], pos)
				}
				if u.Op != token.SUB {
					uclog.Fatalf(ctx, "found a uclog.EventCode UnaryExpr that isn't a simple negative: %v at %v", vs.Values[i], pos)
				}
				v, ok = u.X.(*ast.BasicLit)
				if !ok {
					uclog.Fatalf(ctx, "found a uclog.EventCode UnaryExpr where X isn't a basic literal: %v at %v", vs.Values[i], pos)
				}
				if v.Kind != token.INT {
					uclog.Fatalf(ctx, "found a uclog.EventCode UnaryExpr where X isn't an int: %v at %v", vs.Values[i], pos)
				}
				eci, err = strconv.Atoi(v.Value)
				if err != nil {
					uclog.Fatalf(ctx, "failed to parse uclog.EventCode %v as int: %v", v.Value, err)
				}
				// now apply u.Op, which we asserted above is a negation
				eci = -eci
			} else {
				// everything else is a simple BasicLit
				eci, err = strconv.Atoi(v.Value)
				if err != nil {
					uclog.Fatalf(ctx, "failed to parse uclog.EventCode %v as int: %v", v.Value, err)
				}
			}

			// keep our current max up to date
			if eci > int(maxEventCode) {
				maxEventCode = uclog.EventCode(eci)
			}

			// keep track of all the declarations for this event code
			codeDecls[uclog.EventCode(eci)] = append(codeDecls[uclog.EventCode(eci)], name)

			// only save the event codes for this service
			if !strings.HasPrefix(name.Name, fmt.Sprintf("Event%s", svcName)) {
				continue
			}

			codes[name.Name] = uclog.EventCode(eci)
		}
	}

	return maxEventCode
}

// parseEventMap parses the event map from the events package
func parseEventMap(ctx context.Context, svcConst string, decl *ast.GenDecl, events map[string]event) {
	for _, spec := range decl.Specs {
		vs, ok := spec.(*ast.ValueSpec)
		if !ok {
			return
		}

		// TODO: there's no reason we can't support this but it's not needed right now
		if len(vs.Values) != 1 {
			uclog.Fatalf(ctx, "unexpected number of vars in event var declaration: %v", vs.Names)
		}

		cl, ok := vs.Values[0].(*ast.CompositeLit)
		if !ok {
			uclog.Fatalf(ctx, "uclog.LogEventTypeInfo map declaration is not a CompositeLit: %v", vs.Values[0])
		}

		// TODO: this will break if you rename the uclog import (please don't),
		// or if you use this code inside uclog (also please don't, but maybe someday)
		mt, ok := cl.Type.(*ast.MapType)
		if !ok {
			return
		}
		typeSel, ok := mt.Value.(*ast.SelectorExpr)
		if !ok {
			return
		}
		typeSelID, ok := typeSel.X.(*ast.Ident)
		if !ok {
			return
		}
		if typeSelID.Name != "uclog" && typeSel.Sel.Name != "LogEventTypeInfo" {
			return
		}

		// range through the map elements
		for i, elt := range cl.Elts {
			mapElemKVE, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				uclog.Fatalf(ctx, "uclog.LogEventTypeInfo map element %d is not a KeyValueExpr: %v", i, elt)
			}

			// we're on a LogEventTypeInfo element so we'll store it here as we loop through the fields
			var ev event

			// this is the key of the map
			mekBL, ok := mapElemKVE.Key.(*ast.BasicLit)
			if !ok {
				uclog.Fatalf(ctx, "uclog.LogEventTypeInfo map element %d key is not a BasicLit: %v", i, mapElemKVE.Key)
			}

			// make sure to trim the quotes that the AST gives us
			ev.mapName = strings.Trim(mekBL.Value, "\"")
			if strings.Contains(ev.mapName, ".DB") {
				ev.subcategory = "db"
			} else if strings.Contains(ev.mapName, ".Count") || strings.Contains(ev.mapName, ".Duration") {
				ev.subcategory = "function"
			} else {
				ev.subcategory = "event"
			}

			icl, ok := mapElemKVE.Value.(*ast.CompositeLit) // inner composite lit
			if !ok {
				uclog.Fatalf(ctx, "uclog.LogEventTypeInfo map element %d is not a CompositeLit: %v", i, elt)
			}

			for j, ielt := range icl.Elts { // inner element
				kve, ok := ielt.(*ast.KeyValueExpr)
				if !ok {
					uclog.Fatalf(ctx, "uclog.LogEventTypeInfo map element %d[%d] is not a KeyValueExpr: %v", i, j, ielt)
				}

				kid, ok := kve.Key.(*ast.Ident)
				if !ok {
					uclog.Fatalf(ctx, "uclog.LogEventTypeInfo map element %d[%d] key is not an Ident: %v", i, j, kve.Key)
				}

				// this placeholder is lazy but it will fail loudly, and it's easier to read than a pointer
				// or a parallel found bool, and an empty string is actually a legit (temporary) value for eg. URL
				val := "<not found>"
				if bl, ok := kve.Value.(*ast.BasicLit); ok {
					val = bl.Value
				} else if id, ok := kve.Value.(*ast.Ident); ok {
					val = id.Name
				} else if sel, ok := kve.Value.(*ast.SelectorExpr); ok {
					if id, ok := sel.X.(*ast.Ident); ok {
						val = fmt.Sprintf("%s.%s", id.Name, sel.Sel.Name)
					}
				}
				// trim quotes that AST gives us so we insert consistently later
				val = strings.Trim(val, "\"")

				if val == "<not found>" {
					uclog.Fatalf(ctx, "uclog.LogEventTypeInfo map element %d[%d] value is not a BasicLit or Ident: %v", i, j, kve.Value)
				}

				switch kid.Name {
				case "Name":
					ev.eventName = val
				case "Code":
					ev.code = val
				case "Service":
					ev.service = val
				case "URL":
					ev.url = val
				case "Ignore":
					ev.ignore = val == "true"
				case "Category":
					ev.typ = val
				}
			}

			if !strings.EqualFold(ev.service, svcConst) {
				continue
			}

			events[ev.mapName] = ev
		}
	}
}
