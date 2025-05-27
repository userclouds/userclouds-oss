package codegensdk

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gofrs/uuid"
	"golang.org/x/exp/slices"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uctypes/set"
	"userclouds.com/internal/multitenant"
)

// sdkGenerator is the interface that needs to be implemented for a
// particular language-specific generator
type sdkGenerator interface {
	getFormattedSource([]byte) ([]byte, error)
	getFuncArgComponent(string, bool) string
	getTemplate() string
	getTemplateData(includeExample bool) (*templateData, error)
	setTypeName(objectMemberData) objectMemberData
}

// baseSDKGenerator implements the basic algorithm for the getTemplateData
// method of sdkGenerator. A language-specific sdkGenerator should embed this
// struct and implement the other unimplemented methods of the sdkGenerator
// interface
type baseSDKGenerator struct {
	ctx                   context.Context
	s                     *storage.Storage
	accessorGetRegexp     *regexp.Regexp
	mutatorSetRegexp      *regexp.Regexp
	whereClauseArgsRegexp *regexp.Regexp
	uniqueDataTypes       map[uuid.UUID]templateDataType
	usedObjectNames       map[string]bool
}

func newBaseSDKGenerator(ctx context.Context, s *storage.Storage) baseSDKGenerator {
	return baseSDKGenerator{
		ctx:                   ctx,
		s:                     s,
		accessorGetRegexp:     regexp.MustCompile(`^(Get|Find|List)(.*)$`),
		mutatorSetRegexp:      regexp.MustCompile(`^(Set|CreateAndUpdate|Update|Create)(.*)$`),
		whereClauseArgsRegexp: regexp.MustCompile(`(?i)(any|array|\?)`),
		uniqueDataTypes:       map[uuid.UUID]templateDataType{},
		usedObjectNames:       map[string]bool{},
	}
}

func (g *baseSDKGenerator) getUniqueObjectName(name string) string {
	baseName := name
	suffix := 2
	for {
		if !g.usedObjectNames[name] {
			break
		}
		name = baseName + strconv.Itoa(suffix)
		suffix++
	}
	g.usedObjectNames[name] = true
	return name
}

// Some helper functions for converting names of accessors / mutators / column types to strings that we can use in the template
func (baseSDKGenerator) toPascalCase(s string) string {
	caser := cases.Title(language.English)

	words := strings.Split(s, "_")
	pascalWords := make([]string, len(words))
	for i, word := range words {
		word = strings.ToLower(word)
		if word == "id" {
			pascalWords[i] = "ID"
		} else {
			pascalWords[i] = caser.String(word)
		}
	}

	return strings.Join(pascalWords, "")
}

func (g *baseSDKGenerator) getterNames(accessorName string) (objectName string, functionName string) {
	functionName = strings.ReplaceAll(strings.TrimSuffix(accessorName, "Accessor"), "-", "_")

	if prefixMatch := g.accessorGetRegexp.FindStringSubmatch(functionName); prefixMatch != nil {
		objectName = prefixMatch[2] + "Object"
	} else {
		objectName = functionName + "Object"
		functionName = "Get" + objectName
	}

	objectName = g.getUniqueObjectName(objectName)

	return
}

func (g *baseSDKGenerator) setterNames(mutatorName string) (objectName string, functionName string) {
	functionName = strings.ReplaceAll(strings.TrimSuffix(mutatorName, "Mutator"), "-", "_")

	if prefixMatch := g.mutatorSetRegexp.FindStringSubmatch(functionName); prefixMatch != nil {
		objectName = prefixMatch[2] + "Object"
	} else {
		objectName = functionName + "Object"
		functionName = "Set" + objectName
	}

	objectName = g.getUniqueObjectName(objectName)

	return
}

