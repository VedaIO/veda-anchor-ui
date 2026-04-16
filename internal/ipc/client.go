//go:build windows

package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Microsoft/go-winio"
	"github.com/google/uuid"
)

type Client struct {
	address string
	conn    net.Conn
	mu      sync.Mutex
}

func NewClient() *Client {
	return &Client{
		address: GetIPCAddress(),
	}
}

func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return nil
	}

	// Retry connection — the agent pipe may not be ready yet at UI startup
	var conn net.Conn
	var err error
	for i := 0; i < 15; i++ {
		timeout := 2 * time.Second
		conn, err = winio.DialPipe(c.address, &timeout)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("failed to connect to agent after retries: %w", err)
	}

	c.conn = conn
	return nil
}

func (c *Client) Request(method string, params interface{}) (json.RawMessage, error) {
	if err := c.connect(); err != nil {
		return nil, fmt.Errorf("failed to connect to engine: %w", err)
	}

	id := uuid.New().String()
	paramsJSON, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}

	req := Request{
		ID:     id,
		Method: method,
		Params: paramsJSON,
	}

	c.mu.Lock()
	encoder := json.NewEncoder(c.conn)
	if err := encoder.Encode(req); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}

	decoder := json.NewDecoder(c.conn)
	var resp Response
	if err := decoder.Decode(&resp); err != nil {
		c.conn.Close()
		c.conn = nil
		c.mu.Unlock()
		return nil, err
	}
	c.mu.Unlock()

	if resp.ID != id {
		return nil, fmt.Errorf("request ID mismatch: expected %s, got %s", id, resp.ID)
	}

	if resp.Error != "" {
		return nil, fmt.Errorf("engine error: %s", resp.Error)
	}

	return resp.Result, nil
}
