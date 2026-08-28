package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	sessionCookieName = "ai_control_session"
	csrfHeaderName    = "X-CSRF-Token"
)

type Session struct {
	ID        string
	CSRFToken string
	CreatedAt time.Time
}

type AuthManager struct {
	mu             sync.RWMutex
	bootstrapToken string
	usedBootstrap  bool
	sessions       map[string]*Session
	listenHost     string
	listenPort     string
}

func NewAuthManager(listenHost, listenPort string) (*AuthManager, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, "", err
	}
	bootstrapToken := hex.EncodeToString(bytes)

	return &AuthManager{
		bootstrapToken: bootstrapToken,
		sessions:       make(map[string]*Session),
		listenHost:     listenHost,
		listenPort:     listenPort,
	}, bootstrapToken, nil
}

func (a *AuthManager) ValidateOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		// Non-browser or direct requests might not set Origin; check Referer
		referer := r.Header.Get("Referer")
		if referer == "" {
			return true // Allow CLI or local tools
		}
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		origin = u.Scheme + "://" + u.Host
	}

	// Strictly allow only loopback origin
	return strings.HasPrefix(origin, "http://127.0.0.1") ||
		strings.HasPrefix(origin, "http://localhost") ||
		strings.HasPrefix(origin, "http://[::1]")
}

func (a *AuthManager) CreateSession() (*Session, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	if _, err := rand.Read(csrfBytes); err != nil {
		return nil, err
	}
	csrfToken := hex.EncodeToString(csrfBytes)

	sess := &Session{
		ID:        sessID,
		CSRFToken: csrfToken,
		CreatedAt: time.Now(),
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions[sessID] = sess
	return sess, nil
}

func (a *AuthManager) AuthenticateRequest(r *http.Request) *Session {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil
	}

	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.sessions[cookie.Value]
}

func (a *AuthManager) ExchangeBootstrapToken(token string) (*Session, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.usedBootstrap || token == "" || token != a.bootstrapToken {
		return nil, false
	}

	// Mark bootstrap token as consumed (one-time use)
	a.usedBootstrap = true

	// Create new authenticated session
	bytes := make([]byte, 32)
	_, _ = rand.Read(bytes)
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	_, _ = rand.Read(csrfBytes)
	csrfToken := hex.EncodeToString(csrfBytes)

	sess := &Session{
		ID:        sessID,
		CSRFToken: csrfToken,
		CreatedAt: time.Now(),
	}
	a.sessions[sessID] = sess
	return sess, true
}
