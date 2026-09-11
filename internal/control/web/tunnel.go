package web

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/config"
)

var tunnelURLRegex = regexp.MustCompile(`https://[a-zA-Z0-9-]+\.trycloudflare\.com`)

const (
	cloudflaredBinDir = "bin"
	cloudflaredBin    = "cloudflared"
	cloudflaredWinBin = "cloudflared.exe"
	cloudflaredBase   = "https://github.com/cloudflare/cloudflared/releases/latest/download"
)

// cloudflaredChecksums contains known-good SHA-256 hashes for cloudflared binaries.
// Update when cloudflared releases a new version.
// Source: https://github.com/cloudflare/cloudflared/releases
var cloudflaredChecksums = map[string]string{
	// Fetched from GitHub releases (run: fetch_cloudflared_checksums.sh)
	"cloudflared-linux-amd64":       "53b7a7a5420d188758d24341294acb0d1bca54296548ac05e38811a694ac6134",
	"cloudflared-linux-arm64":       "98aca3173f73248fad6180fc75dade2d186a6e54fa807e088108cb4345de8efe",
	"cloudflared-darwin-amd64":      "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5",
	"cloudflared-darwin-arm64":      "0019dfc4b32d63c1392aa264aed2253c1e0c2fb09216f8e2cc269bbfb8bb49b5",
	"cloudflared-windows-amd64.exe": "547057326266f0e1c7d50d102dbd22ff283d740c055bd61e94f10e2c606f89af",
}

// Tunnel represents a running Cloudflare Quick Tunnel.
type Tunnel struct {
	URL     string
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	mu      sync.Mutex
	stopped bool
}

// Stop terminates the cloudflared process and waits for it to exit.
func (t *Tunnel) Stop() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.stopped {
		return nil
	}
	t.stopped = true
	if t.cancel != nil {
		t.cancel()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		if err := t.cmd.Process.Kill(); err != nil {
			return err
		}
	}
	// Wait for process to be reaped (cmd.Wait called in background goroutine)
	return nil
}

// WaitForTunnel polls the tunnel URL until it's reachable or context expires.
// Call this AFTER the local web server is started.
func (t *Tunnel) WaitForTunnel(ctx context.Context) error {
	return waitForTunnelReady(ctx, t.URL)
}

func cloudflaredBinaryName() string {
	if runtime.GOOS == "windows" {
		return cloudflaredWinBin
	}
	return cloudflaredBin
}

func cloudflaredDir() (string, error) {
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(dataDir, cloudflaredBinDir)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("failed to create cloudflared directory: %w", err)
	}
	return dir, nil
}

func cloudflaredPath() (string, error) {
	dir, err := cloudflaredDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, cloudflaredBinaryName()), nil
}

func cloudflaredDownloadURL() string {
	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Normalize cloudflared naming conventions
	switch arch {
	case "amd64":
		arch = "amd64"
	case "arm64":
		arch = "arm64"
	}

	suffix := ""
	if runtime.GOOS == "windows" {
		suffix = ".exe"
	}

	return fmt.Sprintf("%s/cloudflared-%s-%s%s", cloudflaredBase, osName, arch, suffix)
}

// EnsureCloudflared checks if cloudflared is available and downloads it if not.
// Returns the path to the binary.
func EnsureCloudflared(ctx context.Context) (string, error) {
	// 1. Check PATH
	if path, err := exec.LookPath(cloudflaredBinaryName()); err == nil {
		return path, nil
	}

	// 2. Check local install
	binPath, err := cloudflaredPath()
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(binPath); err == nil {
		return binPath, nil
	}

	// 3. Download
	fmt.Fprintf(os.Stderr, "Baixando cloudflared (%s/%s)...\n", runtime.GOOS, runtime.GOARCH)
	if err := downloadCloudflared(ctx, binPath); err != nil {
		return "", fmt.Errorf("failed to download cloudflared: %w", err)
	}

	return binPath, nil
}

