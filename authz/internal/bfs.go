package internal

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/ucmetrics"
	"userclouds.com/infra/uctrace"
)

var tracer = uctrace.NewTracer("bfs")

// A single edge attribute can have 1 of 3 flags set (direct, inherit, propagate), which affects how
// the graph is traversed. We track it here as a bitfield because we want to keep track both the set
// of visited objects in the graph as well as the way(s) in which they were visited to avoid cycles,
// but also handle the possibility there are multiple paths through an object.
type edgeAttrType int

const (
	edgeAttrTypeInherit edgeAttrType = 1 << iota
	edgeAttrTypeDirect
	edgeAttrTypePropagate
	authZSubsystem = ucmetrics.Subsystem("authz")
)

var (
	maxCandidatesMetric = ucmetrics.CreateGauge(authZSubsystem, "max_candidates", "Number of candidates in BFS", "tenant_id")
)

func loadEdgeMapFromDB(ctx context.Context, s *Storage) (map[uuid.UUID]map[uuid.UUID]*authz.Edge, error) {
	eM := make(map[uuid.UUID]map[uuid.UUID]*authz.Edge)
	lenEdges := 0
	updatedTime := time.Time{}

	pager, err := authz.NewEdgePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		edges, pr, err := s.ListEdgesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		for i, edge := range edges {
			if eM[edge.SourceObjectID] == nil {
				eM[edge.SourceObjectID] = make(map[uuid.UUID]*authz.Edge)
			}
			eM[edge.SourceObjectID][edges[i].ID] = &edges[i]

			if edge.Updated.After(updatedTime) {
				updatedTime = edge.Updated
			}

		}

		lenEdges += len(edges)

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	return eM, nil
}

func loadEdgeTypeMapFromDB(ctx context.Context, s *Storage) (map[uuid.UUID]*authz.EdgeType, error) {
	eTM := make(map[uuid.UUID]*authz.EdgeType)
	pager, err := authz.NewEdgeTypePaginatorFromOptions(pagination.Limit(pagination.MaxLimit))
	if err != nil {
		return nil, ucerr.Wrap(err)
	}

	for {
		edgeTypes, pr, err := s.ListEdgeTypesPaginated(ctx, *pager)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}

		for i, edgeType := range edgeTypes {
			eTM[edgeType.ID] = &(edgeTypes[i])
		}

		if !pager.AdvanceCursor(*pr) {
			break
		}
	}

	return eTM, nil
}

type bfsNode struct {
	authz.AttributePathNode

	// What kind of attribute flag was set on the edge that led us here.
	attrType edgeAttrType

	// The last node in the path
	prevNodeIdx int
}

type bfsSearcher struct {
	sourceObjectID     uuid.UUID
	targetObjectID     uuid.UUID // only one of targetObjectID or targetObjectTypeID should be set
	targetObjectTypeID uuid.UUID
	attributeName      string

	tenantID    uuid.UUID                               // for logging
	edgeMap     map[uuid.UUID]map[uuid.UUID]*authz.Edge // map from source object id to outbound edges
	edgeTypeMap map[uuid.UUID]*authz.EdgeType           // map from edge type id to edge type
	mu          *sync.RWMutex                           // if not nil, mu is needed to synchronize access to the edge map and edge type map

	// Map of nodes that have been visited with a bitfield to indicate how they've been visited
	// (i.e. what type of edge type attribute led to the node)
	visitedMap map[uuid.UUID]int
	candidates []bfsNode
	results    []uuid.UUID // will be populated if targetObjectTypeID is set
}

func checkVisited(visitedMap map[uuid.UUID]int, objectID uuid.UUID, attrType edgeAttrType) bool {
	bitfield, ok := visitedMap[objectID]
	if !ok {
		return false
	}
	return bitfield&int(attrType) != 0
}

// maxCandidatesAllowed limits the BFS in case of a bug or really large graph.
const maxCandidatesAllowed = 60000

func newBfsSearcher(tenantID, sourceObjectID, targetObjectID, targetObjectTypeID uuid.UUID, attributeName string) *bfsSearcher {
	// This is a stateful BFS from the source to the target. For any given candidate node in the search,
	// we also store the type of attribute (inherit, direct, propagate) that brought us to that point so we can
	// allow or disallow future edges accordingly.
	bfs := &bfsSearcher{
		sourceObjectID:     sourceObjectID,
		targetObjectID:     targetObjectID,
		targetObjectTypeID: targetObjectTypeID,
		attributeName:      attributeName,
		tenantID:           tenantID,
		visitedMap:         map[uuid.UUID]int{},
		candidates:         []bfsNode{},
		results:            []uuid.UUID{},
	}

	bfs.candidates = append(bfs.candidates, bfsNode{
		AttributePathNode: authz.AttributePathNode{
			ObjectID: sourceObjectID,
			EdgeID:   uuid.Nil},
		attrType:    edgeAttrTypeInherit,
		prevNodeIdx: -1,
	})

	return bfs
}

type edgeCacheSyncError struct {
	error
}

