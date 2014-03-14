package proto

type DispatchRequest struct {
	Hosts   []string
	Command []string
	Timeout int
}

type DispatchResponse struct {
	SessionID string
}

type OutputRequest struct {
	SessionID string
	Host      string
	Output    []byte
}
