// Command loadtest exercises the local Web control plane at the release
// envelope of 20 projects and 10 terminal runtimes. It is deliberately
// self-contained and uses an isolated data directory, so it cannot modify a
// user's normal Nexus registry.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kivervinicius/ai-cli/internal/control/protocol"
	"github.com/kivervinicius/ai-cli/internal/control/web"
)

type loadClient struct {
	http  *http.Client
	base  string
	csrf  string
	write sync.Mutex
}

type projectResponse struct {
	ID string `json:"id"`
}

type shellResponse struct {
	Runtime struct {
		RuntimeID string `json:"runtime_id"`
	} `json:"runtime"`
}

type sessionResponse struct {
	Authenticated bool   `json:"authenticated"`
	CSRF          string `json:"csrf_token"`
}

type procMetrics struct {
	Goroutines int     `json:"goroutines"`
	RSSBytes   uint64  `json:"rss_bytes"`
	Sockets    int     `json:"sockets"`
	CPUSeconds float64 `json:"cpu_seconds"`
}

type report struct {
	Status              string      `json:"status"`
	Projects            int         `json:"projects"`
	Terminals           int         `json:"terminals"`
	Reconnects          int         `json:"reconnects"`
	OutputMarkers       int         `json:"output_markers"`
	FanoutDroppedChunks uint64      `json:"fanout_dropped_chunks"`
	Before              procMetrics `json:"before"`
	AtPeak              procMetrics `json:"at_peak"`
	After               procMetrics `json:"after"`
	Elapsed             string      `json:"elapsed"`
	Failure             string      `json:"failure,omitempty"`
}

func main() {
	projects := flag.Int("projects", 20, "number of isolated projects to create")
	terminals := flag.Int("terminals", 10, "number of shell terminals to exercise")
	jsonOutput := flag.Bool("json", false, "print the machine-readable report")
	flag.Parse()

	result := run(*projects, *terminals)
	if *jsonOutput {
		encoded, _ := json.MarshalIndent(result, "", "  ")
		fmt.Println(string(encoded))
	} else {
		fmt.Printf("Nexus load test: %s (%d projects, %d terminals, %d reconnects, %d markers)\n", result.Status, result.Projects, result.Terminals, result.Reconnects, result.OutputMarkers)
		fmt.Printf("goroutines %d→%d→%d, RSS %d→%d→%d bytes, sockets %d→%d→%d, CPU %.3fs, fanout drops %d, elapsed %s\n", result.Before.Goroutines, result.AtPeak.Goroutines, result.After.Goroutines, result.Before.RSSBytes, result.AtPeak.RSSBytes, result.After.RSSBytes, result.Before.Sockets, result.AtPeak.Sockets, result.After.Sockets, result.After.CPUSeconds-result.Before.CPUSeconds, result.FanoutDroppedChunks, result.Elapsed)
		if result.Failure != "" {
			fmt.Printf("failure: %s\n", result.Failure)
		}
	}
	if result.Status != "PASS" {
		os.Exit(1)
	}
}

