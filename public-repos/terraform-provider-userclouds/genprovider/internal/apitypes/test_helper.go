package apitypes

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"testing"
	"text/template"
)

type data struct {
	T                      APIType
	ExtraTFAttributeFields map[string]string
	SampleJSONClientValue  string
	SampleTFModelValue     string
	ExtraCode              string
}

var temp = `
package main

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"regexp"

	"github.com/gofrs/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"userclouds.com/infra/ucerr"
)

// disable unused import errors -- some tests use these packages, but others don't
var _ attr.Value
var _ = context.Background
var _ = fmt.Printf
var _ = uuid.FromString
var _ = regexp.MustCompile
var _ validator.String
var _ = stringvalidator.RegexMatches
var _ basetypes.StringValue
var _ = ucerr.Errorf
var _ = []planmodifier.String{}
var _ = stringplanmodifier.RequiresReplace
var _ = int64planmodifier.RequiresReplace
var _ = float64planmodifier.RequiresReplace
var _ = boolplanmodifier.RequiresReplace
var _ = listplanmodifier.RequiresReplace
var _ = objectplanmodifier.RequiresReplace
var _ = mapplanmodifier.RequiresReplace

<< .ExtraCode >>

var TFSchema = map[string]schema.Attribute{
	"sample": << .T.TFSchemaAttributeText .ExtraTFAttributeFields >>,
}

var TFSchemaAttrTypeMapName = map[string]attr.Type{
	"sample": << .T.TFSchemaAttributeType >>,
}

type TFModel struct {
	sample << .T.TFModelType >>
}

type JSONClientModel struct {
	sample << .T.JSONClientModelType >>
}

func main() {
	jsonclientVal := << .SampleJSONClientValue >>
	tfVal := << .SampleTFModelValue >>

	jsonclientConverted, err := << .T.TFModelToJSONClientFunc >>(&tfVal)
	if err != nil {
		log.Fatalf("failed to convert TF to jsonclient: %v", err)
	}
	if !reflect.DeepEqual(jsonclientVal, *jsonclientConverted) {
		log.Fatalf("tf->jsonclient conversion yielded %#v, which does not match expected %#v", *jsonclientConverted, jsonclientVal)
	}

	tfConverted, err := << .T.JSONClientModelToTFFunc >>(&jsonclientVal)
	if err != nil {
		log.Fatalf("failed to convert jsonclient to TF: %v", err)
	}
	if !reflect.DeepEqual(tfVal, tfConverted) {
		log.Fatalf("jsonclient->tf conversion yielded %#v, which does not match expected %#v", tfConverted, tfVal)
	}
}
`

