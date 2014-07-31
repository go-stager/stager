package stager

import (
	"time"
)

// State is the current running state of a backend
type State int

const (
	StateNew      State = iota // Newly created backend
	StateStarted               // Backend process is started
	StateRunning               // Backend is now running and accepting connections
	StateFinished              // Backend has finished running
	StateReaped                // Finished backend has been cleaned up
	StateErrored               // Backend process exited with error
)

const (
	BackendCheckDelay    = 200 * time.Millisecond
	BackendCheckAttempts = 1000
	BackendIdleCheck     = 10 * time.Second
)

const StaticDirName = "static"
const TemplatesDirName = "templates"
