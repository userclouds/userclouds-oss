package idp

import "github.com/gofrs/uuid"

// PlaceholderPassword defines the password to use when syncing a user with an unknown password
// TODO: we should have a login-unavailable state for users rather than checking this constant :)
const PlaceholderPassword = "needs-to-be-synced"

// GetUserAccessorID is the ID of the accessor "get user"
var GetUserAccessorID = uuid.Must(uuid.FromString("28bf0486-9eea-4db5-ba40-5cef12dd48db"))

// UpdateUserMutatorID is the ID of the mutator "update user"
var UpdateUserMutatorID = uuid.Must(uuid.FromString("45c07eb8-f042-4cc8-b50e-3e6eb7b34bdc"))
