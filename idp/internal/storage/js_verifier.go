package storage

import (
	"fmt"
	"strings"
	"sync"

	"github.com/gofrs/uuid"

	"userclouds.com/infra/ucerr"
	"userclouds.com/internal/auditlog"
	"userclouds.com/internal/ucv8go"
)

// cache of js script validations, so we only validate each unique script once
var jsScriptValidations = map[string]error{}
var jsScriptValidationsLock sync.RWMutex

func validateJSScript(scriptType string, jsScript string, auditLogEventType auditlog.EventType) error {
	jsScriptValidationsLock.Lock()
	defer jsScriptValidationsLock.Unlock()

	err, found := jsScriptValidations[jsScript]
	if !found {
		tmpFileName := fmt.Sprintf("%v.js", uuid.Must(uuid.NewV4()))
		_, err = ucv8go.RunScript(jsScript, tmpFileName, nil, auditLogEventType, nil)
		if err != nil {
			err = ucerr.Friendlyf(err, "%v", err) // lint-safe-wrap because this error comes from v8go
			// validation errors (JS syntax errors) contain multiple lines, the 2nd line is the Go stack trace, which we don't want here
			jsError := strings.Split(ucerr.UserFriendlyMessage(err), "\n")[0]
			err = ucerr.Friendlyf(err, "%s javascript validation error: %s", scriptType, jsError)
		}
		jsScriptValidations[jsScript] = err
	}
	return ucerr.Wrap(err)
}