func (g baseSDKGenerator) whereClauseArguments(whereClause string) (
	functionArguments map[string]bool,
	selectorArguments []string,
	updatedWhereClause string,
) {
	isArray := false
	selectorArguments = []string{}
	functionArguments = map[string]bool{}

	matches := g.whereClauseArgsRegexp.FindAllString(whereClause, -1)
	for _, match := range matches {
		switch strings.ToLower(match) {
		case "any":
			isArray = true
		case "array":
			isArray = false
		case "?":
			argName := fmt.Sprintf("arg%d", len(selectorArguments)+1)
			selectorArguments = append(selectorArguments, argName)
			functionArguments[argName] = isArray
			isArray = false
		}
	}

	whereParts := strings.Split(whereClause, "?")
	updatedWhereClause = whereParts[0]
	for i := 1; i < len(whereParts); i++ {
		updatedWhereClause += fmt.Sprintf("{%s}%s", selectorArguments[i-1], whereParts[i])
	}

	return
}

func (g baseSDKGenerator) loadAccessorsAndMutators(
	options ...pagination.Option,
) ([]userstore.Accessor, []userstore.Mutator, *storage.TransformerMap, error) {
	transformerIDs := set.NewUUIDSet()

	columnManager, err := storage.NewUserstoreColumnManager(g.ctx, g.s)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	pager, err := storage.NewAccessorPaginatorFromOptions(options...)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	accessors := []userstore.Accessor{}
	for {
		storageAccessors, pr, err := g.s.GetLatestAccessors(g.ctx, *pager)
		if err != nil {
			return nil, nil, nil, ucerr.Wrap(err)
		}

		for _, a := range storageAccessors {
			accessors = append(accessors, a.ToClientModel())
			for i, tid := range a.TransformerIDs {
				if !tid.IsNil() {
					transformerIDs.Insert(tid)
				} else {
					column := columnManager.GetColumnByID(a.ColumnIDs[i])
					if column == nil {
						return nil, nil, nil, ucerr.Errorf("column not found for accessor %v", a.ID)
					}
					transformerIDs.Insert(column.DefaultTransformerID)
				}
			}
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	pager, err = storage.NewMutatorPaginatorFromOptions(options...)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	mutators := []userstore.Mutator{}
	for {
		storageMutators, pr, err := g.s.GetLatestMutators(g.ctx, *pager)
		if err != nil {
			return nil, nil, nil, ucerr.Wrap(err)
		}

		for _, m := range storageMutators {
			mutators = append(mutators, m.ToClientModel())
			transformerIDs.Insert(m.NormalizerIDs...)
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	transformerMap, err := storage.GetTransformerMapForIDs(g.ctx, g.s, true, transformerIDs.Items()...)
	if err != nil {
		return nil, nil, nil, ucerr.Wrap(err)
	}

	return accessors, mutators, transformerMap, nil
}

func (g *baseSDKGenerator) getObjectMemberData(
	clientCol userstore.Column,
	transformer *storage.Transformer,
	transformerDataType uuid.UUID,
) (objectMemberData, error) {
	col := storage.NewColumnFromClient(clientCol)

	colDataType := col.DataTypeID
	if transformer.TransformType.ToClient() != policy.TransformTypePassThrough {
		colDataType = transformerDataType
	}

	dt, err := g.s.GetDataType(g.ctx, colDataType)
	if err != nil {
		return objectMemberData{}, ucerr.Wrap(err)
	}

	omd := objectMemberData{
		Name:           col.Name,
		DataType:       dt,
		IsArray:        col.IsArray,
		PascalCaseName: g.toPascalCase(col.Name),
	}

	if col.Attributes.Constraints.PartialUpdates {
		omd.MutatorValueField = "value_additions"
		omd.PascalCaseMutatorValueField = "ValueAdditions"
	} else {
		omd.MutatorValueField = "value"
		omd.PascalCaseMutatorValueField = "Value"
	}

	if dt.IsComposite() {
		tdt, found := g.uniqueDataTypes[dt.ID]
		if !found {
			tdt = templateDataType{
				Name: fmt.Sprintf("data_type_%s", dt.Name),
			}
			for _, field := range dt.CompositeAttributes.Fields {
				fdt, err := g.s.GetDataType(g.ctx, field.DataTypeID)
				if err != nil {
					return omd, ucerr.Wrap(err)
				}

				omd := objectMemberData{
					Name:           field.StructName,
					DataType:       fdt,
					PascalCaseName: field.CamelCaseName,
				}
				tdt.Fields = append(tdt.Fields, omd)
			}
			g.uniqueDataTypes[dt.ID] = tdt
		}

		omd.TypeName = tdt.Name
	}

	return omd, nil
}

func (g baseSDKGenerator) getTemplateData(includeExample bool) (*templateData, error) {

	ts := multitenant.MustGetTenantState(g.ctx)

	templateData := templateData{
		IncludeExample: includeExample,
		Origin:         ts.TenantURL.Host,
		GeneratedAt:    time.Now().UTC().Format(time.RFC3339),
	}

	limit := pagination.Limit(1000)

	// Add columns to the template data
	pager, err := storage.NewColumnPaginatorFromOptions(limit)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	columns := []userstore.Column{}
	columnMap := map[uuid.UUID]userstore.Column{}
	for {
		storageCols, pr, err := g.s.ListColumnsPaginated(g.ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		for _, c := range storageCols {
			columns = append(columns, c.ToClientModel())
		}
		for _, column := range columns {
			columnMap[column.ID] = column
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	// Add purposes to the template data
	pager, err = storage.NewPurposePaginatorFromOptions(limit)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	purposes := []userstore.Purpose{}
	for {
		storagePurposes, pr, err := g.s.ListPurposesPaginated(g.ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		for _, p := range storagePurposes {
			purposes = append(purposes, p.ToClientModel())
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	for _, purpose := range purposes {
		templateData.Purposes = append(
			templateData.Purposes,
			templatePurpose{
				Name:           purpose.Name,
				PascalCaseName: g.toPascalCase(purpose.Name),
				AllCapsName:    strings.ToUpper(purpose.Name),
			},
		)
	}

	// look up accessors, mutators, and associated transformers
	accessors, mutators, transformerMap, err := g.loadAccessorsAndMutators(limit)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	// add accessors to the template data
	templateAccessors := make([]templateAccessor, 0, len(accessors))
	for _, accessor := range accessors {
		// gather the list of arguments for the created function and the selector in ExecuteAccessor
		functionArguments, selectorArguments, whereClause :=
			g.whereClauseArguments(accessor.SelectorConfig.WhereClause)

		// process the output columns
		var objectMembers []objectMemberData
		for _, columnOutput := range accessor.Columns {
			clientCol, found := columnMap[columnOutput.Column.ID]
			if !found {
				return nil,
					ucerr.Errorf(
						"could not find column '%v' for accessor '%v'",
						columnOutput.Column.ID,
						accessor.ID,
					)
			}
			var transformerID uuid.UUID
			if !columnOutput.Transformer.ID.IsNil() {
				transformerID = columnOutput.Transformer.ID
			} else {
				transformerID = clientCol.DefaultTransformer.ID
			}
			transformer, err := transformerMap.ForID(transformerID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			omd, err := g.getObjectMemberData(
				clientCol,
				transformer,
				transformer.OutputDataTypeID,
			)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			objectMembers = append(objectMembers, omd)
		}

		slices.SortFunc(
			objectMembers,
			func(a, b objectMemberData) int {
				return strings.Compare(a.Name, b.Name)
			},
		)

		objectName, functionName := g.getterNames(accessor.Name)

		templateAccessors = append(
			templateAccessors,
			templateAccessor{
				AccessorID:        accessor.ID.String(),
				ObjectName:        objectName,
				ObjectMembers:     objectMembers,
				FunctionName:      functionName,
				FunctionArguments: functionArguments,
				SelectorArguments: selectorArguments,
				WhereClause:       whereClause,
			},
		)
	}

	slices.SortFunc(templateAccessors, func(a, b templateAccessor) int {
		return strings.Compare(a.FunctionName, b.FunctionName)
	})

	templateData.TemplateAccessors = templateAccessors

	// Add mutators to the template data
	templateMutators := make([]templateMutator, 0, len(mutators))
	for _, mutator := range mutators {
		// gather the list of arguments for the created function and the selector in ExecuteMutator
		functionArguments, selectorArguments, whereClause :=
			g.whereClauseArguments(mutator.SelectorConfig.WhereClause)

		// process the input columns
		var objectMembers []objectMemberData
		for _, columnInput := range mutator.Columns {
			clientCol, found := columnMap[columnInput.Column.ID]
			if !found {
				return nil,
					ucerr.Errorf(
						"could not find column '%v' for mutator '%v'",
						columnInput.Column.ID,
						mutator.ID,
					)
			}
			normalizer, err := transformerMap.ForID(columnInput.Normalizer.ID)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			omd, err := g.getObjectMemberData(
				clientCol,
				normalizer,
				normalizer.InputDataTypeID,
			)
			if err != nil {
				return nil, ucerr.Wrap(err)
			}
			objectMembers = append(objectMembers, omd)
		}

		slices.SortFunc(
			objectMembers,
			func(a, b objectMemberData) int {
				return strings.Compare(a.Name, b.Name)
			},
		)

		objectName, functionName := g.setterNames(mutator.Name)

		templateMutators = append(
			templateMutators,
			templateMutator{
				MutatorID:         mutator.ID.String(),
				ObjectName:        objectName,
				ObjectMembers:     objectMembers,
				FunctionName:      functionName,
				FunctionArguments: functionArguments,
				SelectorArguments: selectorArguments,
				WhereClause:       whereClause,
			},
		)
	}

	slices.SortFunc(
		templateMutators,
		func(a, b templateMutator) int {
			return strings.Compare(a.FunctionName, b.FunctionName)
		},
	)

	templateData.TemplateMutators = templateMutators

	var tdts []templateDataType
	for _, tdt := range g.uniqueDataTypes {
		tdts = append(tdts, tdt)
	}

	slices.SortFunc(
		tdts,
		func(a, b templateDataType) int {
			return strings.Compare(a.Name, b.Name)
		},
	)

	templateData.TemplateDataTypes = tdts

	return &templateData, nil
}

// generateSDK generates the SDK using the provided sdkGenerator, including an example if requested
func generateSDK(generator sdkGenerator, includeExample bool) ([]byte, error) {
	templateData, err := generator.getTemplateData(includeExample)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for i, dataType := range templateData.TemplateDataTypes {
		for j, field := range dataType.Fields {
			templateData.TemplateDataTypes[i].Fields[j] = generator.setTypeName(field)
		}
	}

	for i, accessor := range templateData.TemplateAccessors {
		funcArgComponents := []string{}
		for funcArg, isArray := range accessor.FunctionArguments {
			funcArgComponents = append(funcArgComponents, generator.getFuncArgComponent(funcArg, isArray))
		}
		templateData.TemplateAccessors[i].setFunctionArgumentString(strings.Join(funcArgComponents, ", "))

		templateData.TemplateAccessors[i].SelectorArgumentString = strings.Join(accessor.SelectorArguments, ", ")

		for j, member := range accessor.ObjectMembers {
			templateData.TemplateAccessors[i].ObjectMembers[j] = generator.setTypeName(member)
		}
	}

	for i, mutator := range templateData.TemplateMutators {
		funcArgComponents := []string{}
		for funcArg, isArray := range mutator.FunctionArguments {
			funcArgComponents = append(funcArgComponents, generator.getFuncArgComponent(funcArg, isArray))
		}
		templateData.TemplateMutators[i].FunctionArgumentString = strings.Join(funcArgComponents, ", ")

		templateData.TemplateMutators[i].SelectorArgumentString = strings.Join(mutator.SelectorArguments, ", ")

		for j, member := range mutator.ObjectMembers {
			templateData.TemplateMutators[i].ObjectMembers[j] = generator.setTypeName(member)
		}
	}

	temp := template.Must(template.New("handlers").Delims("<<", ">>").Parse(generator.getTemplate()))

	var bs []byte
	buf := bytes.NewBuffer(bs)
	if err := temp.Execute(buf, templateData); err != nil {
		return nil, ucerr.Wrap(err)
	}

	output, err := generator.getFormattedSource(buf.Bytes())
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	return output, nil
}
