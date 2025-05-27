package idp_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"testing"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/idp"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/idp/policy"
	"userclouds.com/idp/userstore"
	"userclouds.com/infra/assert"
	"userclouds.com/infra/pagination"
)

type rateLimitStatus string

const (
	rlSuccess    rateLimitStatus = "success"
	rlFail429    rateLimitStatus = "fail_429"
	rlFail400    rateLimitStatus = "fail_400"
	rlFailSilent rateLimitStatus = "fail_silent"
)

type rateLimitingTestFixture struct {
	idptesthelpers.TestFixture
	a     *userstore.Accessor
	ap    *policy.AccessPolicy
	m     *userstore.Mutator
	users map[uuid.UUID]string
}

func newRateLimitingTestFixture(t *testing.T) *rateLimitingTestFixture {
	t.Helper()

	tf := idptesthelpers.NewTestFixture(t)

	// verify test idp storage has caching enabled
	s := idptesthelpers.NewStorage(tf.Ctx, t, tf.TenantDB, tf.TenantID)
	cm := s.CacheManager()
	assert.NotNil(t, cm)
	assert.True(t, cm.Provider.SupportsRateLimits(tf.Ctx))

	// create test access policy
	ap, err := tf.IDPClient.CreateAccessPolicy(
		tf.Ctx,
		policy.AccessPolicy{
			Name:        uniqueName("access_policy"),
			Description: "test access policy",
			Components: []policy.AccessPolicyComponent{
				{
					Template: &userstore.ResourceID{
						ID: policy.AccessPolicyTemplateAllowAll.ID,
					},
				},
			},
			PolicyType: policy.PolicyTypeCompositeAnd,
		},
	)
	assert.NoErr(tf.T, err)

	// create accessor that uses test access policy and tokenizes email
	a, err := tf.CreateAccessorWithWhereClause(
		uniqueName("accessor"),
		userstore.DataLifeCycleStateLive,
		[]string{"email"},
		[]uuid.UUID{uuid.Nil},
		[]uuid.UUID{policy.TransformerEmail.ID},
		[]string{"operational"},
		"{id} = ANY (?)",
		ap.ID,
		ap.ID,
	)
	assert.NoErr(tf.T, err)

	// create mutator that uses test access policy and modifies nickname column
	m, err := tf.CreateMutatorWithWhereClause(
		uniqueName("mutator"),
		ap.ID,
		[]string{"nickname"},
		[]uuid.UUID{policy.TransformerPassthrough.ID},
		"{id} = ANY (?)",
	)
	assert.NoErr(tf.T, err)

	return &rateLimitingTestFixture{
		TestFixture: *tf,
		a:           a,
		ap:          ap,
		m:           m,
		users:       map[uuid.UUID]string{},
	}
}

func (tf *rateLimitingTestFixture) createUser(email string) {
	tf.T.Helper()

	uid, err := tf.IDPClient.CreateUser(
		tf.Ctx,
		userstore.Record{"email": email},
		idp.OrganizationID(tf.Company.ID),
	)
	assert.NoErr(tf.T, err)
	tf.users[uid] = email
}

func (tf rateLimitingTestFixture) exerciseAccessor(rls rateLimitStatus) []string {
	tf.T.Helper()

	userIDs := tf.getUserIDs()
	resp, err := tf.IDPClient.ExecuteAccessor(
		tf.Ctx,
		tf.a.ID,
		nil,
		userstore.UserSelectorValues{userIDs},
	)

	switch rls {
	case rlSuccess:
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(resp.Data), len(userIDs))
		return resp.Data
	case rlFail429:
		assert.HTTPError(tf.T, err, http.StatusTooManyRequests)
		return nil
	case rlFail400:
		assert.HTTPError(tf.T, err, http.StatusBadRequest)
		return nil
	case rlFailSilent:
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(resp.Data), 0)
		return nil
	default:
		assert.Fail(tf.T, "unsupported rateLimitStatus: %v", rls)
		return nil
	}
}

