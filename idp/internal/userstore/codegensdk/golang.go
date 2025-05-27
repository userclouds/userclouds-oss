package codegensdk

import (
	"context"
	"fmt"
	"go/format"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/userstore/datatype"
	"userclouds.com/infra/ucerr"
)

type golangSDKGenerator struct {
	baseSDKGenerator
}

func newGolangSDKGenerator(ctx context.Context, s *storage.Storage) sdkGenerator {
	return golangSDKGenerator{
		baseSDKGenerator: newBaseSDKGenerator(ctx, s),
	}
}

func (golangSDKGenerator) getFormattedSource(b []byte) ([]byte, error) {
	output, err := format.Source(b)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return output, nil
}

func (golangSDKGenerator) getFuncArgComponent(
	funcArg string,
	isArray bool,
) string {
	if isArray {
		return fmt.Sprintf("%s []string", funcArg)
	}
	return fmt.Sprintf("%s string", funcArg)
}

func (golangSDKGenerator) getTemplate() string {
	return `// This file is auto-generated and will be overwritten when a schema is modified.
// DO NOT EDIT.
//
// Origin: << .Origin >>
// Generated at: << .GeneratedAt >>

package codegensdk

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"userclouds.com/idp"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
)

type UCClient struct {
	*idp.Client
}

func NewUCClient(ctx context.Context, url string, opts ...idp.Option) (*UCClient, error) {
	client, err := idp.NewClient(url, opts...)
	if err != nil {
		return nil, err
	}

	return &UCClient{
		Client: client,
	}, nil
}

// composite value types

<< range .TemplateDataTypes >>

type << .Name >> struct {
<< range .Fields >>
	<< .PascalCaseName >> << .TypeName >> ` + "`json:\"<< .Name >>,omitempty\"`" + `
<<- end >>
}
<<- end >>

// accessors

<< range .TemplateAccessors >>

type << .ObjectName >> struct {
<< range .ObjectMembers >>
	<< .PascalCaseName >> << .TypeName >> ` + "`json:\"<< .Name >>\"`" + `
<<- end >>
}

// << .FunctionName >>
// selector: "<< .WhereClause >>"

func (c *UCClient) << .FunctionName >>(ctx context.Context, << .FunctionArgumentString >>opts ...idp.Option) ([]<< .ObjectName >>, error) {
	resp, err := c.ExecuteAccessor(
		ctx,
		uuid.Must(uuid.FromString("<< .AccessorID >>")),
		policy.ClientContext{},
		[]any{<< .SelectorArgumentString >>},
		opts...,
	)
	if err != nil {
		return nil, err
	}

	var objects []<< .ObjectName >>
	for _, value := range resp.Data {
		var m map[string]string
		if err := json.Unmarshal([]byte(value), &m); err != nil {
			return nil, err
		}

		o := << .ObjectName >>{}
		<<- range .ObjectMembers >>
		if m["<< .Name >>"] != "" {
			<<- if .IsArray >>
			if err = json.Unmarshal([]byte(m["<< .Name >>"]), &o.<< .PascalCaseName >>); err != nil {
				return nil, err
			}
			<<- else if eq .TypeName "string" >>
			o.<< .PascalCaseName >> = m["<< .Name >>"]
			<<- else if eq .TypeName "bool" >>
			if o.<< .PascalCaseName >>, err = strconv.ParseBool(m["<< .Name >>"]); err != nil {
				return nil, err
			}
			<<- else if eq .TypeName "int" >>
			if o.<< .PascalCaseName >>, err = strconv.Atoi(m["<< .Name >>"]); err != nil {
				return nil, err
			}
			<<- else if eq .TypeName "time.Time" >>
			if o.<< .PascalCaseName >>, err = time.Parse(time.RFC3339, m["<< .Name >>"]); err != nil {
				return nil, err
			}
			<<- else if eq .TypeName "uuid.UUID" >>
			if o.<< .PascalCaseName >>, err = uuid.FromString(m["<< .Name >>"]); err != nil {
				return nil, err
			}
			<<- else >>
			if err = json.Unmarshal([]byte(m["<< .Name >>"]), &o.<< .PascalCaseName >>); err != nil {
				return nil, err
			}
			<<- end >>
		}
		<<- end >>

		objects = append(objects, o)
	}

	return objects, nil
}

<< end >>

// mutators

<<- $mutators := .TemplateMutators >>
<<- $purposes := .Purposes >>

<<- range $mutator := $mutators >>

type << $mutator.ObjectName >> struct {
<<- range $mutator.ObjectMembers >>
	<< .PascalCaseName >> << .TypeName >> ` + "`json:\"<< .Name >>\"`" + `
<<- end >>
}

<<- range $purpose := $purposes >>

// << $mutator.FunctionName >>For<< $purpose.PascalCaseName >>Purpose
// selector: "<< $mutator.WhereClause >>"

func (c *UCClient) << $mutator.FunctionName >>For<< $purpose.PascalCaseName >>Purpose(ctx context.Context, o << $mutator.ObjectName >>, << $mutator.FunctionArgumentString >>) ([]uuid.UUID, error) {
	rowData := map[string]idp.ValueAndPurposes{}
	<<- range $mutator.ObjectMembers >>
	rowData["<< .Name >>"] = idp.ValueAndPurposes{
		<< .PascalCaseMutatorValueField >>: o.<< .PascalCaseName >>,
		PurposeAdditions: []userstore.ResourceID{{Name: "<< $purpose.Name >>"}},
	}
	<<- end >>

	resp, err := c.ExecuteMutator(
		ctx,
		uuid.Must(uuid.FromString("<< $mutator.MutatorID >>")),
		policy.ClientContext{},
		[]any{<< $mutator.SelectorArgumentString >>},
		rowData,
	)
	if err != nil {
		return nil, err
	}

	return resp.UserIDs, nil
}

<<- end >>
<<- end >>
`
}

func (golangSDKGenerator) setTypeName(omd objectMemberData) objectMemberData {
	switch omd.DataType.ConcreteDataTypeID {
	case datatype.String.ID:
		omd.TypeName = "string"
	case datatype.Boolean.ID:
		omd.TypeName = "bool"
	case datatype.Integer.ID:
		omd.TypeName = "int"
	case datatype.Date.ID, datatype.Timestamp.ID:
		omd.TypeName = "time.Time"
	case datatype.UUID.ID:
		omd.TypeName = "uuid.UUID"
	case datatype.Composite.ID:
		// the passed in type name is used
	}

	if omd.IsArray {
		omd.TypeName = "[]" + omd.TypeName
	}

	return omd
}

// CodegenGolangSDK generates a golang SDK for the given IDP client
func CodegenGolangSDK(ctx context.Context, s *storage.Storage) ([]byte, error) {
	sdkGenerator := newGolangSDKGenerator(ctx, s)
	output, err := generateSDK(sdkGenerator, false)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return output, nil
}
