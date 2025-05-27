package invite

import (
	"context"
	"errors"

	"userclouds.com/infra/ucerr"
	"userclouds.com/plex/internal/otp"
	"userclouds.com/plex/internal/storage"
	"userclouds.com/plex/internal/tenantconfig"
)

// CheckForValidInvite unifies the code to check invites and parse error responses for system vs invalid failures.
// It returns a bool indicating if there is a valid invite, the email assocaited with the invite, and an error if one occurred.
func CheckForValidInvite(ctx context.Context, session *storage.OIDCLoginSession) (bool, string, error) {
	s := tenantconfig.MustGetStorage(ctx)

	// Check if there is an invite associated with this user creation event.
	if otpState, err := otp.HasUnusedInvite(ctx, s, session); err != nil {
		if errors.Is(err, otp.ErrNoInviteAssociatedWithSession) {
			// No invite, ignore
		} else if errors.Is(err, otp.ErrInviteBoundToAnotherUser) {
			// Already used and bound to a user.
			return false, "", ucerr.New("invite already used by another user")
		} else {
			return false, "", ucerr.Wrap(err)
		}
	} else {
		return true, otpState.Email, nil
	}
	return false, "", nil
}
