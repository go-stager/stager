package stager

import (
	"time"
)

type State int

const (
	StateNew State = iota
	StateStarted
	StateRunning
	StateFinished
	StateReaped
	StateErrored
)

const (
	BackendCheckDelay    = 200 * time.Millisecond
	BackendCheckAttempts = 100
	BackendIdleCheck     = 10 * time.Second
)
