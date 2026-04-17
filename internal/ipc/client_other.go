//go:build !windows

package ipc

import (
	"encoding/json"
	"fmt"
)

type Client struct {
	address string
}

func NewClient() *Client {
	return &Client{
		address: GetIPCAddress(),
	}
}

func (c *Client) Request(method string, params any) (json.RawMessage, error) {
	return nil, fmt.Errorf("IPC client is only supported on Windows")
}
