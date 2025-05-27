package genpageable

import (
	"context"
	"fmt"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"

	"userclouds.com/infra/uclog"
	"userclouds.com/tools/generate"
)

type data struct {
	Package string
	Type    string

	HasGetCursor         bool
	HasGetPaginationKeys bool
}

// Run implements the generator interface
func Run(ctx context.Context, p *packages.Package, path string, args ...string) {

	data := data{
		Type:    args[1],
		Package: p.Name,
	}

	dataType := p.Types.Scope().Lookup(data.Type)
	if dataType == nil {
		uclog.Fatalf(ctx, "couldn't find type %s in package scope", data.Type)
	}

	s, ok := dataType.Type().Underlying().(*types.Struct)
	if !ok {
		uclog.Fatalf(ctx, "type %s doesn't appear to be a struct", data.Type)
	}

	hasIDField := false
	for i := range s.NumFields() {
		f := s.Field(i)
		fieldName := f.Name()
		fieldTypeName := f.Type().String()

		if fieldName == "ID" ||
			fieldTypeName == "userclouds.com/infra/ucdb.BaseModel" ||
			fieldTypeName == "userclouds.com/infra/ucdb.SystemAttributeBaseModel" ||
			fieldTypeName == "userclouds.com/infra/ucdb.UserBaseModel" ||
			fieldTypeName == "userclouds.com/infra/ucdb.VersionBaseModel" {
			hasIDField = true
			break
		}
	}

	if !hasIDField {
		uclog.Fatalf(ctx, "type %s does not have an ID field", data.Type)
	}

	if generate.HasMethod(dataType.Type(), "getCursor") {
		data.HasGetCursor = true
	}

	if generate.HasMethod(dataType.Type(), "getPaginationKeys") {
		data.HasGetPaginationKeys = true
	}

	// actually write the template to a file
	fn := filepath.Join(path, fmt.Sprintf("%s_pageable_generated.go", strings.ToLower(data.Type)))
	generate.WriteFileIfChanged(ctx, fn, templateString, data)
}

var templateString = `// NOTE: automatically generated file -- DO NOT EDIT

package << .Package >>

import (
	"fmt"
	"net/http"

	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

// GetCursor is part of the pagination.PageableType interface
func (o << .Type >>) GetCursor(k pagination.Key) pagination.Cursor {
	if k == "id" {
		return pagination.Cursor(fmt.Sprintf("id:%v", o.GetID()))
	}
	<<- if .HasGetCursor >>
	// .getCursor() lets you generate a cursor for additional sort keys
	cursor := pagination.CursorBegin
	o.getCursor(k, &cursor)
	return cursor
	<<- else >>
	return pagination.CursorBegin
	<<- end >>
}

// GetPaginationKeys is part of the pagination.PageableType interface
func (o << .Type >>) GetPaginationKeys() pagination.KeyTypes {
	<<- if .HasGetPaginationKeys >>
	// .getPaginationKeys() lets you add additional supported pagination keys
	keyTypes := o.getPaginationKeys()
	<<- else >>
	keyTypes := pagination.KeyTypes{}
	<<- end >>
	keyTypes["id"] = pagination.UUIDKeyType
	return keyTypes
}

// New<< .Type >>PaginatorFromOptions generates a paginator for a << .Type >>
func New<< .Type >>PaginatorFromOptions(
        options ...pagination.Option,
) (*pagination.Paginator, error) {
        var resultType << .Type >>
        options = append(options, pagination.ResultType(resultType))
        pager, err := pagination.ApplyOptions(options...)
        if err != nil {
                return nil, ucerr.Wrap(err)
        }

	if cursor := resultType.GetCursor(pager.GetSortKey()); cursor == pagination.CursorBegin {
		return nil, ucerr.Friendlyf(nil, "sort key '%s' is unsupported", pager.GetSortKey())
	}

        return pager, nil
}

// New<< .Type >>PaginatorFromQuery generates a paginator for a << .Type >>
func New<< .Type >>PaginatorFromQuery(
        query pagination.Query,
        defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
        var resultType << .Type >>
        defaultOptions = append(defaultOptions, pagination.ResultType(resultType))
        pager, err := pagination.NewPaginatorFromQuery(query, defaultOptions...)
        if err != nil {
                return nil, ucerr.Wrap(err)
        }

	if cursor := resultType.GetCursor(pager.GetSortKey()); cursor == pagination.CursorBegin {
		return nil, ucerr.Friendlyf(nil, "sort key '%s' is unsupported", pager.GetSortKey())
	}

        return pager, nil
}

// New<< .Type >>PaginatorFromRequest generates a paginator and cursor maker for a << .Type >>
func New<< .Type >>PaginatorFromRequest(
        r *http.Request,
        defaultOptions ...pagination.Option,
) (*pagination.Paginator, error) {
        var resultType << .Type >>
        defaultOptions = append(defaultOptions, pagination.ResultType(resultType))
        pager, err := pagination.NewPaginatorFromRequest(r, defaultOptions...)
        if err != nil {
                return nil, ucerr.Wrap(err)
        }

	if cursor := resultType.GetCursor(pager.GetSortKey()); cursor == pagination.CursorBegin {
		return nil, ucerr.Friendlyf(nil, "sort key '%s' is unsupported", pager.GetSortKey())
	}

        return pager, nil
}
`
