package internal_test

import (
	"context"
	"fmt"
	"math/rand"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/internal"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/pagination"
)

func makeCursor(objIDs []uuid.UUID, index int) pagination.Cursor {
	return pagination.Cursor(fmt.Sprintf("id:%v", objIDs[index]))
}

func verifyNoResults(t *testing.T, s *internal.Storage, options ...pagination.Option) {
	t.Helper()

	ctx := context.Background()

	p, err := authz.NewObjectPaginatorFromOptions(options...)
	assert.NoErr(t, err)
	results, respFields, err := s.ListObjectsPaginated(ctx, *p)
	assert.NoErr(t, err)
	assert.Equal(t, len(results), 0)
	assert.False(t, respFields.HasNext)
	assert.False(t, respFields.HasPrev)
}

func validateResult(t *testing.T, results []authz.Object, objIDs []uuid.UUID, ignoredObjIDs map[uuid.UUID]bool, baseIndexAdvance func(i int) int) int {
	t.Helper()

	totalResults := 0

	for i := range results {
		if !ignoredObjIDs[results[i].ID] {
			// Figure out the offset depending on if we are paginating forward or backwards
			bI := baseIndexAdvance(i)

			assert.Equal(t, results[i].ID, objIDs[bI], assert.Must())
			assert.Equal(t, *results[i].Alias, fmt.Sprintf("obj_%d", bI))

			totalResults++
		}
	}
	return totalResults
}

func TestPaginatedNoResults(t *testing.T) {
	t.Parallel()
	s := initStorage(context.Background(), t)

	t.Run("NoResultsAscendingForward", func(t *testing.T) {
		verifyNoResults(t, s, pagination.StartingAfter(pagination.CursorBegin), pagination.SortOrder(pagination.OrderAscending))
	})

	t.Run("NoResultsAscendingBackward", func(t *testing.T) {
		verifyNoResults(t, s, pagination.EndingBefore(pagination.CursorEnd), pagination.SortOrder(pagination.OrderAscending))
	})

	t.Run("NoResultsDescendingForward", func(t *testing.T) {
		verifyNoResults(t, s, pagination.StartingAfter(pagination.CursorBegin), pagination.SortOrder(pagination.OrderDescending))
	})

	t.Run("NoResultsDescendingBackward", func(t *testing.T) {
		verifyNoResults(t, s, pagination.EndingBefore(pagination.CursorEnd), pagination.SortOrder(pagination.OrderDescending))
	})
}

func getObjectsToIgnore(ctx context.Context, t *testing.T, s *internal.Storage) map[uuid.UUID]bool {
	p, err := authz.NewObjectPaginatorFromOptions(
		pagination.Limit(pagination.MaxLimit),
		pagination.StartingAfter(pagination.CursorBegin),
		pagination.SortOrder(pagination.OrderAscending))
	assert.NoErr(t, err)

	pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
	assert.NoErr(t, err)
	// We should have less than pagination.MaxLimit of provisioned objects
	assert.False(t, respFields.HasNext)
	assert.False(t, respFields.HasPrev)

	ignoredObjIDs := map[uuid.UUID]bool{}
	for i := range pagedObjs {
		ignoredObjIDs[pagedObjs[i].ID] = true
	}
	return ignoredObjIDs
}

func createObjects(t *testing.T, s *internal.Storage, numObjs int) []uuid.UUID {
	t.Helper()

	ctx := context.Background()

	createObjectType(t, ctx, s, "TestType")
	objIDs := make([]uuid.UUID, numObjs)
	objs := make([]authz.Object, numObjs)
	// Generate sequential UUIDs so we can validate the IDs we get back later.
	for i := range numObjs {
		objIDs[i] = uuid.Must(uuid.FromString(fmt.Sprintf("246be99a-98a5-4172-adb2-0ad93c2d%04d", i)))
		objs[i] = newObject(t, ctx, s, "TestType", fmt.Sprintf("obj_%d", i))
		objs[i].ID = objIDs[i]
	}

	// Randomize the object creation order so we test that pagination sorts properly
	rand.Shuffle(numObjs, func(i, j int) { objs[i], objs[j] = objs[j], objs[i] })

	for _, o := range objs {
		err := s.SaveObject(ctx, &o)
		assert.NoErr(t, err)
	}

	return objIDs
}

