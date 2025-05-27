// NOTE: automatically generated file -- DO NOT EDIT

package invite

import (
	"net/http"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/internal/auditlog"
	"userclouds.com/plex"
)

func handlerBuilder(builder *builder.HandlerBuilder, h *handler) {

	builder.MethodHandler("/send").Post(h.sendHandlerGenerated)

}

func (h *handler) sendHandlerGenerated(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req plex.SendInviteRequest
	if err := jsonapi.Unmarshal(r, &req); err != nil {
		jsonapi.MarshalError(ctx, w, err)
		return
	}

	var res any
	res, code, entries, err := h.sendHandler(ctx, req)
	auditlog.PostMultipleAsync(ctx, entries)

	if err != nil {
		jsonapi.MarshalError(ctx, w, err, jsonapi.Code(code))
		return
	}

	jsonapi.Marshal(w, res, jsonapi.Code(code))
}
