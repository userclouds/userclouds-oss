package authz_test

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"userclouds.com/authz"
	"userclouds.com/authz/ucauthz"
	"userclouds.com/idp/idptesthelpers"
	"userclouds.com/infra/assert"
)

func TestRBAC(t *testing.T) {
	ctx := context.Background()
	tf := newTestFixture(t)

	t.Run("test_create_user", func(t *testing.T) {
		t.Parallel()

		userID := uuid.Must(uuid.NewV4())
		_, err := tf.rbacClient.GetUser(ctx, userID)
		assert.NotNil(t, err)

		idptesthelpers.CreateUser(t, tf.tenantDB, userID, uuid.Nil, uuid.Nil, tf.tenant.TenantURL)
		user, err := tf.rbacClient.GetUser(ctx, userID)
		assert.NoErr(t, err)
		assert.NotNil(t, user, assert.Must())
		assert.Equal(t, user.ID, userID)
	})
	t.Run("test_user_roles", func(t *testing.T) {
		t.Parallel()

		user := tf.newTestUser()
		groupID := uuid.Must(uuid.NewV4())
		groupName := "testgroup_" + uuid.Must(uuid.NewV4()).String()
		group, err := tf.rbacClient.CreateGroup(ctx, groupID, groupName)
		assert.NoErr(t, err)
		assert.NotNil(t, group, assert.Must())
		assert.Equal(t, group.ID, groupID)
		assert.Equal(t, group.Name, groupName)

		_, err = group.AddUserRole(ctx, user, ucauthz.MemberRole)
		assert.NoErr(t, err)

		roles, err := user.GetMemberships(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(roles), 1, assert.Must())
		assert.Equal(t, roles[0].Role, ucauthz.MemberRole)
		assert.Equal(t, roles[0].Group.ID, group.ID)
		assert.Equal(t, roles[0].Group.Name, groupName)

		members, err := group.GetMemberships(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(members), 1, assert.Must())
		assert.Equal(t, members[0].Role, ucauthz.MemberRole)
		assert.Equal(t, members[0].User.ID, user.ID)
	})
	t.Run("test_remove_user", func(t *testing.T) {
		t.Parallel()

		user1 := tf.newTestUser()
		user2 := tf.newTestUser()

		groupName := "testgroup_" + uuid.Must(uuid.NewV4()).String()

		group, err := tf.rbacClient.CreateGroup(ctx, uuid.Must(uuid.NewV4()), groupName)
		assert.NoErr(t, err)
		_, err = group.AddUserRole(ctx, user1, ucauthz.MemberRole)
		assert.NoErr(t, err)
		_, err = group.AddUserRole(ctx, user1, ucauthz.AdminRole)
		assert.NoErr(t, err)
		_, err = group.AddUserRole(ctx, user2, ucauthz.MemberRole)
		assert.NoErr(t, err)

		members, err := group.GetMemberships(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(members), 3, assert.Must())
		ensureMembership(t, members, user1, ucauthz.MemberRole)
		ensureMembership(t, members, user1, ucauthz.AdminRole)
		ensureMembership(t, members, user2, ucauthz.MemberRole)

		assert.IsNil(t, group.RemoveUser(ctx, user1))
		members, err = group.GetMemberships(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(members), 1, assert.Must())
		ensureMembership(t, members, user2, ucauthz.MemberRole)

		assert.NoErr(t, group.RemoveUserRole(ctx, members[0].User, members[0].Role))
		members, err = group.GetMemberships(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(members), 0, assert.Must())

		// Add user2 back as member
		membership, err := group.AddUserRole(ctx, user2, ucauthz.MemberRole)
		assert.NoErr(t, err)
		members, err = group.GetMemberships(ctx)
		assert.NoErr(t, err)
		ensureMembership(t, members, user2, ucauthz.MemberRole)
		assert.IsNil(t, group.RemoveMembership(ctx, *membership))
		members, err = group.GetMemberships(ctx)
		assert.NoErr(t, err)
		assert.Equal(t, len(members), 0, assert.Must())
	})
}

func ensureMembership(t *testing.T, members []authz.Membership, user authz.User, roleName string) {
	t.Helper()
	for _, member := range members {
		if member.Role == roleName && member.User.ID == user.ID {
			return
		}
	}
	assert.True(t, false, assert.Errorf("did not find user '%s' with role '%s' in member list", user.ID, roleName))
}
