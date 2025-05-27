package main

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/infra/pagination"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
)

func copyAuthz(ctx context.Context, src *tenantRecord, dst *tenantRecord) error {
	testStart := time.Now().UTC()
	uclog.Infof(ctx, "Authz: Starting copy loop")
	// Initalize client for both src and dst
	azcSrc, err := authz.NewClient(src.tenantURL, authz.JSONClient(src.tokenSource))
	if err != nil {
		uclog.Errorf(ctx, "Authz: AuthZ client creation failed: %v", err)
		return ucerr.Wrap(err)
	}
	azcDst, err := authz.NewClient(dst.tenantURL, authz.JSONClient(dst.tokenSource))
	if err != nil {
		uclog.Errorf(ctx, "Authz: AuthZ client creation failed: %v", err)
		return ucerr.Wrap(err)
	}
	// Initalize state for keeping track of data was read from src vs written to dest

	copiedObjectTypes := make(map[uuid.UUID]bool)
	copiedObjects := make(map[uuid.UUID]bool)
	copiedEdgeTypes := make(map[uuid.UUID]bool)
	copiedEdges := make(map[uuid.UUID]bool)

	for {
		copiedData := false
		// Read data from source
		objTypes, err := readAllObjectTypes(ctx, azcSrc)
		if err != nil {
			return ucerr.Wrap(err)
		}
		edgeTypes, err := readAllEdgeTypes(ctx, azcSrc)
		if err != nil {
			return ucerr.Wrap(err)
		}
		objs, err := readAllObjects(ctx, azcSrc)
		if err != nil {
			return ucerr.Wrap(err)
		}
		edges, err := readAllEdges(ctx, azcSrc, objs)
		if err != nil {
			return ucerr.Wrap(err)
		}

		// Copy object types
		writtenCount := 0
		for _, objType := range objTypes {
			if !copiedObjectTypes[objType.ID] {
				_, err = azcDst.CreateObjectType(ctx, objType.ID, objType.TypeName)
				if err != nil {
					uclog.Errorf(ctx, "Authz: Failed to copy over object type %v: %v", objType, err)
					return ucerr.Wrap(err)
				}
				copiedData = true
				copiedObjectTypes[objType.ID] = true
				writtenCount++
			}
		}
		uclog.Debugf(ctx, "Authz: Wrote %d object types to dest", writtenCount)
		// Copy edge types
		writtenCount = 0
		for _, edgeType := range edgeTypes {
			if !copiedEdgeTypes[edgeType.ID] {
				_, err = azcDst.CreateEdgeType(ctx, edgeType.ID, edgeType.SourceObjectTypeID, edgeType.TargetObjectTypeID, edgeType.TypeName, edgeType.Attributes)
				if err != nil {
					uclog.Errorf(ctx, "Authz: Failed to copy over edge type %v: %v", edgeType, err)
					return ucerr.Wrap(err)
				}
				copiedData = true
				copiedEdgeTypes[edgeType.ID] = true
				writtenCount++
			}
		}
		uclog.Debugf(ctx, "Authz: Wrote %d edge types to dest", writtenCount)
		// Copy objects
		writtenCount = 0
		for _, obj := range objs {
			if !copiedObjects[obj.ID] {
				var alias string
				if obj.Alias != nil {
					alias = *obj.Alias
				}
				_, err = azcDst.CreateObject(ctx, obj.ID, obj.TypeID, alias)
				if err != nil {
					uclog.Errorf(ctx, "Authz: Failed to copy over object %v: %v", obj, err)
					return ucerr.Wrap(err)
				}
				copiedData = true
				copiedObjects[obj.ID] = true
				writtenCount++
			}
		}
		uclog.Debugf(ctx, "Authz: Wrote %d objectss to dest", writtenCount)

		// Copy edges
		writtenCount = 0
		for _, edge := range edges {
			if !copiedEdges[edge.ID] {
				_, err = azcDst.CreateEdge(ctx, edge.ID, edge.SourceObjectID, edge.TargetObjectID, edge.EdgeTypeID)
				if err != nil {
					uclog.Errorf(ctx, "Authz: Failed to copy over edge %v: %v", edge, err)
					//return ucerr.Wrap(err)
				}
				copiedData = true
				copiedEdges[edge.ID] = true
				writtenCount++
			}
		}
		uclog.Debugf(ctx, "Authz: Wrote %d edges to dest", writtenCount)
		if !copiedData {
			uclog.Debugf(ctx, "Authz: Exiting copy loop no new data read")
			break
		}
	}

	wallTime := time.Now().UTC().Sub(testStart)
	uclog.Infof(ctx, "Authz: Completed copy loop in %v", wallTime)

	return nil
}

func readAllObjectTypes(ctx context.Context, azc *authz.Client) ([]authz.ObjectType, error) {
	objTypes, err := azc.ListObjectTypes(ctx)
	if err != nil {
		uclog.Errorf(ctx, "Authz: Failed to read object types: %v", err)
		return nil, ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "Authz: Read %d object types from source", len(objTypes))
	return objTypes, nil
}

func readAllEdgeTypes(ctx context.Context, azc *authz.Client) ([]authz.EdgeType, error) {
	edgeTypes, err := azc.ListEdgeTypes(ctx)
	if err != nil {
		uclog.Errorf(ctx, "Authz: Failed to read edge types: %v", err)
		return nil, ucerr.Wrap(err)
	}
	uclog.Debugf(ctx, "Authz: Read %d edge types from source", len(edgeTypes))
	return edgeTypes, nil
}

func readAllObjects(ctx context.Context, azc *authz.Client) ([]authz.Object, error) {
	objs := []authz.Object{}
	cursor := pagination.CursorBegin
	for {
		resp, err := azc.ListObjects(ctx, authz.Pagination(pagination.StartingAfter(cursor)))
		if err != nil {
			uclog.Errorf(ctx, "Authz: Authz: Failed to read objects: %v", err)
			return nil, ucerr.Wrap(err)
		}

		objs = append(objs, resp.Data...)

		if !resp.HasNext {
			break
		}
		cursor = resp.Next
	}
	uclog.Debugf(ctx, "Authz: Read %d objects from source", len(objs))
	return objs, nil
}

func readAllEdges(ctx context.Context, azc *authz.Client, objs []authz.Object) ([]authz.Edge, error) {
	edges := []authz.Edge{}
	for i, obj := range objs {
		cursor := pagination.CursorBegin
		for {
			resp, err := azc.ListEdgesOnObject(ctx, obj.ID, authz.Pagination(pagination.StartingAfter(cursor)))
			if err != nil {
				uclog.Errorf(ctx, "Authz: Failed to read edges: %v", err)
				return nil, ucerr.Wrap(err)
			}
			edges = append(edges, resp.Data...)
			if !resp.HasNext {
				break
			}
			cursor = resp.Next
		}
		uclog.Debugf(ctx, "Authz: Read edges for object %d", i)
	}
	uclog.Debugf(ctx, "Authz: Read %d edges from source", len(edges))
	return edges, nil
}
