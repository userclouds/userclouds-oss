package provider

import (
	"bytes"
	"context"
	"go/format"
	"log"
	"os"
	"text/template"

	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/config"
	"github.com/userclouds/terraform-provider-userclouds/genprovider/internal/resources"
)

type data struct {
	Package          string
	ImportPackages   []string
	NewResourceFuncs []string
}

var temp = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	<<- range $i, $importPackage := .ImportPackages >>
	"github.com/userclouds/terraform-provider-userclouds/internal/provider/<< $importPackage >>"
	<<- end >>
)

var generatedResources = []func() resource.Resource{
	<<- range $i, $name := .NewResourceFuncs >>
	<< $name >>,
	<<- end >>
}

var generatedDataSources = []func() datasource.DataSource{
}
`

// GenProvider generates the code for implementing a Terraform provider that references the
// generated resources.
func GenProvider(ctx context.Context, packageName string, outDir string, config *config.GenerationConfig) {
	data := data{
		Package: packageName,
	}
	for _, spec := range config.Specs {
		data.ImportPackages = append(data.ImportPackages, spec.GeneratedFilePackage)
		for _, r := range spec.Resources {
			data.NewResourceFuncs = append(data.NewResourceFuncs, spec.GeneratedFilePackage+"."+resources.TFTypeNameSuffixToNewResourceFuncName(r.TypeNameSuffix))
		}
	}

	temp := template.Must(template.New("providerTemplate").Delims("<<", ">>").Parse(temp))
	buf := bytes.NewBuffer([]byte{})
	if err := temp.Execute(buf, data); err != nil {
		log.Fatalf("error executing template: %v", err)
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		log.Fatalf("error formatting source: %v", err)
	}
	fh, err := os.Create(outDir + "/provider_generated.go")
	if err != nil {
		log.Fatalf("error opening output file: %v", err)
	}
	if _, err := fh.Write(formatted); err != nil {
		log.Fatalf("error writing output file: %v", err)
	}
	if err := fh.Close(); err != nil {
		log.Fatalf("error closing output file: %v", err)
	}
}
