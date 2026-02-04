package get

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"userclouds.com/authz"
	"userclouds.com/cmd/ucctl/common"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
)

type ObjectsCommand struct {
	URL             string
	ClientID        string
	ClientSecret    string
	ClientSecretVar string
	AuthnType       string
	Limit           int
	Cursor          string
	NoPager         bool
	Filter          string
	RawFilter       string
	Output          string

	credentials *common.Credentials
}

func (c *ObjectsCommand) RunE(cmd *cobra.Command, args []string) error {
	// Validate output format if specified
	if c.Output != "" && c.Output != "table" {
		if err := common.ValidateOutputFormat(c.Output); err != nil {
			return err
		}
	}

	var err error
	c.credentials, err = common.LoadAndSetCredentials(c.URL, c.ClientID, c.ClientSecret, c.ClientSecretVar)
	if err != nil {
		return err
	}

	azClient, err := c.credentials.NewAuthzClient()
	if err != nil {
		return err
	}

	if c.Output != "" && c.Output != "table" {
		return c.outputFormatted(cmd.Context(), azClient)
	}

	config := common.PagerConfig[authz.Object]{
		Ctx: cmd.Context(),
		FetchFunc: func(ctx context.Context, cursor string, limit int) ([]authz.Object, string, error) {
			return c.fetchObjects(ctx, azClient, cursor, limit)
		},
		TableRenderFunc: c.renderTable,
		InitialCursor:   c.Cursor,
		NoItemsMessage:  "No objects found.",
		ItemName:        "objects",
	}

	return common.RunPager(config)
}

func (c *ObjectsCommand) outputFormatted(ctx context.Context, azClient *authz.Client) error {
	cursor := c.Cursor
	format := common.OutputFormat(c.Output)

	var writer interface{}
	switch format {
	case common.OutputFormatJSON:
		writer = common.NewJSONStreamWriter()
	case common.OutputFormatYAML:
		w := common.NewYAMLStreamWriter()
		defer w.Close()
		writer = w
	case common.OutputFormatCSV:
		writer = common.NewCSVWriter()
	default:
		return fmt.Errorf("unsupported output format: %s", c.Output)
	}

	// Stream results as they come in
	for {
		objects, nextCursor, err := c.fetchObjects(ctx, azClient, cursor, 100)
		if err != nil {
			return err
		}

		if len(objects) == 0 {
			break
		}

		// Write this batch immediately
		switch format {
		case common.OutputFormatJSON:
			if err := writer.(*common.JSONStreamWriter).WriteItems(objects); err != nil {
				return err
			}
		case common.OutputFormatYAML:
			if err := writer.(*common.YAMLStreamWriter).WriteItems(objects); err != nil {
				return err
			}
		case common.OutputFormatCSV:
			if err := writer.(*common.CSVWriter).WriteItems(objects); err != nil {
				return err
			}
		}

		if nextCursor == "" {
			break
		}
		cursor = nextCursor
	}

	return nil
}

func (c *ObjectsCommand) fetchObjects(ctx context.Context, azClient *authz.Client, cursor string, limit int) ([]authz.Object, string, error) {
	var paginationOpts []pagination.Option

	filterStr, err := common.GetFilterString(c.Filter, c.RawFilter, common.ObjectFilterKeys)
	if err != nil {
		return nil, "", err
	}
	if filterStr != "" {
		paginationOpts = append(paginationOpts, pagination.Filter(filterStr))
	}

	if cursor != "" {
		paginationOpts = append(paginationOpts, pagination.StartingAfter(pagination.Cursor(cursor)))
	}
	if limit > 0 {
		paginationOpts = append(paginationOpts, pagination.Limit(limit))
	}

	var opts []authz.Option
	if len(paginationOpts) > 0 {
		opts = append(opts, authz.Pagination(paginationOpts...))
	}

	resp, err := azClient.ListObjects(ctx, opts...)
	if err != nil {
		return nil, "", ucerr.Wrap(err)
	}

	nextCursor := ""
	if resp.HasNext {
		nextCursor = string(resp.Next)
	}

	return resp.Data, nextCursor, nil
}

func (c *ObjectsCommand) renderTable(objects []authz.Object) ([]common.TableColumn, []common.TableRow) {
	columns := []common.TableColumn{
		{Header: "ID", Width: 36},              // UUIDs are always 36 characters
		{Header: "ALIAS", Width: 0},            // Dynamic width for aliases
		{Header: "TYPE_ID", Width: 36},         // UUIDs are always 36 characters
		{Header: "ORGANIZATION_ID", Width: 36}, // UUIDs are always 36 characters
		{Header: "CREATED", Width: 19},         // "2006-01-02 15:04:05" is 19 characters
		{Header: "UPDATED", Width: 19},         // "2006-01-02 15:04:05" is 19 characters
	}

	rows := make([]common.TableRow, len(objects))
	for i, obj := range objects {
		var alias string
		if obj.Alias != nil {
			alias = *obj.Alias
		}
		rows[i] = common.TableRow{
			obj.ID.String(),
			alias,
			obj.TypeID.String(),
			obj.OrganizationID.String(),
			obj.Created.Format("2006-01-02 15:04:05"),
			obj.Updated.Format("2006-01-02 15:04:05"),
		}
	}

	return columns, rows
}
