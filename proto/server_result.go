package proto

type ResultRequest struct {
	SessionID string
}

type ResultResponse struct {
	SessionID string
	Output    map[string]OutputRequest
}
