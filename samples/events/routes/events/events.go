package events

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/mux"
	"github.com/userclouds/userclouds/samples/events/app"
	"github.com/userclouds/userclouds/samples/events/routes/templates"
	"github.com/userclouds/userclouds/samples/events/session"

	"userclouds.com/authz"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp"
)

// eventWithInvitees represents an event with membership info
type eventWithInvitees struct {
	app.Event
	Creator  *authz.User   `json:"creator" yaml:"creator"`
	Invitees []*authz.User `json:"invitees" yaml:"invitees"`
}

// Handler renders UI for viewing/editing events.
func Handler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := session.GetProfile(r)
	user, err := app.GetAuthZClient().GetUser(ctx, session.GetUserID(r))
	if err != nil {
		uchttp.Error(ctx, w, err, http.StatusInternalServerError)
		return
	}
	data["userid"] = user.ID
	templates.RenderTemplate(ctx, w, "events", data)
}

func findCreatorAndInvitees(ctx context.Context, group *authz.Group) (*authz.User, []*authz.User, error) {
	members, err := group.GetMemberships(ctx)
	if err != nil {
		return nil, nil, ucerr.Wrap(err)
	}

	var creator *authz.User
	invitees := []*authz.User{}
	for _, member := range members {
		if member.Role == "creator" {
			if creator != nil {
				log.Printf("ignoring error: more than 1 creator for event '%s': '%v' and '%v'", group.Name, creator, member.User)
				continue
			}
			creator = &member.User
		} else if member.Role == "invitee" {
			invitees = append(invitees, &member.User)
		} else {
			log.Printf("ignoring error: unexpected role in event '%s': '%s'", group.Name, member.Role)
			continue
		}
	}
	return creator, invitees, nil
}

func isUserInList(user *authz.User, list []*authz.User) bool {
	for i := range list {
		if list[i].ID == user.ID {
			return true
		}
	}
	return false
}

func newEventWithInvitees(ctx context.Context, group *authz.Group, title string) (*eventWithInvitees, error) {
	creator, invitees, err := findCreatorAndInvitees(ctx, group)
	if err != nil {
		return nil, ucerr.Wrap(err)
	}
	return &eventWithInvitees{
		Event: app.Event{
			ID:    group.ID,
			Title: title,
		},
		Creator:  creator,
		Invitees: invitees,
	}, nil
}

// GetMyEvents returns a list of events in JSON for the logged in user.
func GetMyEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := session.GetUserID(r)
	user, err := app.GetAuthZClient().GetUser(ctx, userID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	groupRoles, err := user.GetMemberships(ctx)
	if err != nil {
		log.Printf("Error getting events for user '%s': %s", userID, err)
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	events := []eventWithInvitees{}
	for _, gr := range groupRoles {
		// Don't care about role here, just all groups we're related to.
		event, err := app.GetStorage().GetEvent(ctx, gr.Group.ID)
		if err != nil {
			log.Printf("ignoring error while loading event '%s': %s", gr.Group.Name, err)
			continue
		}

		eventWithInvitees, err := newEventWithInvitees(ctx, &gr.Group, event.Title)
		if err != nil {
			log.Printf("ignoring error while loading event members '%s': %s", gr.Group.Name, err)
			continue
		}
		events = append(events, *eventWithInvitees)
	}

	jsonapi.Marshal(w, events)
}

// UpdateEvent updates an event.
func UpdateEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	eventID, err := uuid.FromString(params["id"])
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
		return
	}

	var event eventWithInvitees
	if err := jsonapi.Unmarshal(r, &event); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	group, err := app.GetAuthZClient().GetGroup(ctx, eventID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
		return
	}

	creator, existingInvitees, err := findCreatorAndInvitees(ctx, group)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// Only a creator can edit the invitees
	if creator.ID != session.GetUserID(r) {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusForbidden))
		return
	}

	// Very naive double iteration to find delta
	for _, existingInvitee := range existingInvitees {
		if !isUserInList(existingInvitee, event.Invitees) {
			if err := group.RemoveUser(ctx, *existingInvitee); err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
		}
	}
	for _, newInvitee := range event.Invitees {
		if !isUserInList(newInvitee, existingInvitees) {
			if newInvitee.ID == creator.ID {
				continue
			}
			if _, err := group.AddUserRole(ctx, *newInvitee, "invitee"); err != nil {
				jsonapi.MarshalError(ctx, w, err)
				return
			}
		}
	}

	// Update events data, e.g. if title changed
	err = app.GetStorage().UpdateEvent(ctx, event.Event)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// DeleteEvent deletes an event.
func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params := mux.Vars(r)
	eventID, err := uuid.FromString(params["id"])
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
		return
	}

	group, err := app.GetAuthZClient().GetGroup(ctx, eventID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusNotFound))
		return
	}

	err = group.Delete(ctx)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	err = app.GetStorage().DeleteEvent(ctx, eventID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CreateEvent creates an event.
func CreateEvent(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := session.GetUserID(r)
	user, err := app.GetAuthZClient().GetUser(ctx, userID)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var event eventWithInvitees
	if err := jsonapi.Unmarshal(r, &event); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	// JS doesn't pass up ID / creator separately
	// TODO: clean up structs used for JS marshalling
	event.ID = uuid.Must(uuid.NewV4())
	objName := fmt.Sprintf("%s (%s)", event.ID, event.Title)
	group, err := app.GetAuthZClient().CreateGroup(ctx, event.ID, objName)
	if err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	if _, err := group.AddUserRole(ctx, *user, "creator"); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	for _, invitee := range event.Invitees {
		if invitee.ID == user.ID {
			continue
		}

		// Duplicate IDs and other issues will cause this error.
		if _, err := group.AddUserRole(ctx, *invitee, "invitee"); err != nil {
			log.Printf("ignoring error while inviting user '%s' to event '%s'", invitee.ID, group.Name)
		}
	}

	// Write event metadata into our own DB.
	// TODO: This may not be needed since the event title is part of the event object name?
	if err := app.GetStorage().CreateEvent(ctx, event.ID, event.Title); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
