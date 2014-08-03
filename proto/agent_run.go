package proto

type RunRequest struct {
	SessionID    string
	ResponseHost string
	Command      string
	Timeout      int
}