func (bfs *bfsSearcher) doBFS(ctx context.Context) (bool, error) {
	if bfs.edgeMap == nil || bfs.edgeTypeMap == nil {
		return false, ucerr.Errorf("edgeMap and/or edgeTypeMap is not set: %v %v", bfs.edgeMap, bfs.edgeTypeMap)
	}

	if bfs.mu != nil {
		bfs.mu.RLock()
		defer bfs.mu.RUnlock()
	}

	// Only one of targetObjectID or targetObjectTypeID should be set
	if bfs.targetObjectID != uuid.Nil && bfs.targetObjectTypeID != uuid.Nil {
		return false, ucerr.Errorf("targetObjectID and targetObjectTypeID are mutually exclusive")
	}

	maxCandidatesSeen := 0
	defer func() {
		// Using defer here to ensure that the metric is updated even if the function returns early
		maxCandidatesMetric.WithLabelValues(bfs.tenantID.String()).Set(float64(maxCandidatesSeen))
	}()
	// Iterate through all candidate nodes (NOTE: this array grows as a result of the iterations in the loop itself)
	i := 0
	for {
		maxCandidatesSeen = max(maxCandidatesSeen, i)
		if i > maxCandidatesAllowed {
			return false, ucerr.Errorf("exceeded max # of candidates (%d) in BFS without finding path: %d", maxCandidatesAllowed, i)
		}
		found, err := bfs.processEdges(i, &maxCandidatesSeen)
		if err != nil {
			return false, ucerr.Wrap(err)
		}
		if found {
			return true, nil
		}
		i++
		if i >= len(bfs.candidates) {
			break
		}
	}
	// No path possible
	return false, nil
}
func (bfs *bfsSearcher) processEdges(i int, maxCandidatesSeen *int) (bool, error) {
	// Iterate all edges from the candidate node.
	node := &bfs.candidates[i]
	for _, edge := range bfs.edgeMap[node.ObjectID] {
		if edge.SourceObjectID != node.ObjectID {
			// Only traverse outbound edges.
			continue
		}

		// Look up the edge type in cache. There is a small chance of inconsistency between when we cached the edges and the edgetypes
		edgeType, ok := bfs.edgeTypeMap[edge.EdgeTypeID]
		if !ok {
			return false, ucerr.Wrap(edgeCacheSyncError{ucerr.Errorf("Inconsistency detected for edgeType %v edge %v. Repeat the call", edge.EdgeTypeID, edge.BaseModel)})
		}

		for _, attr := range edgeType.Attributes {
			// Only operate on this edge if it has an attribute that matches the one we're searching for.
			if attr.Name == bfs.attributeName {
				newNode := bfsNode{
					AttributePathNode: authz.AttributePathNode{
						ObjectID: edge.TargetObjectID,
						EdgeID:   edge.ID,
					},
					prevNodeIdx: i,
				}

				// Based on the attribute flag that got us to this candidate node in the first place,
				// allow or disallow certain next steps in the path. A path can only go from inherit -> inherit | direct,
				// direct -> propagate, and propagate -> propagate.
				switch node.attrType {
				case edgeAttrTypeInherit:
					// If the last edge attribute type was inherit (which is also the default/starting type),
					// look for direct edges or inherit edges next.
					if attr.Direct {
						if edgeType.TargetObjectTypeID == bfs.targetObjectTypeID {
							bfs.results = append(bfs.results, edge.TargetObjectID)
						}

						newNode.attrType = edgeAttrTypeDirect
						if edge.TargetObjectID == bfs.targetObjectID {
							// Found target
							bfs.candidates = append(bfs.candidates, newNode)
							return true, nil
						}
						// Found intermediate node with right attribute, ensure it's not visited yet
						// (otherwise a shorter path exists) and then add to candidate list
						if !checkVisited(bfs.visitedMap, edge.TargetObjectID, edgeAttrTypeDirect) {
							bfs.visitedMap[edge.TargetObjectID] = bfs.visitedMap[edge.TargetObjectID] | int(edgeAttrTypeDirect)
							bfs.candidates = append(bfs.candidates, newNode)
						}
						// Don't break because there may be multiple attribute relationships that apply?
					}

					if attr.Inherit && edge.TargetObjectID != bfs.targetObjectID {
						// Found intermediate node we can inherit from.
						newNode.attrType = edgeAttrTypeInherit
						if !checkVisited(bfs.visitedMap, edge.TargetObjectID, edgeAttrTypeInherit) {
							bfs.visitedMap[edge.TargetObjectID] = bfs.visitedMap[edge.TargetObjectID] | int(edgeAttrTypeInherit)
							bfs.candidates = append(bfs.candidates, newNode)
						}
						// Don't break because there may be multiple attribute relationships that apply?
					}
				case edgeAttrTypeDirect:
					// If the last edge attribute type was direct, it must not have been the target
					// because we would have terminated the BFS, therefore it was an intermediate node
					// and the only path forward is to propagate attributes to the target.
					fallthrough
				case edgeAttrTypePropagate:
					// Look for propagation to target object or another intermediate object.
					if attr.Propagate {
						if edgeType.TargetObjectTypeID == bfs.targetObjectTypeID {
							bfs.results = append(bfs.results, edge.TargetObjectID)
						}

						newNode.attrType = edgeAttrTypePropagate
						if edge.TargetObjectID == bfs.targetObjectID {
							// Found target
							bfs.candidates = append(bfs.candidates, newNode)
							return true, nil
						}
						// Found intermediate node to propagate to
						if !checkVisited(bfs.visitedMap, edge.TargetObjectID, edgeAttrTypePropagate) {
							bfs.visitedMap[edge.TargetObjectID] = bfs.visitedMap[edge.TargetObjectID] | int(edgeAttrTypePropagate)
							bfs.candidates = append(bfs.candidates, newNode)
						}
					}
				}
			}
		}
	}
	return false, nil
}