func downloadCloudflared(ctx context.Context, destPath string) error {
	url := cloudflaredDownloadURL()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("invalid download URL: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmpPath := destPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer func() {
		f.Close()
		os.Remove(tmpPath)
	}()

	h := sha256.New()
	writer := io.MultiWriter(f, h)

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("failed to write binary: %w", err)
	}
	if err := f.Close(); err != nil {
		return err
	}

	// Verify checksum - fail if no known hash for this binary (fail-closed).
	mapKey := fmt.Sprintf("cloudflared-%s-%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		mapKey += ".exe"
	}
	expected, ok := cloudflaredChecksums[mapKey]
	if !ok {
		return fmt.Errorf("no checksum pinned for %s; cannot verify integrity. Update cloudflaredChecksums map", mapKey)
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(actual, expected) {
		return fmt.Errorf("cloudflared checksum mismatch: expected %s, got %s", expected, actual)
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return os.Rename(tmpPath, destPath)
}

// waitForTunnelReady polls the tunnel URL until DNS propagates and it's reachable.
func waitForTunnelReady(ctx context.Context, tunnelURL string) error {
	// Extract hostname from URL
	hostname := strings.TrimPrefix(tunnelURL, "https://")
	hostname = strings.TrimSuffix(hostname, "/")
	if idx := strings.Index(hostname, "/"); idx >= 0 {
		hostname = hostname[:idx]
	}

	// Use external DNS resolvers for faster propagation check
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{}
			// Try Cloudflare DNS first, then Google
			for _, dns := range []string{"1.1.1.1:53", "8.8.8.8:53"} {
				conn, err := d.DialContext(ctx, "udp", dns)
				if err == nil {
					return conn, nil
				}
			}
			return nil, fmt.Errorf("all DNS resolvers failed")
		},
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	client := &http.Client{
		Timeout: 5 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Check DNS resolution via external resolver
			ips, err := resolver.LookupHost(ctx, hostname)
			if err != nil || len(ips) == 0 {
				continue
			}
			// DNS resolved, now try HTTP
			resp, err := client.Get(tunnelURL)
			if err == nil {
				resp.Body.Close()
				return nil
			}
		}
	}
}

// StartTunnel launches a Cloudflare Quick Tunnel pointing to the given local port.
// It blocks until the public URL is extracted or the context is canceled.
func StartTunnel(ctx context.Context, localPort int) (*Tunnel, error) {
	binPath, err := EnsureCloudflared(ctx)
	if err != nil {
		return nil, err
	}

	// The request context only governs startup. Once the tunnel URL has been
	// discovered, its lifetime belongs to Tunnel.Stop, not to the HTTP handler
	// that initiated it.
	tunnelCtx, cancel := context.WithCancel(context.Background())

	cmd := exec.CommandContext(tunnelCtx, binPath, "tunnel",
		"--url", fmt.Sprintf("http://localhost:%d", localPort),
		"--no-autoupdate",
	)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	tunnel := &Tunnel{
		cmd:    cmd,
		cancel: cancel,
	}

	// Parse stderr in background to find the public URL (one-shot)
	urlChan := make(chan string, 1)
	errChan := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if match := tunnelURLRegex.FindString(line); match != "" {
				select {
				case urlChan <- match:
				default:
				}
			}
		}
		if err := scanner.Err(); err != nil && err != io.EOF {
			select {
			case errChan <- fmt.Errorf("error reading cloudflared output: %w", err):
			default:
			}
		}
	}()

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start cloudflared: %w", err)
	}

	// Wait for URL with timeout
	select {
	case url := <-urlChan:
		tunnel.URL = url
	case err := <-errChan:
		cancel()
		_ = cmd.Process.Kill()
		return nil, err
	case <-time.After(30 * time.Second):
		cancel()
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("timeout waiting for tunnel URL")
	case <-ctx.Done():
		cancel()
		_ = cmd.Process.Kill()
		return nil, ctx.Err()
	}

	// Reap the child process in background to avoid zombies, including when
	// cloudflared exits before the caller explicitly stops the tunnel.
	go func() {
		_ = cmd.Wait()
		stderr.Close()
		tunnel.mu.Lock()
		tunnel.stopped = true
		tunnel.mu.Unlock()
	}()

	return tunnel, nil
}

// IsCloudflaredInstalled returns true if cloudflared is available.
func IsCloudflaredInstalled() bool {
	if _, err := exec.LookPath(cloudflaredBinaryName()); err == nil {
		return true
	}
	binPath, err := cloudflaredPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(binPath)
	return err == nil
}

// CloudflaredVersion returns the version string if available.
func CloudflaredVersion(path string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, path, "version").Output()
	if err != nil {
		return ""
	}
	firstLine := strings.SplitN(string(out), "\n", 2)
	if len(firstLine) > 0 {
		return strings.TrimSpace(firstLine[0])
	}
	return ""
}
