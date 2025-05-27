package resourcecheck

import (
	"net/http"
	"time"

	"userclouds.com/infra/cache"
	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/request"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uchttp/builder"
	"userclouds.com/infra/workerclient"
	"userclouds.com/internal/ucopensearch"
	"userclouds.com/worker"
)

// Response is the response to the /resourcecheck endpoint.
type Response struct {
	RequestID  string              `json:"request_id"`
	Redis      cache.RedisStatus   `json:"redis"`
	OpenSearch ucopensearch.Status `json:"opensearch"`
	Worker     int                 `json:"worker"`
}

func (rc Response) getHTTPCodeOption() jsonapi.Option {
	if !rc.Redis.Ok {
		return jsonapi.Code(http.StatusServiceUnavailable)
	}
	return jsonapi.Code(http.StatusOK)
}

func getNoOpMessage(durationSeconds string) (worker.Message, error) {
	var duration time.Duration
	var err error
	if durationSeconds != "" {
		duration, err = time.ParseDuration(durationSeconds)
		if err != nil {
			return worker.Message{}, ucerr.Wrap(err)
		}
		if duration <= 0 {
			return worker.Message{}, ucerr.New("duration must be positive")
		}
	} else {
		duration = time.Second * 2
	}
	return worker.CreateNoOpMessage(duration), nil
}

// AddResourceCheckEndpoint adds the /resourcecheck endpoint to the given handler builder.
func AddResourceCheckEndpoint(hb *builder.HandlerBuilder, cacheConfig *cache.Config, OpenSearchConfig *ucopensearch.Config, workerClient workerclient.Client) *builder.HandlerBuilder {
	resourceChecker := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		workerCount := 0
		if workerClient != nil {
			msg, err := getNoOpMessage(r.URL.Query().Get("duration"))
			if err != nil {
				jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusBadRequest))
				return
			}
			if err := workerClient.Send(ctx, msg); err != nil {
				jsonapi.MarshalError(ctx, w, err, jsonapi.Code(http.StatusInternalServerError))
				return
			}
			workerCount++
		}
		rc := Response{
			RequestID:  request.GetRequestID(ctx).String(),
			Redis:      cache.GetRedisStatus(ctx, cacheConfig),
			OpenSearch: ucopensearch.GetOpenSearchStatus(ctx, OpenSearchConfig),
			Worker:     workerCount,
		}
		jsonapi.Marshal(w, rc, rc.getHTTPCodeOption())
	}
	return hb.HandleFunc("/resourcecheck/", resourceChecker)

}
