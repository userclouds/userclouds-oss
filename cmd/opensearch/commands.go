package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/gofrs/uuid"

	userstoresearch "userclouds.com/idp/userstore/search"
	"userclouds.com/infra/namespace/region"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

type indexList struct {
}

func (c *indexList) Run(helper *cmdHelper) error {
	indices, err := helper.searchMgr.ListIndices()
	if err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(helper.ctx, "found %d indices in tenant %v:", len(indices), helper.tenantID)
	for _, index := range indices {
		uclog.Infof(helper.ctx, "index: '%+v'", index)
	}

	uclog.Infof(helper.ctx, "--------------------------------")

	client, err := helper.getSearchClient()
	if err != nil {
		return ucerr.Wrap(err)
	}
	metadata, err := client.ListIndices(helper.ctx)
	if err != nil {
		return ucerr.Wrap(err)
	}

	uclog.Infof(helper.ctx, "index list from opensearch: '%v'", metadata)

	return nil
}

type accessorRemove struct {
	AccessorID uuid.UUID `arg:"" name:"accessor_id" help:"Accessor ID."`
}

func (c *accessorRemove) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.RemoveAccessorIndex(c.AccessorID))
}

type accessorSet struct {
	AccessorID uuid.UUID `arg:"" name:"accessor_id" help:"Accessor ID."`
	IndexID    uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
	QueryType  string    `arg:"" name:"query_type" help:"Query type."`
}

func (c *accessorSet) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.SetAccessorIndex(c.AccessorID, c.IndexID, userstoresearch.QueryType(c.QueryType)))
}

type indexCreate struct {
	Name          string      `arg:"" name:"name" help:"Name."`
	Description   string      `arg:"" name:"description" help:"Description."`
	IndexType     string      `arg:"" name:"index_type" help:"Index type."`
	IndexSettings string      `arg:"" name:"index_settings" help:"Index settings."`
	ColumnIDs     []uuid.UUID `arg:"" name:"column_ids" help:"Column IDs."`
}

func getIndexSettings(settings string) (userstoresearch.IndexSettings, error) {
	var indexSettings userstoresearch.IndexSettings
	if err := json.Unmarshal([]byte(settings), &indexSettings); err != nil {
		return userstoresearch.IndexSettings{}, ucerr.Wrap(err)
	}
	if err := indexSettings.Validate(); err != nil {
		return userstoresearch.IndexSettings{}, ucerr.Wrap(err)
	}
	return indexSettings, nil
}

func (c *indexCreate) Run(helper *cmdHelper) error {
	indexType := userstoresearch.IndexType(c.IndexType)
	indexSettings, err := getIndexSettings(c.IndexSettings)
	if err != nil {
		return ucerr.Wrap(err)
	}
	indexID, err := helper.searchMgr.CreateIndex(c.Name, c.Description, indexType, indexSettings, c.ColumnIDs...)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(helper.ctx, "created index '%v'", indexID)
	return nil
}

type indexContinueBootstrap struct {
	IndexID   uuid.UUID         `arg:"" name:"index_id" help:"Index ID."`
	Region    region.DataRegion `arg:"" name:"region" help:"Region."`
	BatchSize int               `help:"Batch size. default value (0) will defaultValuesPerBootstrapBatch const, which is 10k" default:"0"`
}

func (c *indexContinueBootstrap) Run(helper *cmdHelper) error {
	if err := c.Region.Validate(); err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(helper.searchMgr.ContinueIndexBootstrap(c.IndexID, region.DataRegion(c.Region), c.BatchSize))
}

type indexDelete struct {
	IndexID uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
}

func (c *indexDelete) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.DeleteIndex(c.IndexID))
}

type indexGet struct {
	IndexID uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
}

func (c *indexGet) Run(helper *cmdHelper) error {
	index, err := helper.searchMgr.GetIndex(c.IndexID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(helper.ctx, "index: '%+v'", index)
	return nil
}

type indexMetadata struct {
	IndexID    uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
	OutputFile string    `help:"Output file."`
}

func (c *indexMetadata) Run(helper *cmdHelper) error {
	index, err := helper.searchMgr.GetIndex(c.IndexID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	client, err := helper.getSearchClient()
	if err != nil {
		return ucerr.Wrap(err)
	}
	metadata, err := client.IndexMetadata(helper.ctx, index.GetIndexName(helper.tenantID))
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(writeOrLogJSONResult(helper.ctx, "index metadata", c.OutputFile, metadata))
}

type indexQuery struct {
	IndexID   uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
	TermKey   string    `arg:"" name:"term_key" help:"Term key."`
	Query     string    `arg:"" name:"query" help:"Query."`
	QueryType string    `arg:"" name:"query_type" help:"Query type."`
}

func (c *indexQuery) Run(helper *cmdHelper) error {
	queryType := "term"
	if c.QueryType != "" {
		queryType = c.QueryType
	}
	index, err := helper.searchMgr.GetIndex(c.IndexID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	client, err := helper.getSearchClient()
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(executeQuery(helper.ctx, client, index.GetIndexName(helper.tenantID), c.TermKey, c.Query, queryType))
}

type indexSetColumns struct {
	IndexID   uuid.UUID   `arg:"" name:"index_id" help:"Index ID."`
	ColumnIDs []uuid.UUID `arg:"" name:"column_ids" help:"Column IDs."`
}

func (c *indexSetColumns) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.SetIndexColumnIDs(c.IndexID, c.ColumnIDs...))
}

type indexSetDescription struct {
	IndexID     uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
	Description string    `arg:"" name:"description" help:"Description."`
}

func (c *indexSetDescription) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.SetIndexDescription(c.IndexID, c.Description))
}

