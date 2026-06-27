package errs

import "errors"

// creating instace errs
var (
	ErrFailedToCreateClent         = errors.New("failed to create client for socket")
	ErrOverrideSocketNotAccessible = errors.New("override socket not accessible")
	ErrNoSockerFound               = errors.New("no accessible socket found")
	ErrNotASocket                  = errors.New("not a socket")
)

// intreacting with the socket for cotainer thing errs
var (
	ErrFailedToStartContainer   = errors.New("failed to start container")
	ErrFailedToDeleteContainer  = errors.New("failed to delete container")
	ErrFailedToInspectContainer = errors.New("failed to inspect container")
	ErrFailedToCreateContainer  = errors.New("failed to create container")
	ErrFailedToPullImage        = errors.New("failed to pull image")
	ErrFailedToCreateExec       = errors.New("failed to create exec")
	ErrFailedToAttachExec       = errors.New("failed to attach exec")
	ErrFailedToStdCopy          = errors.New("failed to copy")
	ErrFailedToLogs             = errors.New("failed to logs")
	ErrFailedToStopContainer    = errors.New("failed to stop container")
	ErrFailedToResumeContainer  = errors.New("failed to resume container")
	ErrFailedToSuspendContainer = errors.New("failed to suspend container")
)
