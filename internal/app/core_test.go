package app

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCoreLifecycle(t *testing.T) {
	core, err := NewCore(CoreConfig{
		Host:   "127.0.0.1",
		Port:   0, // ephemeral port
		NoOpen: true,
	})
	if err != nil {
		t.Fatalf("failed to create Core: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- core.Start(ctx)
	}()

	select {
	case <-core.Ready():
		// Core is listening
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for Core to be ready")
	}

	url := core.URL()
	if url == "" {
		t.Fatal("expected non-empty URL")
	}

	resp, err := http.Get(url + "/api/v1/health")
	if err != nil {
		t.Fatalf("health check failed: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", resp.StatusCode)
	}

	// Test graceful shutdown
	stopCtx, stopCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer stopCancel()
	if err := core.Stop(stopCtx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}
