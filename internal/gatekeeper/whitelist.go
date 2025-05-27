package gatekeeper

import "github.com/gofrs/uuid"

var allbirdsdevappID = uuid.Must(uuid.FromString("afd015cf-9d11-4235-b67d-4e8400c5785b"))
var palmettoucptestdefaultappID = uuid.Must(uuid.FromString("e964c305-5f6e-4dbb-aea8-cb0fdb6f782f"))
var palmettoucptestsampleapplicationID = uuid.Must(uuid.FromString("c9888b30-cc0f-4dbf-a3ff-d623593b237c"))
var palmettoucptestemployeeappID = uuid.Must(uuid.FromString("83a985bd-2465-4ddd-bd63-63fd1744e46e"))
var palmettodevdefaultappID = uuid.Must(uuid.FromString("e538c808-3f57-4415-b475-5b67d9396a6d"))
var palmettodevemployeeappID = uuid.Must(uuid.FromString("feba84e5-4e68-4abf-aebc-6af2a9157944"))

var featureWhitelist = map[string]map[uuid.UUID]bool{
	"plex_login_page": {
		allbirdsdevappID:                   true,
		palmettoucptestdefaultappID:        true,
		palmettoucptestsampleapplicationID: true,
		palmettoucptestemployeeappID:       true,
		palmettodevdefaultappID:            true,
		palmettodevemployeeappID:           true,
	},
	"plex_create_user_page": {
		allbirdsdevappID:                   true,
		palmettoucptestdefaultappID:        true,
		palmettoucptestsampleapplicationID: true,
		palmettoucptestemployeeappID:       true,
		palmettodevdefaultappID:            true,
		palmettodevemployeeappID:           true,
	},
}

// IsAppOnWhitelist returns true if the app is on the whitelist for the given feature
func IsAppOnWhitelist(appID uuid.UUID, feature string) bool {
	if w, found := featureWhitelist[feature]; found {
		if b, found := w[appID]; found {
			return b
		}
	}
	return false
}
