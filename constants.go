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
)

const (
	BackendCheckDelay    = 200 * time.Millisecond
	BackendCheckAttempts = 100
)
