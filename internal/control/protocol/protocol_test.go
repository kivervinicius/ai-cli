package protocol

import (
	"bufio"
	"encoding/json"
	"testing"
	"time"
)

func TestProtocolSerialization(t *testing.T) {
	req, err := NewRequest(CmdResize, ResizePayload{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if req.Command != CmdResize {
		t.Errorf("expected CmdResize, got %s", req.Command)
	}
	if req.ID == "" {
		t.Errorf("expected request ID to be populated")
	}

	resp, err := NewResponse(StatusData{
		RuntimeID: "rt-123",
		State:     "RUNNING",
		PID:       42,
	})
	if err != nil {
		t.Fatalf("failed to create response: %v", err)
	}
	if !resp.OK {
		t.Errorf("expected OK true")
	}

	errResp := NewErrorResponse("something went wrong")
	if errResp.OK || errResp.Error != "something went wrong" {
		t.Errorf("unexpected error response: %+v", errResp)
	}
}

func TestClientServerCommunication(t *testing.T) {
	runtimeID := "test-rt-001"
	l, err := Listen(runtimeID)
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer l.Close()

	// Mock Server
	go func() {
		conn, err := l.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				return
			}
			var req Request
			if err := json.Unmarshal(line, &req); err != nil {
				return
			}

			var resp Response
			switch req.Command {
			case CmdPing:
				resp, _ = NewResponse("pong")
			case CmdStatus:
				resp, _ = NewResponse(StatusData{
					RuntimeID: runtimeID,
					State:     "RUNNING",
					PID:       1001,
				})
			default:
				resp = NewErrorResponse("unknown command")
			}
			data, _ := json.Marshal(resp)
			_, _ = conn.Write(append(data, '\n'))
		}
	}()

	time.Sleep(50 * time.Millisecond)

	client, err := NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		t.Errorf("ping failed: %v", err)
	}

	st, err := client.Status()
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	if st.RuntimeID != runtimeID || st.State != "RUNNING" || st.PID != 1001 {
		t.Errorf("unexpected status data: %+v", st)
	}
}

func TestSubmitPromptRequestUsesDedicatedCommandAndPayload(t *testing.T) {
	req, err := NewRequest(CmdSubmitPrompt, SubmitPromptPayload{Prompt: "/ai literal prompt"})
	if err != nil {
		t.Fatal(err)
	}
	if req.Command != CmdSubmitPrompt {
		t.Fatalf("command=%s", req.Command)
	}
	var payload SubmitPromptPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Prompt != "/ai literal prompt" {
		t.Fatalf("prompt=%q", payload.Prompt)
	}
}
