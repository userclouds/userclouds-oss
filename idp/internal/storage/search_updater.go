package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp/config"
	"userclouds.com/infra/featureflags"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/uuidarray"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/ucopensearch"
	"userclouds.com/worker"
)

type searchUpdateBuffer struct {
	buf *bytes.Buffer
}

// Based on current customer usage in prod.
const defaultSearchBufferSize = 300

func newSearchUpdateBuffer() searchUpdateBuffer {
	return searchUpdateBuffer{buf: bytes.NewBuffer(make([]byte, 0, defaultSearchBufferSize))}
}

func (sub *searchUpdateBuffer) writeJSONLines(data ...any) error {
	// write both lines or nothing at all, so first marshal and then write
	lines := make([][]byte, 0, len(data))
	for _, d := range data {
		bs, err := json.Marshal(d)
		if err != nil {
			return ucerr.Wrap(err)
		}
		lines = append(lines, bs)
	}
	for _, line := range lines {
		sub.buf.Write(line)
		sub.buf.WriteString("\n")
	}
	return nil
}

type innerSearchIndex struct {
	IndexName string    `json:"_index"`
	ID        uuid.UUID `json:"_id"`
}

type searchIndex struct {
	Index innerSearchIndex `json:"index"`
}

type searchUpdateDataSource map[string]string

// SearchCandidate represents a potential value to include in the search index
type SearchCandidate struct {
	ValueID            uuid.UUID `db:"id"`
	UserID             uuid.UUID `db:"user_id"`
	ColumnID           uuid.UUID `db:"column_id"`
	VarcharValue       *string   `db:"varchar_value"`
	VarcharUniqueValue *string   `db:"varchar_unique_value"`
}

func newSearchCandidate(uclv UserColumnLiveValue) SearchCandidate {
	return SearchCandidate{
		ValueID:            uclv.ID,
		UserID:             uclv.UserID,
		ColumnID:           uclv.ColumnID,
		VarcharValue:       uclv.VarcharValue,
		VarcharUniqueValue: uclv.VarcharUniqueValue,
	}
}

func (sc SearchCandidate) getValue(c *Column) (string, error) {
	columnName, _, err := c.GetUserRowColumnNames()
	if err != nil {
		return "", ucerr.Wrap(err)
	}

	switch columnName {
	case "varchar_value":
		return *sc.VarcharValue, nil
	case "varchar_unique_value":
		return *sc.VarcharUniqueValue, nil
	default:
		return "", ucerr.Errorf("cannot currently index non-string terms: '%v'", c)
	}
}

// SearchUpdater collects updates that should be passed on to opensearch indices
type SearchUpdater struct {
	searchClient             *ucopensearch.Client
	workerClient             workerclient.Client
	s                        *UserStorage
	cm                       *ColumnManager
	sim                      *SearchIndexManager
	sub                      searchUpdateBuffer
	totalIndexUpdates        int
	totalProcessedCandidates int
}

