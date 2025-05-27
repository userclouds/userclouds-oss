package cmdline

import (
	"context"

	"userclouds.com/infra/namespace/universe"
	"userclouds.com/infra/ucerr"
	"userclouds.com/infra/uclog"
	"userclouds.com/infra/workerclient"
	"userclouds.com/worker/config"
)

// GetWorkerClientForTool returns a worker client for the tool. This is used to send messages/tasks to the worker from a command line tool.
func GetWorkerClientForTool(ctx context.Context) (workerclient.Client, error) {
	var wc workerclient.Client
	uv := universe.Current()
	if uv.IsCloud() {
		sqsURL, err := workerclient.GetSQSUrlForUniverse(ctx, uv)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		uclog.Infof(ctx, "Queue URL: %s", sqsURL)
		wc, err = workerclient.NewSQSWorkerClientForTool(ctx, sqsURL)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	} else {
		workerCfg, err := config.LoadConfig()
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
		uclog.Infof(ctx, "Queue config: %+v", workerCfg.WorkerClient)
		wc, err = workerclient.NewClientFromConfig(ctx, &workerCfg.WorkerClient)
		if err != nil {
			return nil, ucerr.Wrap(err)
		}
	}
	return wc, nil
}