func (tf rateLimitingTestFixture) exercisePaginatedAccessor(
	rls rateLimitStatus,
	forward bool,
	pageSize int,
) {
	assert.True(tf.T, pageSize > 0)

	userIDs := tf.getUserIDs()

	limit := idp.Pagination(pagination.Limit(pageSize))

	resp, err := tf.IDPClient.ExecuteAccessor(
		tf.Ctx,
		tf.a.ID,
		nil,
		userstore.UserSelectorValues{userIDs},
		tf.getCursor(forward),
		limit,
	)

	switch rls {
	case rlSuccess:
		assert.NoErr(tf.T, err)
	case rlFail429:
		assert.HTTPError(tf.T, err, http.StatusTooManyRequests)
		return
	case rlFail400:
		assert.HTTPError(tf.T, err, http.StatusBadRequest)
		return
	case rlFailSilent:
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(resp.Data), 0)
		assert.False(tf.T, resp.HasNext)
		assert.False(tf.T, resp.HasPrev)
		return
	default:
		assert.Fail(tf.T, "unsupported rateLimitStatus: %v", rls)
		return
	}

	numRemaining := len(userIDs)
	for numRemaining > 0 {
		numRetrieved := len(resp.Data)

		if numRemaining <= pageSize {
			assert.Equal(tf.T, numRetrieved, numRemaining)
		} else {
			assert.Equal(tf.T, numRetrieved, pageSize)
			resp, err = tf.IDPClient.ExecuteAccessor(
				tf.Ctx,
				tf.a.ID,
				nil,
				userstore.UserSelectorValues{userIDs},
				tf.getCursor(forward, resp.ResponseFields),
				limit,
			)
			assert.NoErr(tf.T, err)
		}

		numRemaining -= numRetrieved
	}
}

func (tf rateLimitingTestFixture) exerciseMutator(rls rateLimitStatus) {
	tf.T.Helper()

	userIDs := tf.getUserIDs()
	resp, err := tf.IDPClient.ExecuteMutator(
		tf.Ctx,
		tf.m.ID,
		nil,
		userstore.UserSelectorValues{userIDs},
		map[string]idp.ValueAndPurposes{
			"nickname": {
				Value: "bilbo",
				PurposeAdditions: []userstore.ResourceID{
					{Name: "operational"},
				},
			},
		},
	)

	switch rls {
	case rlSuccess:
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(resp.UserIDs), len(userIDs))
	case rlFail429:
		assert.HTTPError(tf.T, err, http.StatusTooManyRequests)
	case rlFail400:
		assert.HTTPError(tf.T, err, http.StatusBadRequest)
	case rlFailSilent:
		assert.NoErr(tf.T, err)
		assert.Equal(tf.T, len(resp.UserIDs), 0)
	default:
		assert.Fail(tf.T, "unsupported rateLimitStatus: %v", rls)
	}
}

func (tf rateLimitingTestFixture) exerciseTokenResolution(rls rateLimitStatus, results ...string) {
	tf.T.Helper()

	userIDs := tf.getUserIDs()
	assert.Equal(tf.T, len(results), len(userIDs))

	tokens := make([]string, 0, len(results))
	for _, result := range results {
		fields := map[string]string{}
		err := json.Unmarshal([]byte(result), &fields)
		assert.NoErr(tf.T, err)
		tokens = append(tokens, fields["email"])
	}

	resolvedTokens, err := tf.IDPClient.ResolveTokens(tf.Ctx, tokens, nil, nil)
	assert.NoErr(tf.T, err)
	assert.Equal(tf.T, len(resolvedTokens), len(tokens))
	switch rls {
	case rlSuccess:
		for i, resolvedToken := range resolvedTokens {
			assert.Equal(tf.T, resolvedToken, tf.users[userIDs[i]])
		}
	case rlFailSilent:
		for _, resolvedToken := range resolvedTokens {
			assert.Equal(tf.T, resolvedToken, "")
		}
	default:
		assert.Fail(tf.T, "unsupported rateLimitStatus: %v", rls)
	}
}

func (tf rateLimitingTestFixture) getCursor(forward bool, respFields ...pagination.ResponseFields) idp.Option {
	tf.T.Helper()

	assert.True(tf.T, len(respFields) < 2)
	if forward {
		if len(respFields) == 0 {
			return idp.Pagination(pagination.StartingAfter(pagination.CursorBegin))
		}
		assert.True(tf.T, respFields[0].HasNext)
		return idp.Pagination(pagination.StartingAfter(respFields[0].Next))
	}

	if len(respFields) == 0 {
		return idp.Pagination(pagination.EndingBefore(pagination.CursorEnd))
	}
	assert.True(tf.T, respFields[0].HasPrev)
	return idp.Pagination(pagination.EndingBefore(respFields[0].Prev))
}

