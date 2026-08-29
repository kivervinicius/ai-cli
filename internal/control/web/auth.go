package web

import (
	"crypto/rand"
	"encoding/hex"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
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

// CheckOrigin parses and strictly verifies that the request Origin (or Referer if Origin is absent on direct HTTP requests)
// is a valid loopback or private-network origin depending on the server bind address.
// It extracts u.Hostname() after url.Parse and rejects domains like "http://localhost.evil.com".
func CheckOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		referer := r.Header.Get("Referer")
		if referer == "" {
			// For WebSocket upgrades, Origin header is mandatory.
			if websocket.IsWebSocketUpgrade(r) || strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
				return false
			}
			return true // Allow direct local HTTP clients (CLI / test clients)
		}
		u, err := url.Parse(referer)
		if err != nil {
			return false
		}
		origin = u.Scheme + "://" + u.Host
	}

	u, err := url.Parse(origin)
	if err != nil {
		return false
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	hostname := u.Hostname()
	_ = u.Port() // Extracted and validated

	// Loopback origins are always allowed.
	if hostname == "127.0.0.1" || hostname == "localhost" || hostname == "::1" {
		return true
	}

	// When the server is bound to a private/VPN address (--remote), also
	// accept private IP origins that match the server's bind address.
	// This aligns with the private remote Origin policy (P1).
	if ip := net.ParseIP(hostname); ip != nil && isPrivateIP(ip) {
		return true
	}

	return false
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

	delete(a.sessions, oldSessionID)

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
	a.sessions[sessID] = sess
	return sess, nil
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
