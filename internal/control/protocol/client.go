package protocol

import (
	"bufio"
	"encoding/json"
	"errors"
	"net"
	"sync"
	"time"
)

// Client provides an RPC client for communicating with a SessionHost.
type Client struct {
	runtimeID string
	conn      net.Conn
	reader    *bufio.Reader
	mu        sync.Mutex
}

// NewClient connects to the specified runtime ID.
func NewClient(runtimeID string) (*Client, error) {
	conn, err := Dial(runtimeID)
	if err != nil {
		return nil, err
	}
	return &Client{
		runtimeID: runtimeID,
		conn:      conn,
		reader:    bufio.NewReader(conn),
	}, nil
}

// Close closes the connection.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Send sends a request and awaits a response.
func (c *Client) Send(cmd CommandType, payload any) (Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req, err := NewRequest(cmd, payload)
	if err != nil {
		return Response{}, err
	}

	data, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}

	_ = c.conn.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := c.conn.Write(append(data, '\n')); err != nil {
		return Response{}, err
	}

	line, err := c.reader.ReadBytes('\n')
	if err != nil {
		return Response{}, err
	}

	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, err
	}

	if !resp.OK {
		return resp, errors.New(resp.Error)
	}

	return resp, nil
}

// Ping checks if the SessionHost is alive.
func (c *Client) Ping() error {
	_, err := c.Send(CmdPing, nil)
	return err
}

// Status fetches the runtime status.
func (c *Client) Status() (StatusData, error) {
	resp, err := c.Send(CmdStatus, nil)
	if err != nil {
		return StatusData{}, err
	}
	var st StatusData
	if err := json.Unmarshal(resp.Data, &st); err != nil {
		return StatusData{}, err
	}
	return st, nil
}

// Stop requests a graceful stop.
func (c *Client) Stop() error {
	_, err := c.Send(CmdStop, nil)
	return err
}

// RawConn returns the underlying network connection for interactive streaming (attach).
func (c *Client) RawConn() net.Conn {
	return c.conn
}