func run(projectCount, terminalCount int) report {
	started := time.Now()
	result := report{Status: "FAIL", Projects: projectCount, Terminals: terminalCount}
	if projectCount <= 0 || terminalCount <= 0 || terminalCount > projectCount {
		result.Failure = "require 0 < terminals <= projects"
		result.Elapsed = time.Since(started).Round(time.Millisecond).String()
		return result
	}

	dataRoot, err := os.MkdirTemp("", "nexus-load-data-")
	if err != nil {
		result.Failure = err.Error()
		return result
	}
	defer os.RemoveAll(dataRoot)
	_ = os.Setenv("AI_MANAGER_DATA_DIR", filepath.Join(dataRoot, "data"))
	_ = os.Setenv("AI_CLI_DATA_DIR", filepath.Join(dataRoot, "data"))

	srv, err := web.NewServer(web.ServerOptions{Host: "127.0.0.1", Port: 0, NoOpen: true})
	if err != nil {
		result.Failure = err.Error()
		return result
	}
	serverDone := make(chan error, 1)
	go func() { serverDone <- srv.Start() }()
	defer func() {
		_ = srv.Shutdown(context.Background())
		select {
		case <-serverDone:
		case <-time.After(time.Second):
		}
	}()

	jar, _ := cookiejar.New(nil)
	client := &loadClient{http: &http.Client{Jar: jar, Timeout: 60 * time.Second}, base: srv.URL()}
	if err := client.bootstrap(srv.BootstrapURL()); err != nil {
		result.Failure = err.Error()
		return result
	}
	result.Before = readMetrics()

	projectIDs := make([]string, 0, projectCount)
	projectRoot, err := os.MkdirTemp(dataRoot, "projects-")
	if err != nil {
		result.Failure = err.Error()
		return result
	}
	for i := 0; i < projectCount; i++ {
		path := filepath.Join(projectRoot, fmt.Sprintf("project-%02d", i))
		if err := os.MkdirAll(path, 0700); err != nil {
			result.Failure = err.Error()
			return result
		}
		var project projectResponse
		if err := client.request(http.MethodPost, "/api/v1/projects", map[string]string{"name": fmt.Sprintf("Load Project %02d", i), "path": path}, &project); err != nil {
			result.Failure = fmt.Sprintf("create project %d: %v", i, err)
			return result
		}
		projectIDs = append(projectIDs, project.ID)
	}

	runtimeIDs := make([]string, terminalCount)
	var starts sync.WaitGroup
	var startErrMu sync.Mutex
	var startErr error
	for i := range runtimeIDs {
		starts.Add(1)
		go func(index int) {
			defer starts.Done()
			var shell shellResponse
			if err := client.request(http.MethodPost, "/api/v1/projects/"+url.PathEscape(projectIDs[index])+"/shell", map[string]string{}, &shell); err != nil {
				startErrMu.Lock()
				if startErr == nil {
					startErr = err
				}
				startErrMu.Unlock()
				return
			}
			runtimeIDs[index] = shell.Runtime.RuntimeID
		}(i)
	}
	starts.Wait()
	if startErr != nil {
		result.Failure = "start shells: " + startErr.Error()
		stopRuntimes(client, runtimeIDs)
		return result
	}

	result.AtPeak = readMetrics()
	var wsMu sync.Mutex
	var reconnects, markers int
	var wsErr error
	var wsWG sync.WaitGroup
	for index, runtimeID := range runtimeIDs {
		wsWG.Add(1)
		go func(index int, runtimeID string) {
			defer wsWG.Done()
			if err := exerciseTerminal(client, runtimeID, index); err != nil {
				wsMu.Lock()
				if wsErr == nil {
					wsErr = err
				}
				wsMu.Unlock()
				return
			}
			wsMu.Lock()
			markers++
			reconnects++
			wsMu.Unlock()
		}(index, runtimeID)
	}
	wsWG.Wait()
	if wsErr != nil {
		result.Failure = "terminal exercise: " + wsErr.Error()
	}
	result.Reconnects = reconnects
	result.OutputMarkers = markers
	for _, runtimeID := range runtimeIDs {
		if runtimeID == "" {
			continue
		}
		if runtimeClient, err := protocol.NewClient(runtimeID); err == nil {
			if status, statusErr := runtimeClient.Status(); statusErr == nil {
				result.FanoutDroppedChunks += status.DroppedOutputChunks
			}
			_ = runtimeClient.Close()
		}
	}
	stopRuntimes(client, runtimeIDs)
	result.After = readMetrics()
	result.Elapsed = time.Since(started).Round(time.Millisecond).String()
	if result.Failure == "" && result.Reconnects == terminalCount && result.OutputMarkers == terminalCount {
		result.Status = "PASS"
	}
	return result
}

func (c *loadClient) bootstrap(bootstrapURL string) error {
	resp, err := c.http.Get(bootstrapURL)
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("bootstrap returned HTTP %d", resp.StatusCode)
	}
	var session sessionResponse
	if err := c.request(http.MethodGet, "/api/v1/session", nil, &session); err != nil {
		return err
	}
	if !session.Authenticated || session.CSRF == "" {
		return fmt.Errorf("bootstrap did not authenticate")
	}
	c.csrf = session.CSRF
	return nil
}

