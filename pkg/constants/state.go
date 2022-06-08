package constants

const (
	// StateCreating indicates that the container is being created.
	StateCreating string = "creating"
	// StateCreated indicates that the runtime has finished the create operation.
	StateCreated string = "created"
	// StateRunning indicates that the container process has executed the
	// user-specified program but has not exited.
	StateRunning string = "running"
	// StateStopped indicates that the container process has exited.
	StateStopped string = "stopped"
)
