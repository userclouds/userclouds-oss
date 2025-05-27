package home

import (
	"net/http"

	"github.com/userclouds/userclouds/samples/events/routes/templates"
)

// Handler renders home
func Handler(w http.ResponseWriter, r *http.Request) {
	templates.RenderTemplate(r.Context(), w, "home", nil)
}