// NewSearchUpdater returns a new search updater
func NewSearchUpdater(ctx context.Context, s *UserStorage, cm *ColumnManager, sim *SearchIndexManager, cfg *config.SearchUpdateConfig, retryOnTimeout bool) (su *SearchUpdater, err error) {
	var sc *ucopensearch.Client
	if cfg != nil && cfg.SearchCfg != nil {
		if retryOnTimeout {
			sc, err = ucopensearch.NewClientWithRetry(ctx, cfg.SearchCfg)
		} else {
			sc, err = ucopensearch.NewClient(ctx, cfg.SearchCfg)
		}
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	var workerClient workerclient.Client
	if cfg != nil {
		workerClient = cfg.WorkerClient
	}
	su = &SearchUpdater{
		searchClient: sc,
		workerClient: workerClient,
		s:            s,
		cm:           cm,
		sim:          sim,
	}
	su.reset()

	return su, nil
}

// hasUpdates returns true if there are any search updates to send
func (su SearchUpdater) hasUpdates() bool {
	return su.isSearchEnabled() && su.totalIndexUpdates > 0
}

func (su SearchUpdater) isSearchEnabled() bool {
	return su.searchClient != nil
}

// SendUpdatesIfNeeded sends any queued search updates (async)
func (su SearchUpdater) SendUpdatesIfNeeded(ctx context.Context) {
	if !su.hasUpdates() {
		return
	}
	go func(ctx context.Context) {
		if su.workerClient == nil {
			if err := su.sendUpdates(ctx); err != nil {
				uclog.Errorf(ctx, "InsertUserColumnLiveValues failed to update search: '%v'", err)
			}
		} else {
			data := su.sub.buf.Bytes()
			msg := worker.CreateUpdateTenantOpenSearchIndexMessage(su.s.tenantID, data, 0)
			uclog.Infof(ctx, "sending %d search candidates producing %d index updates to worker. %d bytes", su.totalProcessedCandidates, su.totalIndexUpdates, len(data))
			if err := su.workerClient.Send(ctx, msg); err != nil {
				uclog.Errorf(ctx, "Failed to send message to worker %v: '%v'", su.workerClient, err)
			}

		}
	}(context.WithoutCancel(ctx))
}

// ProcessColumnValue processes the user column value, queueing search updates to be sent as appropriate
func (su *SearchUpdater) ProcessColumnValue(
	ctx context.Context,
	v UserColumnLiveValue,
) error {
	if !su.isSearchEnabled() {
		return nil
	}

	su.totalProcessedCandidates++

	for _, usi := range su.sim.GetIndices() {
		if !usi.IsEnabled() {
			continue
		}

		if err := su.processSearchCandidate(usi, newSearchCandidate(v)); err != nil {
			return ucerr.Wrap(err)
		}
	}

	return nil
}

// ProcessIndex processes user column values for a specific index, processing
// the requested number of values for value ids that are greater than the last
// bootstrapped value ID. True is returned if the index has been fully bootstrapped.
func (su *SearchUpdater) ProcessIndex(ctx context.Context, indexID, lastRegionalBootstrappedValueID uuid.UUID, numValues int) (uuid.UUID, error) {
	if !su.isSearchEnabled() {
		return uuid.Nil, ucerr.Friendlyf(nil, "opensearch is not enabled")
	}

	usi := su.sim.GetIndexByID(indexID)
	if usi == nil {
		return uuid.Nil, ucerr.Friendlyf(nil, "index id '%v' is unrecognized", indexID)
	}
	if !usi.IsEnabled() {
		return uuid.Nil, ucerr.Friendlyf(nil, "index must be enabled: '%v'", usi)
	}
	if exists, err := su.searchClient.IndexExists(ctx, usi.GetIndexName(su.s.tenantID)); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	} else if !exists {
		return uuid.Nil, ucerr.Friendlyf(nil, "index '%v' does not exist in opensearch", usi.GetIndexName(su.s.tenantID))
	}
	alreadyBootstrapped := usi.IsBootstrapped()
	if lastRegionalBootstrappedValueID.IsNil() {
		val, ok := usi.LastRegionalBootstrappedValueIDs[su.s.GetRegion()]
		if ok {
			lastRegionalBootstrappedValueID = val
		}
	}
	if alreadyBootstrapped {
		return uuid.Nil, ucerr.Friendlyf(nil, "index must be unbootstrapped: '%v'", usi)
	}

	if numValues < 1 {
		return uuid.Nil, ucerr.Friendlyf(nil, "numValues must be larger than zero")
	}

	su.s.db.SetTimeout(2 * time.Minute) // longer timeout is fine since we are in the worker
	su.reset()
	// const countCandidatesQuery = `
	// /* lint-sql-unsafe-columns bypass-known-table-check */
	// SELECT COUNT(id) FROM user_column_pre_delete_values WHERE column_id = ANY($1) AND id > $2;`
	// var remainingCount int
	// if err := su.s.db.GetContext(ctx, "CountRemainingSearchCandidates", &remainingCount, countCandidatesQuery, uuidarray.UUIDArray(usi.ColumnIDs), lastBootstrappedValueID); err != nil {
	// 	return uuid.Nil, ucerr.Wrap(err)
	// }

	const getCandidatesQuery = `
/* lint-sql-unsafe-columns bypass-known-table-check */
SELECT
id,
user_id,
column_id,
varchar_value,
varchar_unique_value
FROM
user_column_pre_delete_values
WHERE
column_id = ANY($1)
AND id > $2
ORDER BY id
LIMIT $3;
`
	uclog.Infof(ctx, "retrieve up to (%d) search index bootstrap candidates for index '%v' starting at value id '%v'", numValues, usi.ID, lastRegionalBootstrappedValueID)
	useReplica := featureflags.IsEnabledGlobally(ctx, featureflags.ReadFromReadReplica)
	var scs []SearchCandidate
	if err := su.s.db.SelectContextWithDirty(
		ctx,
		"ProcessSearchCandidates",
		&scs,
		getCandidatesQuery,
		!useReplica,
		uuidarray.UUIDArray(usi.ColumnIDs),
		lastRegionalBootstrappedValueID,
		numValues+1,
	); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	uclog.Infof(ctx, "processing %d search index bootstrap candidates for index '%v' starting at value id '%v'", len(scs), usi.ID, lastRegionalBootstrappedValueID)
	totalValues := 0
	for _, sc := range scs {
		totalValues++
		if totalValues > numValues {
			break
		}

		lastRegionalBootstrappedValueID = sc.ValueID
		su.totalProcessedCandidates++

		if err := su.processSearchCandidate(*usi, sc); err != nil {
			return uuid.Nil, ucerr.Wrap(err)
		}
	}

	if err := su.sendUpdates(ctx); err != nil {
		return uuid.Nil, ucerr.Wrap(err)
	}

	if len(scs) <= numValues {
		uclog.Infof(ctx, "setting index '%v' bootstrapped (last batch: %d, max: %d)", usi.ID, len(scs), numValues)
		usi.LastRegionalBootstrappedValueIDs[su.s.GetRegion()] = uuid.Nil
		lastRegionalBootstrappedValueID = uuid.Nil
		usi.Bootstrapped = time.Now().UTC()
	} else {
		uclog.Infof(ctx, "setting last bootstrapped for index '%v' from: %v to %v", usi.ID, usi.LastRegionalBootstrappedValueIDs[su.s.GetRegion()], lastRegionalBootstrappedValueID)
		usi.LastRegionalBootstrappedValueIDs[su.s.GetRegion()] = lastRegionalBootstrappedValueID
	}
	if !alreadyBootstrapped {
		// If we already bootstrapped, then this was a forced bootstrap and we don't want to update the DB.
		if _, err := su.sim.UpdateIndex(ctx, usi); err != nil {
			return uuid.Nil, ucerr.Wrap(err)
		}
	}

	return lastRegionalBootstrappedValueID, nil
}

