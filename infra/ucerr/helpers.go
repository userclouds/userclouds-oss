package ucerr

import (
	"context"
	"errors"
	"strings"
)

// IsContextCanceledError returns true if the error is a context.Canceled or if it is a DB execution canceled error
func IsContextCanceledError(err error) bool {
	return err != nil && (errors.Is(err, context.Canceled) ||
		(strings.Contains(err.Error(), "pq: query execution canceled")) || strings.Contains(err.Error(), "pq: canceling statement due to user request"))
}
