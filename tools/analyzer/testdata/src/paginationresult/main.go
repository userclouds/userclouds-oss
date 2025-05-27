package paginationresult

import (
	"context"
)

type Storage struct{}

func (s *Storage) GetLatestAccessors(ctx context.Context, p struct{}) ([]string, *struct{ HasNext bool }, error) {
	return nil, nil, nil
}

type pgr struct{}

func (p *pgr) AdvanceCursor(pr struct{ HasNext bool }) bool {
	return pr.HasNext
}

func badPagination(ctx context.Context) error {
	var storage Storage
	pager := struct{}{}

	// This should be flagged - ignoring pagination result
	items, _, err := storage.GetLatestAccessors(ctx, pager) // want "pagination result field from storage call is ignored"
	if err != nil {
		return err
	}
	for _, item := range items {
		_ = item
	}
	return nil
}

func goodPagination(ctx context.Context) error {
	var storage Storage
	pager := pgr{}

	// This should not be flagged - properly using pagination
	for {
		items, pr, err := storage.GetLatestAccessors(ctx, pager)
		if err != nil {
			return err
		}
		for _, item := range items {
			_ = item
		}
		if !pager.AdvanceCursor(*pr) {
			break
		}
	}
	return nil
}
