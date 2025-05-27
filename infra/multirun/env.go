package multirun

import (
	"context"
	"sync"
)

// Env holds the environment for a multirun execution (the singleton io out, err, etc)
type Env struct {
	Output chan []byte
	Error  chan error

	MaxLen int

	Context context.Context
	Cancel  context.CancelFunc

	AllDone sync.WaitGroup
}

// NewEnv returns a new Env
func NewEnv(ctx context.Context, cmds []Command) *Env {
	var maxLen int
	for _, c := range cmds {
		maxLen = max(maxLen, len(c.GetName()))
	}
	ctx, cancel := context.WithCancel(ctx)
	return &Env{
		Output:  make(chan []byte),
		Error:   make(chan error),
		MaxLen:  maxLen,
		Context: ctx,
		Cancel:  cancel,
	}
}
