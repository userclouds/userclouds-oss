package uclog // TODO move this to better place once it is more filled out

import (
	"time"
)

// LocalStatus contains basic approximate statistics about the service
type LocalStatus struct {
	CallCount          int       `json:"callcount" yaml:"callcount"`                     // total calls received by the service
	InputErrorCount    int       `json:"input_errorcount" yaml:"input_errorcount"`       // number of input errors
	InternalErrorCount int       `json:"internal_errorcount" yaml:"internal_errorcount"` // number of internal errors
	LastCall           time.Time `json:"lastcall_time" yaml:"lastcall_time"`             // timestamp of last successful call
	LastErrorCall      time.Time `json:"lasterror_time" yaml:"lasterror_time"`           // timestamp of last error
	LastErrorCode      int       `json:"lasterror_code" yaml:"lasterror_code"`           // last error code
	ComputeTime        int       `json:"computetime" yaml:"computetime"`                 // amount of time spent in handlers

	LoggerStats []LogTransportStats `json:"loggerstats" yaml:"loggerstats"`
}

var status LocalStatus

// GetStatus return approximate statistics about the service
func GetStatus() LocalStatus {
	status.LoggerStats = GetStats()
	return status
}

// updateStatus updates stats, last writer wins some the results are approximate
func (s *LocalStatus) updateStatus(e LogEvent, t LogEventTypeInfo) {
	if e.Code == EventCodeNone {
		return
	}

	if t.Category == EventCategoryCall {
		s.CallCount++
		s.LastCall = time.Now().UTC()
		return
	}

	if t.Category == EventCategoryDuration {
		s.ComputeTime = s.ComputeTime + e.Count
		return
	}

	if t.Category == EventCategoryInternalError || t.Category == EventCategoryInputError {
		if t.Category == EventCategoryInternalError {
			s.InternalErrorCount++
		} else {
			s.InputErrorCount++
		}
		s.LastErrorCode = int(e.Code)
		s.LastErrorCall = time.Now().UTC()
	}
}