func (tf rateLimitingTestFixture) getUserIDs() []uuid.UUID {
	tf.T.Helper()

	assert.True(tf.T, len(tf.users) > 0)

	var userIDs []uuid.UUID
	for userID := range tf.users {
		userIDs = append(userIDs, userID)
	}
	sort.Slice(userIDs, func(i int, j int) bool { return userIDs[i].String() < userIDs[j].String() })
	return userIDs
}

func (tf *rateLimitingTestFixture) resetUsers() {
	tf.users = map[uuid.UUID]string{}
}

func (tf *rateLimitingTestFixture) setThresholds(apt policy.AccessPolicyThresholds) {
	tf.T.Helper()

	currAP := *tf.ap
	currAP.Thresholds = apt
	updatedAP, err := tf.IDPClient.UpdateAccessPolicy(tf.Ctx, currAP)
	assert.NoErr(tf.T, err)
	tf.ap = updatedAP
}

func TestRateLimiting(t *testing.T) {
	t.Parallel()
	tf := newRateLimitingTestFixture(t)

	tf.createUser("foo@schmo.org")

	//// succeed with no thresholds

	tf.exerciseMutator(rlSuccess)
	results := tf.exerciseAccessor(rlSuccess)
	tf.exerciseTokenResolution(rlSuccess, results...)

	//// test result thresholds

	// succeed with result threshold greater than number of users

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			MaxResultsPerExecution: 2,
		},
	)

	tf.exerciseMutator(rlSuccess)
	results = tf.exerciseAccessor(rlSuccess)
	tf.exerciseTokenResolution(rlSuccess, results...)

	// succeed with number of users equal to result threshold

	tf.createUser("bar@schmo.org")

	tf.exerciseMutator(rlSuccess)
	results = tf.exerciseAccessor(rlSuccess)
	tf.exerciseTokenResolution(rlSuccess, results...)

	// fail silently with result threshold less than number of users

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			MaxResultsPerExecution: 1,
		},
	)

	tf.exerciseTokenResolution(rlFailSilent, results...)
	tf.exerciseMutator(rlFailSilent)
	tf.exerciseAccessor(rlFailSilent)

	// fail with appropriate error, except for token resolution, which
	// will still fail silently

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			AnnounceMaxResultFailure: true,
			MaxResultsPerExecution:   1,
		},
	)

	tf.exerciseTokenResolution(rlFailSilent, results...)
	tf.exerciseMutator(rlFail400)
	tf.exerciseAccessor(rlFail400)

	//// test rate thresholds

	// succeed until rate threshold is exceeded then fail silently

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			MaxExecutions:               2,
			MaxExecutionDurationSeconds: 5,
		},
	)

	tf.exerciseMutator(rlSuccess)
	tf.exerciseMutator(rlSuccess)
	tf.exerciseMutator(rlFailSilent)

	tf.exerciseAccessor(rlSuccess)
	results = tf.exerciseAccessor(rlSuccess)
	tf.exerciseAccessor(rlFailSilent)

	tf.exerciseTokenResolution(rlSuccess, results...)
	tf.exerciseTokenResolution(rlSuccess, results...)
	tf.exerciseTokenResolution(rlFailSilent, results...)

	// after window is passed should have same result

	time.Sleep(time.Second * 6)

	tf.exerciseMutator(rlSuccess)
	tf.exerciseMutator(rlSuccess)
	tf.exerciseMutator(rlFailSilent)

	tf.exerciseAccessor(rlSuccess)
	results = tf.exerciseAccessor(rlSuccess)
	tf.exerciseAccessor(rlFailSilent)

	tf.exerciseTokenResolution(rlSuccess, results...)
	tf.exerciseTokenResolution(rlSuccess, results...)
	tf.exerciseTokenResolution(rlFailSilent, results...)

	// after window has passed, succeed until rate threshold
	// is exceeded then fail with appropriate error

	time.Sleep(time.Second * 6)

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			AnnounceMaxExecutionFailure: true,
			MaxExecutions:               2,
			MaxExecutionDurationSeconds: 5,
		},
	)

	tf.exerciseMutator(rlSuccess)
	tf.exerciseMutator(rlSuccess)
	tf.exerciseMutator(rlFail429)

	tf.exerciseAccessor(rlSuccess)
	results = tf.exerciseAccessor(rlSuccess)
	tf.exerciseAccessor(rlFail429)

	tf.exerciseTokenResolution(rlSuccess, results...)
	tf.exerciseTokenResolution(rlSuccess, results...)
	tf.exerciseTokenResolution(rlFailSilent, results...)

	// after window is passed should have same result

	time.Sleep(time.Second * 6)

	tf.exerciseMutator(rlSuccess)
	tf.exerciseMutator(rlSuccess)
	tf.exerciseMutator(rlFail429)

	tf.exerciseAccessor(rlSuccess)
	results = tf.exerciseAccessor(rlSuccess)
	tf.exerciseAccessor(rlFail429)

	tf.exerciseTokenResolution(rlSuccess, results...)
	tf.exerciseTokenResolution(rlSuccess, results...)
	tf.exerciseTokenResolution(rlFailSilent, results...)

	//// test paginated accessor calls

	tf.resetUsers()
	for i := range 10 {
		tf.createUser(fmt.Sprintf("foo%d@schmo.org", i))
	}

	// should work fine with no thresholds

	tf.setThresholds(policy.AccessPolicyThresholds{})

	tf.exercisePaginatedAccessor(rlSuccess, true, 4)
	tf.exercisePaginatedAccessor(rlSuccess, false, 4)

	// succeed with full iteration until rate threshold
	// then fail silently

	time.Sleep(time.Second * 6)

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			MaxExecutions:               2,
			MaxExecutionDurationSeconds: 5,
		},
	)

	tf.exercisePaginatedAccessor(rlSuccess, true, 4)
	tf.exercisePaginatedAccessor(rlSuccess, false, 4)
	tf.exercisePaginatedAccessor(rlFailSilent, true, 4)
	tf.exercisePaginatedAccessor(rlFailSilent, false, 4)

	// succeed with full iteration until rate threshold
	// then fail with appropriate error

	time.Sleep(time.Second * 6)

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			AnnounceMaxExecutionFailure: true,
			MaxExecutions:               2,
			MaxExecutionDurationSeconds: 5,
		},
	)

	tf.exercisePaginatedAccessor(rlSuccess, true, 4)
	tf.exercisePaginatedAccessor(rlSuccess, false, 4)
	tf.exercisePaginatedAccessor(rlFail429, true, 4)
	tf.exercisePaginatedAccessor(rlFail429, false, 4)

	// test result threshold and page size permutations

	// succeed if result threshold is greater than or equal to num results

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			MaxResultsPerExecution: 10,
		},
	)

	tf.exercisePaginatedAccessor(rlSuccess, true, 4)
	tf.exercisePaginatedAccessor(rlSuccess, false, 4)
	tf.exercisePaginatedAccessor(rlSuccess, true, 12)
	tf.exercisePaginatedAccessor(rlSuccess, false, 12)

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			MaxResultsPerExecution: 12,
		},
	)

	tf.exercisePaginatedAccessor(rlSuccess, true, 4)
	tf.exercisePaginatedAccessor(rlSuccess, false, 4)
	tf.exercisePaginatedAccessor(rlSuccess, true, 12)
	tf.exercisePaginatedAccessor(rlSuccess, false, 12)

	// fail silently with result threshold less than num results

	// less than page size

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			MaxResultsPerExecution: 3,
		},
	)

	tf.exercisePaginatedAccessor(rlFailSilent, true, 4)
	tf.exercisePaginatedAccessor(rlFailSilent, false, 4)

	// greater than page size

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			MaxResultsPerExecution: 6,
		},
	)

	tf.exercisePaginatedAccessor(rlFailSilent, true, 4)
	tf.exercisePaginatedAccessor(rlFailSilent, false, 4)

	// fail with appropriate error with result threshold less than num results

	// less than page size

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			AnnounceMaxResultFailure: true,
			MaxResultsPerExecution:   3,
		},
	)

	tf.exercisePaginatedAccessor(rlFail400, true, 4)
	tf.exercisePaginatedAccessor(rlFail400, false, 4)

	// greater than page size

	tf.setThresholds(
		policy.AccessPolicyThresholds{
			AnnounceMaxResultFailure: true,
			MaxResultsPerExecution:   6,
		},
	)

	tf.exercisePaginatedAccessor(rlFail400, true, 4)
	tf.exercisePaginatedAccessor(rlFail400, false, 4)
}
