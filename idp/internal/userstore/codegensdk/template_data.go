package codegensdk

import (
	"fmt"

	"userclouds.com/idp/internal/storage/column"
)

// A bunch of structs to hold the data that we need to pass to the template

type objectMemberData struct {
	Name                        string
	PascalCaseName              string
	DataType                    *column.DataType
	TypeName                    string
	IsArray                     bool
	MutatorValueField           string
	PascalCaseMutatorValueField string
}

type templateDataType struct {
	Name   string
	Fields []objectMemberData
}

type templateAccessor struct {
	AccessorID             string
	ObjectName             string
	ObjectMembers          []objectMemberData
	FunctionName           string
	FunctionArguments      map[string]bool
	FunctionArgumentString string
	SelectorArguments      []string
	SelectorArgumentString string
	WhereClause            string
}

func (ta *templateAccessor) setFunctionArgumentString(fas string) {
	if fas != "" {
		fas = fmt.Sprintf("%s, ", fas)
	}

	ta.FunctionArgumentString = fas
}

type templateMutator struct {
	MutatorID              string
	ObjectName             string
	ObjectMembers          []objectMemberData
	FunctionName           string
	FunctionArguments      map[string]bool
	FunctionArgumentString string
	SelectorArguments      []string
	SelectorArgumentString string
	WhereClause            string
}

type templatePurpose struct {
	Name           string
	PascalCaseName string
	AllCapsName    string
}

type templateData struct {
	Purposes          []templatePurpose
	TemplateDataTypes []templateDataType
	TemplateAccessors []templateAccessor
	TemplateMutators  []templateMutator
	IncludeExample    bool
	Origin            string
	GeneratedAt       string
}
