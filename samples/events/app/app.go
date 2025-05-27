package app

import (
	"context"
	"encoding/gob"
	"log"
	"net/http"
	"os"

	"github.com/gofrs/uuid"
	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"

	"userclouds.com/authz"
	"userclouds.com/infra/jsonclient"
)

var (
	store      *sessions.FilesystemStore
	storage    *Storage
	rbacClient *authz.RBACClient
)

func initAuthZClient() error {
	ctx := context.Background()
	tenantURL := os.Getenv("UC_TENANT_BASE_URL")
	clientID := os.Getenv("PLEX_CLIENT_ID")
	clientSecret := os.Getenv("PLEX_CLIENT_SECRET")
	tokenSource := jsonclient.ClientCredentialsTokenSource(tenantURL+"/oidc/token", clientID, clientSecret, nil)
	authZClient, err := authz.NewClient(tenantURL, authz.JSONClient(tokenSource))
	if err != nil {
		return err
	}
	rbacClient = authz.NewRBACClient(authZClient)
	_, err = rbacClient.GetRole(ctx, "creator")
	if err != nil {
		_, err = rbacClient.CreateRole(ctx, "creator")
		if err != nil {
			return err
		}
	}
	_, err = rbacClient.GetRole(ctx, "invitee")
	if err != nil {
		_, err = rbacClient.CreateRole(ctx, "invitee")
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAuthSession returns a session object, parsed from the `auth-session` cookie
func GetAuthSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, "auth-session")
}

// GetStorage returns an object to access Events storage
func GetStorage() *Storage {
	return storage
}

// GetAuthZClient returns an RBAC client for the authz system
func GetAuthZClient() *authz.RBACClient {
	return rbacClient
}

// Init sets up the events app
func Init() error {
	err := godotenv.Load()
	if err != nil {
		log.Print(err.Error())
		return err
	}

	storage, err = NewStorage()
	if err != nil {
		log.Print(err.Error())
		return err
	}

	err = initAuthZClient()
	if err != nil {
		log.Print(err.Error())
		return err
	}

	store = sessions.NewFilesystemStore("", []byte("something-very-secret"))
	gob.Register(map[string]any{})
	gob.Register(uuid.Nil)
	return nil
}