func TestPaginatedObjects(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := initStorage(ctx, t)

	ignoredObjIDs := getObjectsToIgnore(ctx, t, s)
	const numObjs = 12
	objIDs := createObjects(t, s, numObjs)

	t.Run("PageForwardAscendingLimit4Unrolled", func(t *testing.T) {
		pageSize := 4
		totalResults := 0
		baseIndex := 0
		p, err := authz.NewObjectPaginatorFromOptions(
			pagination.Limit(pageSize),
			pagination.StartingAfter(pagination.CursorBegin),
			pagination.SortOrder(pagination.OrderAscending))
		assert.NoErr(t, err)

		pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.False(t, respFields.HasPrev)
		assert.True(t, respFields.HasNext)
		assert.Equal(t, respFields.Next, makeCursor(objIDs, baseIndex+pageSize-1))
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex + i })
		baseIndex += pageSize
		assert.True(t, p.AdvanceCursor(*respFields))

		pagedObjs, respFields, err = s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.True(t, respFields.HasPrev)
		assert.Equal(t, respFields.Prev, makeCursor(objIDs, baseIndex))
		assert.True(t, respFields.HasNext)
		assert.Equal(t, respFields.Next, makeCursor(objIDs, baseIndex+pageSize-1))
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex + i })
		baseIndex += pageSize
		assert.True(t, p.AdvanceCursor(*respFields))

		pagedObjs, respFields, err = s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.True(t, respFields.HasPrev)
		assert.Equal(t, respFields.Prev, makeCursor(objIDs, baseIndex))
		assert.False(t, respFields.HasNext)
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex + i })
		assert.False(t, p.AdvanceCursor(*respFields))
		assert.Equal(t, totalResults, numObjs)
	})

	t.Run("PageBackwardAscendingLimit4Unrolled", func(t *testing.T) {
		pageSize := 4
		totalResults := 0
		baseIndex := numObjs
		p, err := authz.NewObjectPaginatorFromOptions(
			pagination.Limit(pageSize),
			pagination.EndingBefore(pagination.CursorEnd),
			pagination.SortOrder(pagination.OrderAscending))
		assert.NoErr(t, err)

		pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.True(t, respFields.HasPrev)
		assert.Equal(t, respFields.Prev, makeCursor(objIDs, baseIndex-pageSize))
		assert.False(t, respFields.HasNext)
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex - pageSize + i })
		baseIndex -= pageSize
		assert.True(t, p.AdvanceCursor(*respFields))

		pagedObjs, respFields, err = s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.True(t, respFields.HasPrev)
		assert.Equal(t, respFields.Prev, makeCursor(objIDs, baseIndex-pageSize))
		assert.True(t, respFields.HasNext)
		assert.Equal(t, respFields.Next, makeCursor(objIDs, baseIndex-1))
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex - pageSize + i })
		baseIndex -= pageSize
		assert.True(t, p.AdvanceCursor(*respFields))

		pagedObjs, respFields, err = s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.False(t, respFields.HasPrev)
		assert.True(t, respFields.HasNext)
		assert.Equal(t, respFields.Next, makeCursor(objIDs, baseIndex-1))
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex - pageSize + i })
		assert.False(t, p.AdvanceCursor(*respFields))
		assert.Equal(t, totalResults, numObjs)
	})

	t.Run("PageForwardDescendingLimit4Unrolled", func(t *testing.T) {
		pageSize := 4
		totalResults := 0
		baseIndex := numObjs - 1
		p, err := authz.NewObjectPaginatorFromOptions(
			pagination.Limit(pageSize),
			pagination.StartingAfter(pagination.CursorBegin),
			pagination.SortOrder(pagination.OrderDescending))
		assert.NoErr(t, err)

		pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.False(t, respFields.HasPrev)
		assert.True(t, respFields.HasNext)
		assert.Equal(t, respFields.Next, makeCursor(objIDs, baseIndex-pageSize+1))
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex - i })
		baseIndex -= pageSize
		assert.True(t, p.AdvanceCursor(*respFields))

		pagedObjs, respFields, err = s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.True(t, respFields.HasPrev)
		assert.Equal(t, respFields.Prev, makeCursor(objIDs, baseIndex))
		assert.True(t, respFields.HasNext)
		assert.Equal(t, respFields.Next, makeCursor(objIDs, baseIndex-pageSize+1))
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex - i })
		baseIndex -= pageSize
		assert.True(t, p.AdvanceCursor(*respFields))

		pagedObjs, respFields, err = s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.True(t, respFields.HasPrev)
		assert.Equal(t, respFields.Prev, makeCursor(objIDs, baseIndex))
		assert.False(t, respFields.HasNext)
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex - i })
		assert.False(t, p.AdvanceCursor(*respFields))
		assert.Equal(t, totalResults, numObjs)
	})

	t.Run("PageBackwardDescendingLimit4Unrolled", func(t *testing.T) {
		pageSize := 4
		totalResults := 0
		baseIndex := 0
		p, err := authz.NewObjectPaginatorFromOptions(
			pagination.Limit(pageSize),
			pagination.EndingBefore(pagination.CursorEnd),
			pagination.SortOrder(pagination.OrderDescending))
		assert.NoErr(t, err)

		pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.True(t, respFields.HasPrev)
		assert.Equal(t, respFields.Prev, makeCursor(objIDs, baseIndex+pageSize-1))
		assert.False(t, respFields.HasNext)
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex + pageSize - 1 - i })
		baseIndex += pageSize
		assert.True(t, p.AdvanceCursor(*respFields))

		pagedObjs, respFields, err = s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.True(t, respFields.HasPrev)
		assert.Equal(t, respFields.Prev, makeCursor(objIDs, baseIndex+pageSize-1))
		assert.True(t, respFields.HasNext)
		assert.Equal(t, respFields.Next, makeCursor(objIDs, baseIndex))
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex + pageSize - 1 - i })
		baseIndex += pageSize
		assert.True(t, p.AdvanceCursor(*respFields))

		pagedObjs, respFields, err = s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.False(t, respFields.HasPrev)
		assert.True(t, respFields.HasNext)
		assert.Equal(t, respFields.Next, makeCursor(objIDs, baseIndex))
		assert.Equal(t, len(pagedObjs), pageSize)
		totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex + pageSize - 1 - i })
		assert.False(t, p.AdvanceCursor(*respFields))
		assert.Equal(t, totalResults, numObjs)
	})

	// test a page size that divides evenly into numObjs, one that does not, and one that is larger than numObjs
	pageSizes := []int{6, 7, 15}
	for _, pageSize := range pageSizes {
		t.Run(fmt.Sprintf("PageForwardAscendingLimit%d", pageSize), func(t *testing.T) {
			totalResults := 0
			baseIndex := 0
			p, err := authz.NewObjectPaginatorFromOptions(
				pagination.Limit(pageSize),
				pagination.StartingAfter(pagination.CursorBegin),
				pagination.SortOrder(pagination.OrderAscending))
			assert.NoErr(t, err)

			for {
				pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
				assert.NoErr(t, err)

				totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex + i })

				if !p.AdvanceCursor(*respFields) {
					break
				}
				baseIndex += pageSize
			}
			assert.Equal(t, totalResults, numObjs)
		})

		t.Run(fmt.Sprintf("PageBackwardAscendingLimit%d", pageSize), func(t *testing.T) {
			totalResults := 0
			baseIndex := numObjs
			p, err := authz.NewObjectPaginatorFromOptions(
				pagination.Limit(pageSize),
				pagination.EndingBefore(pagination.CursorEnd),
				pagination.SortOrder(pagination.OrderAscending))
			assert.NoErr(t, err)

			for {
				pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
				assert.NoErr(t, err)

				totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex - len(pagedObjs) + i })
				baseIndex -= len(pagedObjs)

				if !p.AdvanceCursor(*respFields) {
					break
				}
			}
			assert.Equal(t, totalResults, numObjs)
		})

		t.Run(fmt.Sprintf("PageForwardDescendingLimit%d", pageSize), func(t *testing.T) {
			totalResults := 0
			baseIndex := numObjs - 1
			p, err := authz.NewObjectPaginatorFromOptions(
				pagination.Limit(pageSize),
				pagination.StartingAfter(pagination.CursorBegin),
				pagination.SortOrder(pagination.OrderDescending))
			assert.NoErr(t, err)

			for {
				pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
				assert.NoErr(t, err)

				totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex - i })
				baseIndex -= pageSize

				if !p.AdvanceCursor(*respFields) {
					break
				}
			}
			assert.Equal(t, totalResults, numObjs)
		})

		t.Run(fmt.Sprintf("PageBackwardDescendingLimit%d", pageSize), func(t *testing.T) {
			totalResults := 0
			baseIndex := 0
			p, err := authz.NewObjectPaginatorFromOptions(
				pagination.Limit(pageSize),
				pagination.EndingBefore(pagination.CursorEnd),
				pagination.SortOrder(pagination.OrderDescending))
			assert.NoErr(t, err)

			for {
				pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
				assert.NoErr(t, err)

				totalResults += validateResult(t, pagedObjs, objIDs, ignoredObjIDs, func(i int) int { return baseIndex + len(pagedObjs) - 1 - i })
				baseIndex += len(pagedObjs)

				if !p.AdvanceCursor(*respFields) {
					break
				}
			}
			assert.Equal(t, totalResults, numObjs)
		})
	}
}

