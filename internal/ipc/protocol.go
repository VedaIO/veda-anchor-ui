package ipc

import (
	"encoding/json"
)

// Request is a message sent to the server.
type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// Response is a message received from the server.
type Response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// GetIPCAddress returns the Windows Named Pipe address.
func GetIPCAddress() string {
	return `\\.\pipe\veda-anchor`
}
