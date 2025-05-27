package usermanager

import (
	"context"

	"userclouds.com/idp"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/plex/internal/provider/iface"
)

// MarkEmailVerified updates the active & follower IDPs to confirm that an email is verified
// TODO: what is the best way to handle partial success?
// TODO: I think all of this code can simplify based on recent decision to limit sync features.
func MarkEmailVerified(ctx context.Context, activeClient iface.ManagementClient, followerClients []iface.ManagementClient, user *iface.UserProfile) error {
	// Nothing to do
	if user.EmailVerified {
		return nil
	}

	if err := activeClient.SetEmailVerified(ctx, user.ID, true); err != nil {
		uclog.Debugf(ctx, "set user '%s' email_verified to true failed on active provider (desc: '%s'): %v", user.Email, activeClient, err)
		return ucerr.Wrap(err)
	}

	// If the active succeeds, mark it in the user object.
	// NOTE: this doesn't save back to the DB, this just ensures that the caller has an up-to-date object)
	user.EmailVerified = true

	// NOTE: followers won't have their state updated if the active was already verified.
	var lastError error
	// TODO: if any of these fail, we should do something smarter than bail with an error
	for _, follower := range followerClients {
		// TODO: we only mark emails verified for username+password logins, should re-evaluate
		followerUsers, err := follower.ListUsersForEmail(ctx, user.Email, idp.AuthnTypePassword)
		if err == nil && len(followerUsers) != 1 {
			err = ucerr.Errorf("got %d users, expected 1", len(followerUsers))
		}
		if err != nil {
			lastError = err
			uclog.Debugf(ctx, "get user by email '%s' failed on follower provider (desc: '%s'): %v", user.Email, follower, err)
			continue
		}
		if err := follower.SetEmailVerified(ctx, followerUsers[0].ID, true); err != nil {
			lastError = err
			uclog.Debugf(ctx, "set user '%s' email_verified to true failed on follower provider (desc: '%s'): %v", user.Email, follower, err)
			continue
		}
	}
	return ucerr.Wrap(lastError)
}
