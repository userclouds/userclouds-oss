package user

import (
	"log"
	"net/http"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/userclouds/userclouds/samples/events/app"
	"github.com/userclouds/userclouds/samples/events/routes/templates"
	"github.com/userclouds/userclouds/samples/events/session"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp"
)

func filterUsersBySubstring(users []app.User, filter string) []app.User {
	result := []app.User{}
	filter = strings.ToLower(filter)
	for _, user := range users {
		if strings.Contains(strings.ToLower(user.Name), filter) {
			result = append(result, user)
		}
	}
	return result
}

func filterUsersByIDs(users []app.User, ids []uuid.UUID) []app.User {
	result := []app.User{}
	// O(N^2) because why not.
	for _, user := range users {
		for _, id := range ids {
			if user.ID == id {
				result = append(result, user)
				break
			}
		}
	}
	return result
}

// Handler displays the user's profile
func Handler(w http.ResponseWriter, r *http.Request) {
	templates.RenderTemplate(r.Context(), w, "user", session.GetProfile(r))
}

// GetUsers returns a list of app users in JSON
func GetUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := app.GetStorage().ListUsers(ctx)
	if err != nil {
		log.Printf("Error getting users: %s", err)
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}

	filter := r.URL.Query().Get("filter")
	if len(filter) > 0 {
		users = filterUsersBySubstring(users, filter)
	}

	idStrings, ok := r.URL.Query()["id"]
	if ok && len(idStrings[0]) > 0 {
		var ids []uuid.UUID
		for _, v := range idStrings {
			id, err := uuid.FromString(v)
			if err != nil {
				uchttp.Error(ctx, w, err, http.StatusBadRequest)
				return
			}
			ids = append(ids, id)
		}

		users = filterUsersByIDs(users, ids)
	}

	jsonapi.Marshal(w, users)
}
