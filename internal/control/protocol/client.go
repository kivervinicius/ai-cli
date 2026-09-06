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

const MaxRPCResponseSize = 1024 * 1024

func readBounded(r *bufio.Reader, limit int) ([]byte, error) {
	var buf []byte
	for {
		chunk, isPrefix, err := r.ReadLine()
		buf = append(buf, chunk...)
		if len(buf) > limit {
			return nil, errors.New("response exceeded max size limit")
		}
		if err != nil {
			return nil, err
		}
		if !isPrefix {
			break
		}
	}
	return buf, nil
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

	line, err := readBounded(c.reader, MaxRPCResponseSize)
	_ = c.conn.SetDeadline(time.Time{}) // Disable deadline after RPC completes
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

// ClearDeadline removes any read/write timeouts from the connection.
func (c *Client) ClearDeadline() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		return c.conn.SetDeadline(time.Time{})
	}
	return nil
}

// Reader returns the client's buffered reader.
func (c *Client) Reader() *bufio.Reader {
	return c.reader
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

// Resize notifies the SessionHost of a window size change.
func (c *Client) Resize(rows, cols int) error {
	_, err := c.Send(CmdResize, ResizePayload{Rows: rows, Cols: cols})
	return err
}

// SubmitPrompt atomically sends a high-level prompt to an existing runtime.
// It intentionally does not acquire or steal an interactive terminal writer lease.
func (c *Client) SubmitPrompt(prompt string) error {
	_, err := c.Send(CmdSubmitPrompt, SubmitPromptPayload{Prompt: prompt})
	return err
}

// RawConn returns the underlying network connection for interactive streaming (attach).
func (c *Client) RawConn() net.Conn {
	return c.conn
}
