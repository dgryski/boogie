package proto

type DispatchRequest struct {
	Hosts   []string
	Command string
	Timeout int
}

type DispatchResponse struct {
	SessionID string
}

type OutputRequest struct {
	SessionID string
	Host      string
	Stdout    []byte
	Stderr    []byte
	Err       string
	ExitCode  int
}