type indexSetDisabled struct {
	IndexID uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
}

func (c *indexSetDisabled) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.DisableIndex(c.IndexID))
}

type indexSetEnabled struct {
	IndexID uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
}

func (c *indexSetEnabled) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.EnableIndex(c.IndexID))
}

type indexSetName struct {
	IndexID uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
	Name    string    `arg:"" name:"name" help:"Name."`
}

func (c *indexSetName) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.SetIndexName(c.IndexID, c.Name))
}

type indexSetSearchable struct {
	IndexID uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
}

func (c *indexSetSearchable) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.MarkIndexSearchable(c.IndexID))
}

type indexSetType struct {
	IndexID       uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
	IndexType     string    `arg:"" name:"index_type" help:"Index type."`
	IndexSettings string    `arg:"" name:"index_settings" help:"Index settings."`
}

func (c *indexSetType) Run(helper *cmdHelper) error {
	indexType := userstoresearch.IndexType(c.IndexType)
	indexSettings, err := getIndexSettings(c.IndexSettings)
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(helper.searchMgr.SetIndexType(c.IndexID, indexType, indexSettings))
}

type indexSetUnsearchable struct {
	IndexID uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
}

func (c *indexSetUnsearchable) Run(helper *cmdHelper) error {
	return ucerr.Wrap(helper.searchMgr.MarkIndexUnsearchable(c.IndexID))
}

type indexStats struct {
	IndexID    uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
	OutputFile string    `help:"Output file."`
}

func (c *indexStats) Run(helper *cmdHelper) error {
	index, err := helper.searchMgr.GetIndex(c.IndexID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	client, err := helper.getSearchClient()
	if err != nil {
		return ucerr.Wrap(err)
	}
	stats, err := client.IndicesStatsRequest(helper.ctx, index.GetIndexName(helper.tenantID))
	if err != nil {
		return ucerr.Wrap(err)
	}
	return ucerr.Wrap(writeOrLogJSONResult(helper.ctx, "index stats", c.OutputFile, stats))
}

type indexCountDocuments struct {
	IndexID    uuid.UUID `arg:"" name:"index_id" help:"Index ID."`
	InputFile  string    `arg:"" name:"input_file" help:"Input file containing document IDs, one per line."`
	OutputFile string    `help:"Output file."`
}

func (c *indexCountDocuments) Run(helper *cmdHelper) error {
	client, err := helper.getSearchClient()
	if err != nil {
		return ucerr.Wrap(err)
	}

	index, err := helper.searchMgr.GetIndex(c.IndexID)
	if err != nil {
		return ucerr.Wrap(err)
	}
	docIDs, err := readDocumentIDsFromFile(c.InputFile)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(helper.ctx, "retrieving %d documents from index %s", len(docIDs), index.GetIndexName(helper.tenantID))
	count, err := client.CountDocumentsByIDs(helper.ctx, index.GetIndexName(helper.tenantID), docIDs)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(helper.ctx, "found %d/%d documents", count, len(docIDs))
	return nil
}

func readDocumentIDsFromFile(filename string) ([]string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	lines := strings.Split(string(content), "\n")
	docIDs := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			docIDs = append(docIDs, line)
		}
	}

	if len(docIDs) == 0 {
		return nil, ucerr.Errorf("no document IDs found in input file %s", filename)
	}
	return docIDs, nil
}

func writeOrLogJSONResult(ctx context.Context, resultName, outputFileName, result string) error {
	// Pretty print the JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(result), "", "  "); err != nil {
		return ucerr.Wrap(err)
	}
	prettyResult := prettyJSON.String()

	if outputFileName != "" {
		if err := os.WriteFile(outputFileName, []byte(prettyResult), 0644); err != nil {
			return ucerr.Wrap(err)
		}
		uclog.Infof(ctx, "%s written to '%s' (%d bytes)", resultName, outputFileName, len(prettyResult))
	} else {
		uclog.Infof(ctx, "%s:\n%v", resultName, prettyResult)
	}
	return nil
}