func TestPaginatedFilterQuery(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := initStorage(ctx, t)

	const numObjs = 8
	objIDs := createObjects(t, s, numObjs)

	t.Run("PaginationFilterNoResults", func(t *testing.T) {
		p, err := authz.NewObjectPaginatorFromOptions(
			pagination.Filter(fmt.Sprintf("('id',EQ,'%v')", uuid.Nil)),
			pagination.SortOrder(pagination.OrderAscending),
			pagination.StartingAfter(pagination.CursorBegin))
		assert.NoErr(t, err)

		pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.False(t, respFields.HasNext)
		assert.False(t, respFields.HasPrev)
		assert.Equal(t, len(pagedObjs), 0)
	})

	t.Run("PaginationFilterFoundResult", func(t *testing.T) {
		testID := objIDs[3]
		p, err := authz.NewObjectPaginatorFromOptions(
			pagination.Filter(fmt.Sprintf("('id',EQ,'%v')", testID)),
			pagination.SortOrder(pagination.OrderAscending),
			pagination.StartingAfter(pagination.CursorBegin))
		assert.NoErr(t, err)

		pagedObjs, respFields, err := s.ListObjectsPaginated(ctx, *p)
		assert.NoErr(t, err)
		assert.False(t, respFields.HasNext)
		assert.False(t, respFields.HasPrev)
		assert.Equal(t, len(pagedObjs), 1)
		assert.Equal(t, pagedObjs[0].ID, testID)
	})
}