func (bfs *bfsSearcher) doBFSFromStorage(ctx context.Context, s *Storage, skipCache bool) (bool, error) {
	if err := bfs.populateEdgeMapFromStorage(ctx, s, skipCache); err != nil {
		return false, ucerr.Wrap(err)
	}

	found, err := bfs.doBFS(ctx)
	if err != nil {
		var ecse edgeCacheSyncError
		if errors.As(err, &ecse) {
			resetGlobalCacheForTenant(bfs.tenantID, s.edgeCache, true)
		}
		return false, ucerr.Wrap(err)
	}

	return found, nil
}

func (bfs *bfsSearcher) populateEdgeMapFromStorage(ctx context.Context, s *Storage, skipCache bool) error {
	var edgeMap map[uuid.UUID]map[uuid.UUID]*authz.Edge
	var err error

	if !skipCache {
		if bfs.edgeMap != nil {
			return nil
		}
		if edgeMap, err = s.GetBFSEdgeGlobalCache(ctx); err != nil {
			return ucerr.Wrap(err)
		}
	}

	if edgeMap != nil {
		bfs.edgeMap = edgeMap
	} else {
		if bfs.edgeMap, err = uctrace.Wrap1(ctx, tracer, "loadEdgeMapFromDB", false, func(ctx context.Context) (map[uuid.UUID]map[uuid.UUID]*authz.Edge, error) {
			return loadEdgeMapFromDB(ctx, s)
		}); err != nil {
			return ucerr.Wrap(err)
		}
	}

	bfs.edgeTypeMap, err = loadEdgeTypeMapFromDB(ctx, s)
	if err != nil {
		return ucerr.Wrap(err)
	}

	return nil
}

// CheckAttributeBFS performs a Breadth-First Search for the shortest valid path from the source object to the target object.
// A valid path consists of a set of edges that connect the source to the target via 0 or more intermediate objects.
// Edges are valid candidates to form the path if their EdgeType has the desired attribute (exact string match), with some
// additional requirements:
//  1. Zero or more edges marked 'inherit' may connect a source object to an intermediate (non-target) object.
//  2. There must be exactly 1 'direct' edge in the path (in the trivial case, directly from source to target, but in more complex
//     cases the 'direct' edge may connect source to intermediate (if there are no 'inherit' edges on the path), intermediate to
//     intermediate (if there are both 'inherit' and 'propagate' edges on the path), or intermediate to target (if there are no
//     'propagate' edges on the path).
//  3. Zero or more edges marked 'propagate' may connect an intermediate (non-source) object to the target object.
//  4. Cycles are disallowed, though an object may be revisited with a different edge attribute type (hard to imagine why?).
func CheckAttributeBFS(ctx context.Context, s *Storage, tenantID, sourceObjectID, targetObjectID uuid.UUID, attributeName string, skipCache bool) (bool, []authz.AttributePathNode, error) {

	bfs := newBfsSearcher(tenantID, sourceObjectID, targetObjectID, uuid.Nil, attributeName)

	var found bool
	var err error

	found, err = bfs.doBFSFromStorage(ctx, s, skipCache)
	if err != nil {
		return false, nil, ucerr.Wrap(err)
	}

	var path []authz.AttributePathNode
	if found {
		path = make([]authz.AttributePathNode, 0)
		for i := len(bfs.candidates) - 1; i != -1; {
			node := bfs.candidates[i]
			path = append([]authz.AttributePathNode{{
				ObjectID: node.ObjectID,
				EdgeID:   node.EdgeID,
			}}, path...)
			i = node.prevNodeIdx
		}
	}
	return found, path, nil
}

// ListObjectsReachableWithAttributeBFS performs the same BFS as CheckAttributeBFS, but passes in a nil target object ID and returns the list of
// all objects that can be reached from the source object via a valid path.
func ListObjectsReachableWithAttributeBFS(ctx context.Context, s *Storage, tenantID, sourceObjectID uuid.UUID, targetObjectTypeID uuid.UUID, attributeName string) ([]uuid.UUID, error) {
	bfs := newBfsSearcher(tenantID, sourceObjectID, uuid.Nil, targetObjectTypeID, attributeName)
	if _, err := bfs.doBFSFromStorage(ctx, s, false); err != nil {
		return nil, ucerr.Wrap(err)
	}

	return bfs.results, nil
}
