package cli

type ExitCodeError struct {
	Message  string
	ExitCode int
}

func (e ExitCodeError) Error() string {
	return e.Message
}
