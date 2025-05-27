package api

import (
	"net/http"

	"github.com/go-http-utils/headers"
	"github.com/gofrs/uuid"

	"userclouds.com/infra/uclog"
)

func (h *handler) getCodegenGolangSDK(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		uclog.Errorf(ctx, "Error getting IDP Client: %+v", err)
		return
	}

	sdk, err := idpClient.DownloadGolangSDK(ctx)
	if err != nil {
		uclog.Errorf(ctx, "Error generating Golang SDK: %+v", err)
		return
	}

	w.Header().Set(headers.ContentType, "application/text")
	w.Write([]byte(sdk))
}

func (h *handler) getCodegenPythonSDK(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {
	ctx := r.Context()

	idpClient, err := h.newIDPClient(ctx, tenantID)
	if err != nil {
		uclog.Errorf(ctx, "Error getting IDP Client: %+v", err)
		return
	}

	sdk, err := idpClient.DownloadPythonSDK(ctx)
	if err != nil {
		uclog.Errorf(ctx, "Error generating Python SDK: %+v", err)
		return
	}

	w.Header().Set(headers.ContentType, "application/text")
	w.Write([]byte(sdk))
}

func (h *handler) getCodegenTypescriptSDK(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID) {

	w.Header().Set(headers.ContentType, "application/text")
	w.Write([]byte("Not yet implemented"))

}
