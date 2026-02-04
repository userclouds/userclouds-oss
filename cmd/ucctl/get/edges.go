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

type EdgesCommand struct {
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

func (c *EdgesCommand) RunE(cmd *cobra.Command, args []string) error {
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

	// If output format is specified (and not table), fetch all data and output in that format
	if c.Output != "" && c.Output != "table" {
		return c.outputFormatted(cmd.Context(), azClient)
	}

	config := common.PagerConfig[authz.Edge]{
		Ctx: cmd.Context(),
		FetchFunc: func(ctx context.Context, cursor string, limit int) ([]authz.Edge, string, error) {
			return c.fetchEdges(ctx, azClient, cursor, limit)
		},
		TableRenderFunc: c.renderTable,
		InitialCursor:   c.Cursor,
		NoItemsMessage:  "No edges found.",
		ItemName:        "edges",
	}

	return common.RunPager(config)
}

func (c *EdgesCommand) outputFormatted(ctx context.Context, azClient *authz.Client) error {
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
		edges, nextCursor, err := c.fetchEdges(ctx, azClient, cursor, 100)
		if err != nil {
			return err
		}

		if len(edges) == 0 {
			break
		}

		// Write this batch immediately
		switch format {
		case common.OutputFormatJSON:
			if err := writer.(*common.JSONStreamWriter).WriteItems(edges); err != nil {
				return err
			}
		case common.OutputFormatYAML:
			if err := writer.(*common.YAMLStreamWriter).WriteItems(edges); err != nil {
				return err
			}
		case common.OutputFormatCSV:
			if err := writer.(*common.CSVWriter).WriteItems(edges); err != nil {
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

func (c *EdgesCommand) fetchEdges(ctx context.Context, azClient *authz.Client, cursor string, limit int) ([]authz.Edge, string, error) {
	var paginationOpts []pagination.Option

	filterStr, err := common.GetFilterString(c.Filter, c.RawFilter, common.EdgeFilterKeys)
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

	resp, err := azClient.ListEdges(ctx, opts...)
	if err != nil {
		return nil, "", ucerr.Wrap(err)
	}

	nextCursor := ""
	if resp.HasNext {
		nextCursor = string(resp.Next)
	}

	return resp.Data, nextCursor, nil
}

func (c *EdgesCommand) renderTable(edges []authz.Edge) ([]common.TableColumn, []common.TableRow) {
	columns := []common.TableColumn{
		{Header: "ID", Width: 36},               // UUIDs are always 36 characters
		{Header: "SOURCE_OBJECT_ID", Width: 36}, // UUIDs are always 36 characters
		{Header: "TARGET_OBJECT_ID", Width: 36}, // UUIDs are always 36 characters
		{Header: "EDGE_TYPE_ID", Width: 36},     // UUIDs are always 36 characters
		{Header: "CREATED", Width: 19},          // "2006-01-02 15:04:05" is 19 characters
		{Header: "UPDATED", Width: 19},          // "2006-01-02 15:04:05" is 19 characters
	}

	rows := make([]common.TableRow, len(edges))
	for i, edge := range edges {
		rows[i] = common.TableRow{
			edge.ID.String(),
			edge.SourceObjectID.String(),
			edge.TargetObjectID.String(),
			edge.EdgeTypeID.String(),
			edge.Created.Format("2006-01-02 15:04:05"),
			edge.Updated.Format("2006-01-02 15:04:05"),
		}
	}

	return columns, rows
}
