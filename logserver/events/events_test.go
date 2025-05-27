package events

import (
	"testing"

	"userclouds.com/infra/namespace/service"
	"userclouds.com/infra/uclog"
)

func TestDuplicateCodes(t *testing.T) {
	m := GetLogEventTypes()
	rm := map[uclog.EventCode]string{}
	for n, v := range m {
		if dv, ok := rm[v.Code]; ok && !v.Ignore {
			t.Errorf("Found duplicate codes between events [%s] and [%s]. code: %v ", n, dv, v.Code)
		}
		if !v.Ignore {
			rm[v.Code] = n
		}
	}
}

func TestMissingName(t *testing.T) {
	m := GetLogEventTypes()

	for n, v := range m {
		if v.Name == "" || v.Name == "TBD" {
			t.Errorf("Missing event name [%s] code [%d]", n, v.Code)
		}
	}
}

func TestMissingService(t *testing.T) {
	m := GetLogEventTypes()

	for n, v := range m {
		if v.Service == "" {
			continue
		}
		if !service.IsValid(service.Service(v.Service)) {
			t.Errorf("Missing service name [%s] service [%s]", n, v.Service)
		}
	}
}
