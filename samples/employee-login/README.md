# UserClouds Employee Login Sample App

This app demonstrates how to use the employee login flow in a customer application.

## Running Employee Login

Execute `go run main.go` to run the Employee Login app.

By default this will target the locally-running dev instances of Plex and AuthZ. Edit .env to change the PLEX_CLIENT_ID, PLEX_CLIENT_ECRET, UC_TENANT_BASE_URL, or OIDC_CALLBACK_URL. You'll want to run `make dev or make-ui-dev` in a separate prompt to ensure that the UserClouds services are running.