func runTestProgram(t *testing.T, d data) {
	dname, err := os.MkdirTemp("", "apitypes_test")
	if err != nil {
		t.Fatalf("error creating temp dir: %v", err)
	}

	err = os.WriteFile(dname+"/go.mod", []byte(`
		module apitypes_test
		go 1.20
		require (
			github.com/hashicorp/terraform-plugin-framework v1.3.3
			userclouds.com v0.7.6
		)
		require (
			github.com/fatih/color v1.15.0 // indirect
			github.com/gofrs/uuid v4.4.0+incompatible
			github.com/hashicorp/go-hclog v1.5.0 // indirect
			github.com/hashicorp/terraform-plugin-framework-validators v0.11.0
			github.com/hashicorp/terraform-plugin-go v0.18.0 // indirect
			github.com/hashicorp/terraform-plugin-log v0.9.0 // indirect
			github.com/mattn/go-colorable v0.1.13 // indirect
			github.com/mattn/go-isatty v0.0.17 // indirect
			github.com/mitchellh/go-testing-interface v1.14.1 // indirect
			github.com/vmihailenco/msgpack/v5 v5.3.5 // indirect
			github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
			golang.org/x/sys v0.13.0 // indirect
		)
	`), 0644)
	if err != nil {
		t.Fatalf("error writing go.mod: %v", err)
	}

	err = os.WriteFile(dname+"/go.sum", []byte(`
		github.com/davecgh/go-spew v1.1.0/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
		github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
		github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
		github.com/fatih/color v1.13.0/go.mod h1:kLAiJbzzSOZDVNGyDpeOxJ47H46qBXwg5ILebYFFOfk=
		github.com/fatih/color v1.15.0 h1:kOqh6YHBtK8aywxGerMG2Eq3H6Qgoqeo13Bk2Mv/nBs=
		github.com/fatih/color v1.15.0/go.mod h1:0h5ZqXfHYED7Bhv2ZJamyIOUej9KtShiJESRwBDUSsw=
		github.com/gofrs/uuid v4.4.0+incompatible h1:3qXRTX8/NbyulANqlc0lchS1gqAVxRgsuW1YrTJupqA=
		github.com/gofrs/uuid v4.4.0+incompatible/go.mod h1:b2aQJv3Z4Fp6yNu3cdSllBxTCLRxnplIgP/c0N/04lM=
		github.com/google/go-cmp v0.6.0 h1:ofyhxvXcZhMsU5ulbFiLKl/XBFqE1GSq7atu8tAmTRI=
		github.com/hashicorp/go-hclog v1.5.0 h1:bI2ocEMgcVlz55Oj1xZNBsVi900c7II+fWDyV9o+13c=
		github.com/hashicorp/go-hclog v1.5.0/go.mod h1:W4Qnvbt70Wk/zYJryRzDRU/4r0kIg0PVHBcfoyhpF5M=
		github.com/hashicorp/terraform-plugin-framework v1.3.3 h1:D18BlA8gdV4+W8WKhUqxudiYomPZHv94FFzyoSCKC8Q=
		github.com/hashicorp/terraform-plugin-framework v1.3.3/go.mod h1:2gGDpWiTI0irr9NSTLFAKlTi6KwGti3AoU19rFqU30o=
		github.com/hashicorp/terraform-plugin-framework-validators v0.11.0 h1:DKb1bX7/EPZUTW6F5zdwJzS/EZ/ycVD6JAW5RYOj4f8=
		github.com/hashicorp/terraform-plugin-framework-validators v0.11.0/go.mod h1:dzxOiHh7O9CAwc6p8N4mR1H++LtRkl+u+21YNiBVNno=
		github.com/hashicorp/terraform-plugin-go v0.18.0 h1:IwTkOS9cOW1ehLd/rG0y+u/TGLK9y6fGoBjXVUquzpE=
		github.com/hashicorp/terraform-plugin-go v0.18.0/go.mod h1:l7VK+2u5Kf2y+A+742GX0ouLut3gttudmvMgN0PA74Y=
		github.com/hashicorp/terraform-plugin-log v0.9.0 h1:i7hOA+vdAItN1/7UrfBqBwvYPQ9TFvymaRGZED3FCV0=
		github.com/hashicorp/terraform-plugin-log v0.9.0/go.mod h1:rKL8egZQ/eXSyDqzLUuwUYLVdlYeamldAHSxjUFADow=
		github.com/mattn/go-colorable v0.1.9/go.mod h1:u6P/XSegPjTcexA+o6vUJrdnUu04hMope9wVRipJSqc=
		github.com/mattn/go-colorable v0.1.12/go.mod h1:u5H1YNBxpqRaxsYJYSkiCWKzEfiAb1Gb520KVy5xxl4=
		github.com/mattn/go-colorable v0.1.13 h1:fFA4WZxdEF4tXPZVKMLwD8oUnCTTo08duU7wxecdEvA=
		github.com/mattn/go-colorable v0.1.13/go.mod h1:7S9/ev0klgBDR4GtXTXX8a3vIGJpMovkB8vQcUbaXHg=
		github.com/mattn/go-isatty v0.0.12/go.mod h1:cbi8OIDigv2wuxKPP5vlRcQ1OAZbq2CE4Kysco4FUpU=
		github.com/mattn/go-isatty v0.0.14/go.mod h1:7GGIvUiUoEMVVmxf/4nioHXj79iQHKdU27kJ6hsGG94=
		github.com/mattn/go-isatty v0.0.16/go.mod h1:kYGgaQfpe5nmfYZH+SKPsOc2e4SrIfOl2e/yFXSvRLM=
		github.com/mattn/go-isatty v0.0.17 h1:BTarxUcIeDqL27Mc+vyvdWYSL28zpIhv3RoTdsLMPng=
		github.com/mattn/go-isatty v0.0.17/go.mod h1:kYGgaQfpe5nmfYZH+SKPsOc2e4SrIfOl2e/yFXSvRLM=
		github.com/mitchellh/go-testing-interface v1.14.1 h1:jrgshOhYAUVNMAJiKbEu7EqAwgJJ2JqpQmpLJOu07cU=
		github.com/mitchellh/go-testing-interface v1.14.1/go.mod h1:gfgS7OtZj6MA4U1UrDRp04twqAjfvlZyCfX3sDjEym8=
		github.com/pmezard/go-difflib v1.0.0 h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=
		github.com/pmezard/go-difflib v1.0.0/go.mod h1:iKH77koFhYxTK1pcRnkKkqfTogsbg7gZNVY4sRDYZ/4=
		github.com/stretchr/objx v0.1.0/go.mod h1:HFkY916IF+rwdDfMAkV7OtwuqBVzrE8GR6GFx+wExME=
		github.com/stretchr/testify v1.6.1/go.mod h1:6Fq8oRcR53rry900zMqJjRRixrwX3KX962/h/Wwjteg=
		github.com/stretchr/testify v1.7.2 h1:4jaiDzPyXQvSd7D0EjG45355tLlV3VOECpq10pLC+8s=
		github.com/stretchr/testify v1.7.2/go.mod h1:R6va5+xMeoiuVRoj+gSkQ7d3FALtqAAGI1FQKckRals=
		github.com/vmihailenco/msgpack/v5 v5.3.5 h1:5gO0H1iULLWGhs2H5tbAHIZTV8/cYafcFOr9znI5mJU=
		github.com/vmihailenco/msgpack/v5 v5.3.5/go.mod h1:7xyJ9e+0+9SaZT0Wt1RGleJXzli6Q/V5KbhBonMG9jc=
		github.com/vmihailenco/tagparser/v2 v2.0.0 h1:y09buUbR+b5aycVFQs/g70pqKVZNBmxwAhO7/IwNM9g=
		github.com/vmihailenco/tagparser/v2 v2.0.0/go.mod h1:Wri+At7QHww0WTrCBeu4J6bNtoV6mEfg5OIWRZA9qds=
		golang.org/x/sys v0.0.0-20200116001909-b77594299b42/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
		golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae/go.mod h1:h1NjWce9XRLGQEsW7wpKNCjG9DtNlClVuFLEZdDNbEs=
		golang.org/x/sys v0.0.0-20210630005230-0f9fa26af87c/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
		golang.org/x/sys v0.0.0-20210927094055-39ccf1dd6fa6/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
		golang.org/x/sys v0.0.0-20220503163025-988cb79eb6c6/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
		golang.org/x/sys v0.0.0-20220811171246-fbc7d0a398ab/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
		golang.org/x/sys v0.13.0 h1:Af8nKPmuFypiUBjVoU9V20FiaFXOcuZI21p0ycVYYGE=
		golang.org/x/sys v0.13.0/go.mod h1:oPkhp1MJrh7nUepCBck5+mAzfO9JrbApNNgaTdGDITg=
		gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
		gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
		gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
		gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
		userclouds.com v0.7.6 h1:3WkIjjsuzKkGeyPGROvnpgURyxBxurl6B8H/dAZx4Kk=
		userclouds.com v0.7.6/go.mod h1:+PLT4agT4RqUMZaQESBR8mCnsW8IKJdi9tq96vx738c=
	`), 0644)
	if err != nil {
		t.Fatalf("error writing go.mod: %v", err)
	}

	buf := bytes.NewBuffer([]byte{})
	temp := template.Must(template.New("temp").Delims("<<", ">>").Parse(temp))
	if err := temp.Execute(buf, d); err != nil {
		t.Fatalf("error executing template: %v", err)
	}
	bs := buf.Bytes()

	formatted, err := format.Source(bs)
	if err != nil {
		t.Fatalf("error formatting source: %v\n\n%s", err, string(bs))
	}

	outPath := dname + "/sampleprogram.go"
	fh, err := os.Create(outPath)
	if err != nil {
		t.Fatalf("error opening output file: %v", err)
	}
	fmt.Printf("writing test file to %s\n", outPath)
	if _, err = fh.Write(formatted); err != nil {
		t.Fatalf("error writing to output file: %v", err)
	}
	if err = fh.Close(); err != nil {
		t.Fatalf("error closing file: %v", err)
	}

	fmt.Printf("running sample program...\n")
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = dname
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", out)
		t.Fatalf("error running test program: %v", err)
	}
}
