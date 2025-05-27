package app

import (
	"encoding/gob"
	"log"
	"net/http"

	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
)

var (
	store *sessions.FilesystemStore
)

// GetAuthSession returns a session object, parsed from the `auth-session` cookie
func GetAuthSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, "auth-session")
}

// Init sets up the employee login app
func Init() error {
	if err := godotenv.Load(); err != nil {
		log.Print(err.Error())
		return err
	}

	store = sessions.NewFilesystemStore("", []byte("something-very-secret"))
	gob.Register(map[string]any{})
	gob.Register(uuid.Nil)
	return nil
}