func (su *SearchUpdater) processSearchCandidate(usi UserSearchIndex, sc SearchCandidate) error {
	if !usi.supportsColumnID(sc.ColumnID) {
		return nil
	}

	c := su.cm.GetColumnByID(sc.ColumnID)
	if c == nil {
		return ucerr.Friendlyf(
			nil,
			"unrecognized search candidate column ID '%v'",
			sc.ColumnID,
		)
	}

	v, err := sc.getValue(c)
	if err != nil {
		return ucerr.Wrap(err)
	}

	suds := make(searchUpdateDataSource)
	suds[c.ID.String()] = v

	if err := su.sub.writeJSONLines(
		searchIndex{
			Index: innerSearchIndex{
				IndexName: usi.GetIndexName(su.s.tenantID),
				ID:        sc.UserID,
			},
		},
		suds,
	); err != nil {
		return ucerr.Wrap(err)
	}

	su.totalIndexUpdates++

	return nil
}

func (su *SearchUpdater) reset() {
	su.totalIndexUpdates = 0
	su.totalProcessedCandidates = 0
	su.sub = newSearchUpdateBuffer()
}

// sendUpdates sends any queued search updates
func (su *SearchUpdater) sendUpdates(ctx context.Context) error {
	if !su.isSearchEnabled() {
		return nil
	}
	if !su.hasUpdates() {
		uclog.Infof(ctx, "no search updates to send")
		return nil
	}
	data := su.sub.buf.Bytes()
	uclog.Infof(ctx, "processed %d search candidates producing %d index updates. %d bytes", su.totalProcessedCandidates, su.totalIndexUpdates, len(data))
	resp, err := su.searchClient.BulkRequest(ctx, data)
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "bulk search insert response: %s", string(resp)[:300])
	su.reset()
	return nil
}
