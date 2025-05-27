package userstore

import (
	"net/http"

	"github.com/go-http-utils/headers"

	"userclouds.com/idp/internal/storage"
	"userclouds.com/idp/internal/userstore/codegensdk"
	"userclouds.com/infra/uclog"
)

func (h *handler) getCodegenGolangSDK(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	sdk, err := codegensdk.CodegenGolangSDK(ctx, s)
	if err != nil {
		uclog.Errorf(ctx, "Error generating Golang SDK: %+v", err)
		return
	}

	w.Header().Set(headers.ContentType, "application/text")
	w.Write(sdk)
}

func (h *handler) getCodegenPythonSDK(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	s := storage.MustCreateStorage(ctx)

	includeExample := true
	if r.URL.Query().Get("include_example") == "false" {
		includeExample = false
	}

	sdk, err := codegensdk.CodegenPythonSDK(ctx, s, includeExample)
	if err != nil {
		uclog.Errorf(ctx, "Error generating Golang SDK: %+v", err)
		return
	}

	w.Header().Set(headers.ContentType, "application/text")
	w.Write(sdk)

}

func (h *handler) getCodegenTypescriptSDK(w http.ResponseWriter, r *http.Request) {
	w.Header().Set(headers.ContentType, "application/text")
	w.Write([]byte("Not yet implemented"))
}
