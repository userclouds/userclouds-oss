package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/uctypes/timer"
	"userclouds.com/internal/ucopensearch"
)

type searchResp struct {
	Hits struct {
		Hits []struct {
			ID     uuid.UUID      `json:"_id"`
			Source map[string]any `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

func executeQuery(ctx context.Context, client *ucopensearch.Client, index string, termKey string, q string, queryType string) error {
	var searchTerm string

	switch queryType {
	case "term":
		searchTerm = strings.ToLower(q)

	case "wildcard":
		searchTerm = fmt.Sprintf("*%s*", strings.ToLower(q))
	default:
		return ucerr.Errorf("unsupported query type: %s", queryType)
	}

	jsonSearchQueryBytes, err := json.Marshal(map[string]any{"size": 5, "query": map[string]any{queryType: map[string]any{termKey: searchTerm}}})
	if err != nil {
		return ucerr.Wrap(err)
	}
	jsonSearchQuery := string(jsonSearchQueryBytes)
	uclog.Infof(ctx, "search content: '%s'", jsonSearchQuery)
	t := timer.Start()
	respBody, err := client.SearchRequest(ctx, index, jsonSearchQuery)
	uclog.Infof(ctx, "took %v to search", t.Elapsed())
	if err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "response: %s", respBody)

	var res searchResp
	if err := json.Unmarshal(respBody, &res); err != nil {
		return ucerr.Wrap(err)
	}
	uclog.Infof(ctx, "%+v", res)
	return nil
}
