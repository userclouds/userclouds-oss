package internal

import (
	"context"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/jsonapi"
	"userclouds.com/infra/uclog"
	"userclouds.com/worker"
)

// RunningTasks is a struct that holds the running tasks
type RunningTasks struct {
	tasks map[uuid.UUID]taskInfo
	mu    sync.Mutex
}
type taskInfo struct {
	TaskName  string    `json:"TaskName" yaml:"TaskName"`
	TenantID  uuid.UUID `json:"TenantID" yaml:"TenantID"`
	RequestID uuid.UUID `json:"RequestID" yaml:"RequestID"`
	StartTime time.Time `json:"StartTime" yaml:"StartTime"`
}
type runningTaskOutput struct {
	taskInfo
	RunTime string `json:"RunTime" yaml:"RunTime"`
}

// NewRunningTasks creates a new RunningTasks instance
func NewRunningTasks() *RunningTasks {
	return &RunningTasks{
		tasks: make(map[uuid.UUID]taskInfo),
		mu:    sync.Mutex{},
	}
}

func (rt *RunningTasks) addTask(ctx context.Context, msg worker.Message, requestID uuid.UUID) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if rt, exists := rt.tasks[requestID]; exists {
		uclog.Errorf(ctx, "Request ID %s already exists in running tasks. %+v -- %+v", requestID, rt, msg)
	}
	rt.tasks[requestID] = taskInfo{
		TaskName:  string(msg.Task),
		TenantID:  msg.GetTenantID(),
		RequestID: requestID,
		StartTime: time.Now().UTC(),
	}

}
func (rt *RunningTasks) removeTask(requestID uuid.UUID) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.tasks, requestID)
}

func (rt *RunningTasks) getRunningTasksOutput() []runningTaskOutput {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	tasks := make([]taskInfo, 0, len(rt.tasks))
	for _, taskInfo := range rt.tasks {
		tasks = append(tasks, taskInfo)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].StartTime.Before(tasks[j].StartTime)
	})

	runningTasks := make([]runningTaskOutput, 0, len(tasks))
	for _, tInfo := range tasks {
		runningTasks = append(runningTasks, runningTaskOutput{
			taskInfo: tInfo,
			RunTime:  time.Since(tInfo.StartTime).String(),
		})
	}
	return runningTasks
}

// GetHTTPHandler returns an http.Handler that returns the running tasks
func (rt *RunningTasks) GetHTTPHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		jsonapi.Marshal(w, rt.getRunningTasksOutput(), jsonapi.Code(http.StatusOK))
	})
}
