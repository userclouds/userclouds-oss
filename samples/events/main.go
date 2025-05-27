package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/userclouds/userclouds/samples/events/app"
	"github.com/userclouds/userclouds/samples/events/routes/callback"
	"github.com/userclouds/userclouds/samples/events/routes/events"
	"github.com/userclouds/userclouds/samples/events/routes/home"
	"github.com/userclouds/userclouds/samples/events/routes/login"
	"github.com/userclouds/userclouds/samples/events/routes/logout"
	"github.com/userclouds/userclouds/samples/events/routes/user"
	"github.com/userclouds/userclouds/samples/events/session"

	"userclouds.com/infra/middleware"
	"userclouds.com/infra/request"
	loggingmiddlewre "userclouds.com/infra/uclog/middleware"
)

func initRoutes() *mux.Router {
	baseMiddleware := middleware.Chain(request.Middleware(), loggingmiddlewre.HTTPLoggerMiddleware(), middleware.ReadAll())
	// Redirect to home page for non-logged in users.
	authMiddleware := middleware.Chain(baseMiddleware, session.RedirectIfNotAuthenticated("/"))
	// Redirect to user home page for certain endpoints if logged in.
	redirectIfAuthedMiddleware := middleware.Chain(baseMiddleware, session.RedirectIfAuthenticated("/user"))

	// TODO: switch over to use uchttp to remove another dependency
	r := mux.NewRouter()
	r.Handle("/", redirectIfAuthedMiddleware.Apply(http.HandlerFunc(home.Handler)))
	r.Handle("/login", redirectIfAuthedMiddleware.Apply(http.HandlerFunc(login.Handler)))
	r.HandleFunc("/embeddedlogin", login.EmbeddedHandler)
	r.HandleFunc("/logout", logout.Handler)
	r.HandleFunc("/callback", callback.Handler)
	r.Handle("/user", authMiddleware.Apply(http.HandlerFunc(user.Handler)))
	r.Handle("/events", authMiddleware.Apply(http.HandlerFunc(events.Handler)))

	r.Handle("/api/events", authMiddleware.Apply(http.HandlerFunc(events.GetMyEvents))).Methods("GET")
	r.Handle("/api/event", authMiddleware.Apply(http.HandlerFunc(events.CreateEvent))).Methods("POST")
	r.Handle("/api/event/{id:[A-Za-z0-9-]+}", authMiddleware.Apply(http.HandlerFunc(events.UpdateEvent))).Methods("PUT")
	r.Handle("/api/event/{id:[A-Za-z0-9-]+}", authMiddleware.Apply(http.HandlerFunc(events.DeleteEvent))).Methods("DELETE")

	r.Handle("/api/users", authMiddleware.Apply(http.HandlerFunc(user.GetUsers))).Methods("GET")

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
