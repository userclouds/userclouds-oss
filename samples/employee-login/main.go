package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/userclouds/userclouds/samples/employee-login/app"
	"github.com/userclouds/userclouds/samples/employee-login/routes/callback"
	"github.com/userclouds/userclouds/samples/employee-login/routes/home"
	"github.com/userclouds/userclouds/samples/employee-login/routes/login"
	"github.com/userclouds/userclouds/samples/employee-login/routes/logout"
	"github.com/userclouds/userclouds/samples/employee-login/routes/roles"
	"github.com/userclouds/userclouds/samples/employee-login/session"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/request"
)

func initRoutes() *mux.Router {
	baseMiddleware := middleware.Chain(request.Middleware(), middleware.ReadAll())
	// Redirect to home page for non-logged in users.
	authMiddleware := middleware.Chain(baseMiddleware, session.RedirectIfNotAuthenticated("/"))
	// Redirect to roles page if logged in.
	redirectIfAuthedMiddleware := middleware.Chain(baseMiddleware, session.RedirectIfAuthenticated("/roles"))

	// TODO: switch over to use uchttp to remove another dependency
	r := mux.NewRouter()
	r.Handle("/", redirectIfAuthedMiddleware.Apply(http.HandlerFunc(home.Handler)))
	r.Handle("/login", redirectIfAuthedMiddleware.Apply(http.HandlerFunc(login.Handler)))
	r.HandleFunc("/logout", logout.Handler)
	r.HandleFunc("/callback", callback.Handler)
	r.Handle("/roles", authMiddleware.Apply(http.HandlerFunc(roles.Handler)))

	r.PathPrefix("/public/").Handler(http.StripPrefix("/public/", http.FileServer(http.Dir("public/"))))

	return r
}

func main() {
	if err := app.Init(); err != nil {
		log.Fatal("Couldn't initialize the app")
	}

	r := initRoutes()
	http.Handle("/", r)
	log.Print("Server listening on http://localhost:3000/")
	log.Fatal(http.ListenAndServe("127.0.0.1:3000", nil))
}
