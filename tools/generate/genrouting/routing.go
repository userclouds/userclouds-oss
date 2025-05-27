package genrouting

import (
	"bytes"
	"context"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/uclog"
	"userclouds.com/internal/routinghelper"
)

// Run implements the generator interface
func Run(ctx context.Context, path string) {
	routeCfg, err := routinghelper.ParseIngressConfig(ctx, universe.Current(), "", "", service.AllWebServices)
	if err != nil {
		uclog.Fatalf(ctx, "failed to parse routing config: %v", err)
	}

	routeCfg.Rules = routeCfg.GetNonHostRules()
	fn := filepath.Join(path, "routing_generated.go")
	fh, err := os.Create(fn)
	if err != nil {
		uclog.Fatalf(ctx, "error opening output file: %v", err)
	}

	temp := template.Must(template.New("routing").Delims("<<", ">>").Parse(templateString))
	var bs []byte
	buf := bytes.NewBuffer(bs)
	if err := temp.Execute(buf, routeCfg); err != nil {
		uclog.Fatalf(ctx, "error executing template: %v", err)
	}

	// run gofmt to get things like spacing right
	p, err := format.Source(buf.Bytes())
	if err != nil {
		uclog.Fatalf(ctx, "error formatting source: %v", err)
	}
	if _, err := fh.Write(p); err != nil {
		uclog.Fatalf(ctx, "error writing output file: %v", err)
	}
	if err := fh.Close(); err != nil {
		uclog.Fatalf(ctx, "error closing output file: %v", err)
	}
}

var templateString = `// NOTE: automatically generated file -- DO NOT EDIT

package routing

import (
	"userclouds.com/infra/namespace/service"
	"userclouds.com/internal/routinghelper"
)

// Routing rules that are not host specific
var nonHostRules = []routinghelper.Rule{
<<- range $rule := .Rules >>
	{
		PathPrefixes: <<- printf "%#v" $rule.PathPrefixes >>,
		HostHeaders: nil,
		Service: service.<<- $rule.Service.ToCodeName >>,
	},
<<- end >>
}

// serviceToPort maps a service name to a port
var serviceToPort = map[service.Service]int{
<<- range $svc, $port := .Ports >>
	service.<<- $svc.ToCodeName >>: <<- $port >>,
<<- end >>
}
`
