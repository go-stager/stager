package stager

type State int

const (
	StateNew State = iota
	StateStarted
	StateRunning
	StateFinished
	StateReaped
)