func (c *loadClient) request(method, path string, body any, output any) error {
	var encoded []byte
	var err error
	if body != nil {
		encoded, err = json.Marshal(body)
		if err != nil {
			return err
		}
	}
	var reader *strings.Reader
	if encoded == nil {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(string(encoded))
	}
	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if method != http.MethodGet && method != http.MethodHead {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var payload map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return fmt.Errorf("%s %s returned HTTP %d: %v", method, path, resp.StatusCode, payload)
	}
	if output != nil {
		return json.NewDecoder(resp.Body).Decode(output)
	}
	return nil
}

func exerciseTerminal(client *loadClient, runtimeID string, index int) error {
	wsURL := strings.Replace(client.base, "http://", "ws://", 1) + "/api/v1/runtimes/" + url.PathEscape(runtimeID) + "/terminal"
	baseURL, _ := url.Parse(client.base)
	header := http.Header{"Origin": []string{client.base}}
	if cookies := client.http.Jar.Cookies(baseURL); len(cookies) > 0 {
		parts := make([]string, 0, len(cookies))
		for _, cookie := range cookies {
			parts = append(parts, cookie.Name+"="+cookie.Value)
		}
		header.Set("Cookie", strings.Join(parts, "; "))
	}
	dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
	conn, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return err
	}
	marker := fmt.Sprintf("NEXUS_LOAD_%02d", index)
	if err := readTerminalUntil(conn, "lease", "", 5*time.Second); err != nil {
		_ = conn.Close()
		return err
	}
	if err := conn.WriteJSON(map[string]any{"type": "input", "data": "printf '" + marker + "\\n'\n"}); err != nil {
		_ = conn.Close()
		return err
	}
	if err := readTerminalUntil(conn, "output", marker, 5*time.Second); err != nil {
		_ = conn.Close()
		return err
	}
	_ = conn.Close()

	reconnected, _, err := dialer.Dial(wsURL, header)
	if err != nil {
		return fmt.Errorf("reconnect: %w", err)
	}
	defer reconnected.Close()
	return readTerminalUntil(reconnected, "lease", "", 5*time.Second)
}

func readTerminalUntil(conn *websocket.Conn, wantType, contains string, timeout time.Duration) error {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	for {
		var message struct {
			Type string `json:"type"`
			Data string `json:"data"`
		}
		if err := conn.ReadJSON(&message); err != nil {
			return err
		}
		if message.Type == "error" {
			return fmt.Errorf("terminal error: %s", message.Data)
		}
		if message.Type == wantType && (contains == "" || strings.Contains(message.Data, contains)) {
			return nil
		}
	}
}

func stopRuntimes(client *loadClient, runtimeIDs []string) {
	for _, runtimeID := range runtimeIDs {
		if runtimeID != "" {
			_ = client.request(http.MethodPost, "/api/v1/runtimes/"+url.PathEscape(runtimeID)+"/stop", map[string]string{}, nil)
		}
	}
}

func readMetrics() procMetrics {
	metrics := procMetrics{Goroutines: runtime.NumGoroutine()}
	if data, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "VmRSS:") {
				var kilobytes uint64
				_, _ = fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "VmRSS:")), "%d kB", &kilobytes)
				metrics.RSSBytes = kilobytes * 1024
			}
		}
	}
	if entries, err := os.ReadDir("/proc/self/fd"); err == nil {
		for _, entry := range entries {
			target, targetErr := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
			if targetErr == nil && strings.HasPrefix(target, "socket:") {
				metrics.Sockets++
			}
		}
	}
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err == nil {
		metrics.CPUSeconds = float64(usage.Utime.Sec) + float64(usage.Utime.Usec)/1e6 + float64(usage.Stime.Sec) + float64(usage.Stime.Usec)/1e6
	}
	return metrics
}
