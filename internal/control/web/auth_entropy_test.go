package web

import (
	"crypto/rand"
	"errors"
	"io"
	"testing"
)

type failingEntropyReader struct{}

func (failingEntropyReader) Read([]byte) (int, error) { return 0, errors.New("entropy unavailable") }

func TestExchangeBootstrapTokenDoesNotConsumeTokenWhenEntropyFails(t *testing.T) {
	auth, token, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatal(err)
	}

	original := rand.Reader
	rand.Reader = failingEntropyReader{}
	if sess, ok := auth.ExchangeBootstrapToken(token); ok || sess != nil {
		t.Fatal("bootstrap exchange must fail closed when session entropy cannot be generated")
	}
	rand.Reader = original
	t.Cleanup(func() { rand.Reader = original })

	if sess, ok := auth.ExchangeBootstrapToken(token); !ok || sess == nil {
		t.Fatal("bootstrap token must remain usable after a transient entropy failure")
	}
}

var _ io.Reader = failingEntropyReader{}

func TestRotateSessionPreservesOldSessionWhenEntropyFails(t *testing.T) {
	auth, _, err := NewAuthManager("127.0.0.1", "0")
	if err != nil {
		t.Fatal(err)
	}
	old, err := auth.CreateSession()
	if err != nil {
		t.Fatal(err)
	}

	original := rand.Reader
	rand.Reader = failingEntropyReader{}
	if rotated, err := auth.RotateSession(old.ID); err == nil || rotated != nil {
		t.Fatal("rotation must fail when session entropy cannot be generated")
	}
	rand.Reader = original
	t.Cleanup(func() { rand.Reader = original })

	auth.mu.RLock()
	_, stillPresent := auth.sessions[old.ID]
	auth.mu.RUnlock()
	if !stillPresent {
		t.Fatal("old session must remain valid after a transient rotation entropy failure")
	}
}
