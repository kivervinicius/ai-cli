package web

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/kivervinicius/ai-cli/internal/control/originpolicy"
)

const (
	sessionCookieName = "ai_control_session"
	csrfHeaderName    = "X-CSRF-Token"
	sessionTTL        = 24 * time.Hour
	sessionIdleTTL    = 4 * time.Hour
)

type Session struct {
	ID           string
	CSRFToken    string
	CreatedAt    time.Time
	ExpiresAt    time.Time
	LastActiveAt time.Time
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

// CheckOrigin verifies that browser-originated requests target the exact Host
// seen by the HTTP server. Direct non-browser HTTP clients may omit Origin,
// while WebSocket upgrades must always provide it.
func CheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		referer := r.Header.Get("Referer")
		if referer == "" {
			if websocket.IsWebSocketUpgrade(r) || strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
				return false
			}
			return true
		}
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		origin = u.Scheme + "://" + u.Host
	}
	return originpolicy.Validate(r.Host, origin)
}

func (a *AuthManager) ValidateOrigin(r *http.Request) bool {
	return CheckOrigin(r)
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

	now := time.Now()
	sess := &Session{
		ID:           sessID,
		CSRFToken:    csrfToken,
		CreatedAt:    now,
		ExpiresAt:    now.Add(sessionTTL),
		LastActiveAt: now,
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

	a.mu.Lock()
	defer a.mu.Unlock()

	sess, ok := a.sessions[cookie.Value]
	if !ok {
		return nil
	}

	now := time.Now()
	// Check absolute expiry and idle TTL (A8 security requirement)
	if now.After(sess.ExpiresAt) || now.Sub(sess.LastActiveAt) > sessionIdleTTL {
		delete(a.sessions, cookie.Value)
		return nil
	}

	sess.LastActiveAt = now
	return sess
}

// RevokeSession destroys an existing authenticated session (logout).
func (a *AuthManager) RevokeSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.sessions, sessionID)
}

// RotateSession atomically creates a new session ID/CSRF token and deletes the old one.
func (a *AuthManager) RotateSession(oldSessionID string) (*Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Generate the complete replacement session before revoking the old one.
	// A transient entropy failure must never log out an otherwise valid user.
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

	now := time.Now()
	sess := &Session{
		ID:           sessID,
		CSRFToken:    csrfToken,
		CreatedAt:    now,
		ExpiresAt:    now.Add(sessionTTL),
		LastActiveAt: now,
	}
	delete(a.sessions, oldSessionID)
	a.sessions[sessID] = sess
	return sess, nil
}

func (a *AuthManager) ExchangeBootstrapToken(token string) (*Session, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.usedBootstrap || token == "" || token != a.bootstrapToken {
		return nil, false
	}

	// Generate all session entropy before consuming the one-time bootstrap token.
	// A transient CSPRNG failure must fail closed without burning the token.
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, false
	}
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	if _, err := rand.Read(csrfBytes); err != nil {
		return nil, false
	}
	csrfToken := hex.EncodeToString(csrfBytes)

	// Mark bootstrap token as consumed only after a complete session can be created.
	a.usedBootstrap = true

	now := time.Now()
	sess := &Session{
		ID:           sessID,
		CSRFToken:    csrfToken,
		CreatedAt:    now,
		ExpiresAt:    now.Add(sessionTTL),
		LastActiveAt: now,
	}
	a.sessions[sessID] = sess
	return sess, true
}
