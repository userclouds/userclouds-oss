package service

import (
	"fmt"
	"net/http"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/migrate"
	"userclouds.com/infra/ucdb"
	"userclouds.com/infra/uchttp/builder"
)

var buildHash = "missing"
var buildTime = "missing"

const (
	deployedPath = "/deployed"
)

// AddGetDeployedEndpoint adds the handler /deployed endpoint with the base service middleware
func AddGetDeployedEndpoint(hb *builder.HandlerBuilder) *builder.HandlerBuilder {
	hb.Handle(deployedPath, BaseMiddleware.Apply(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, buildHash)
			fmt.Fprintln(w, buildTime)
		}),
	))
	return hb
}

// CreateMigrationVersionHandler returns a simple http.Handler to return the max migration version
// used for deploy warnings
func CreateMigrationVersionHandler(db *ucdb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		v, err := migrate.GetMaxVersion(ctx, db)
		if err != nil {
			jsonapi.MarshalError(ctx, w, err)
			return
		}
		jsonapi.Marshal(w, []int{v})
	}
}

// GetBuildHash returns the linker-stamped git build hash
func GetBuildHash() string {
	return buildHash
}

// GetBuildTime returns the linker-stamped build time (based on the commit time, not the current time)
func GetBuildTime() string {
	return buildTime
}
