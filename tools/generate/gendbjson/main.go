package gendbjson

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"golang.org/x/tools/go/packages"

	"userclouds.com/infra/uclog"
)

type data struct {
	Package string
	Type    string
}

// Run implements the generator interface
func Run(ctx context.Context, p *packages.Package, path string, args ...string) {
	data := data{Type: args[1], Package: p.Name}
	fn := filepath.Join(path, fmt.Sprintf("%s_dbjson_generated.go", strings.ToLower(data.Type)))
	fh, err := os.Create(fn)
	if err != nil {
		uclog.Fatalf(ctx, "error opening output file: %v", err)
	}

	temp := template.Must(template.New("dbjson").Delims("<<", ">>").Parse(templateString))
	if err := temp.Execute(fh, data); err != nil {
		uclog.Fatalf(ctx, "error executing template: %v", err)
	}
}

var templateString = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>

import (
	"database/sql/driver"
	"encoding/json"

	"userclouds.com/infra/ucerr"
)

// Value implements sql.Valuer
func (o << .Type >>) Value() (driver.Value, error) {
	return json.Marshal(o)
}

// Scan implements sql.Scanner
func (o *<< .Type >>) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return ucerr.New("type assertion failed for << .Type >>.Scan()")
	}
	return ucerr.Wrap(json.Unmarshal(b, &o))
}
`
