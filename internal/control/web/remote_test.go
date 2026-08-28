package web

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"strconv"
	"testing"
	"time"
)

// TestRemote_SSHTunnel simulates an SSH Port-Forwarding Tunnel (local forward):
// Remote Machine: runs AI Control Web on 127.0.0.1:<remotePort>
// SSH Tunnel: forwards localPort -> remotePort
// Local Client: connects to 127.0.0.1:<localPort> and verifies full control plane functionality.
func TestRemote_SSHTunnel(t *testing.T) {
	// 1. Start Web Server on loopback (simulating remote workstation)
	srv, err := NewServer(ServerOptions{
		Host:   "127.0.0.1",
		Port:   0,
		NoOpen: true,
	})
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}
	go func() {
		_ = srv.Start()
	}()
	defer srv.Shutdown(context.Background())

	time.Sleep(50 * time.Millisecond)

	remoteAddr := srv.listener.Addr().String()

	// 2. Create simulated SSH Port-Forwarding listener on local machine
	localListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create simulated tunnel listener: %v", err)
	}
	defer localListener.Close()

	tunnelStop := make(chan struct{})
	defer close(tunnelStop)

	go func() {
		for {
			localConn, err := localListener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				remoteConn, err := net.Dial("tcp", remoteAddr)
				if err != nil {
					return
				}
				defer remoteConn.Close()

				done := make(chan struct{}, 2)
				go func() {
					_, _ = io.Copy(remoteConn, c)
					done <- struct{}{}
				}()
				go func() {
					_, _ = io.Copy(c, remoteConn)
					done <- struct{}{}
				}()
				<-done
			}(localConn)
		}
	}()

	localPort := localListener.Addr().(*net.TCPAddr).Port

	// 3. Client on local machine visits tunnel URL: http://127.0.0.1:<localPort>/?token=...
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	tunnelBootstrapURL := "http://127.0.0.1:" + strconv.Itoa(localPort) + "/?token=" + srv.bootstrap
	resp, err := client.Get(tunnelBootstrapURL)
	if err != nil {
		t.Fatalf("failed to access via SSH tunnel: %v", err)
	}
	resp.Body.Close()

	// 4. Verify session cookie and CSRF token through the tunnel
	sessResp, err := client.Get("http://127.0.0.1:" + strconv.Itoa(localPort) + "/api/v1/session")
	if err != nil {
		t.Fatalf("failed to query session via tunnel: %v", err)
	}
	defer sessResp.Body.Close()

	var sessData struct {
		Authenticated bool   `json:"authenticated"`
		CSRFToken     string `json:"csrf_token"`
	}
	if err := json.NewDecoder(sessResp.Body).Decode(&sessData); err != nil {
		t.Fatalf("failed to decode session: %v", err)
	}

	if !sessData.Authenticated || sessData.CSRFToken == "" {
		t.Errorf("expected authenticated session over SSH tunnel, got %+v", sessData)
	}

	// 5. Verify API calls work over the tunnel
	healthResp, err := client.Get("http://127.0.0.1:" + strconv.Itoa(localPort) + "/api/v1/health")
	if err != nil || healthResp.StatusCode != http.StatusOK {
		t.Errorf("health check over SSH tunnel failed: %v", err)
	}
	healthResp.Body.Close()
}
