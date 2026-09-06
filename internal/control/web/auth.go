package web

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
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
	desktopSession *Session
	listenHost     string
	listenPort     string
	storeDir       string
	lastPersist    time.Time
	entropy        io.Reader
}

func (a *AuthManager) SetDesktopSession(sess *Session) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.desktopSession != nil && (sess == nil || a.desktopSession.ID != sess.ID) {
		delete(a.sessions, a.desktopSession.ID)
	}
	a.desktopSession = sess
	if sess != nil {
		a.sessions[sess.ID] = sess
	}
	_ = a.persistLocked()
}

func (a *AuthManager) GetDesktopSession() *Session {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.desktopSession == nil {
		return nil
	}
	if _, exists := a.sessions[a.desktopSession.ID]; !exists {
		a.desktopSession = nil
		return nil
	}
	now := time.Now()
	if now.After(a.desktopSession.ExpiresAt) || now.Sub(a.desktopSession.LastActiveAt) > sessionIdleTTL {
		delete(a.sessions, a.desktopSession.ID)
		a.desktopSession = nil
		_ = a.persistLocked()
		return nil
	}
	return a.desktopSession
}

func NewAuthManager(listenHost, listenPort string) (*AuthManager, string, error) {
	return NewAuthManagerWithStore(listenHost, listenPort, "")
}

// NewAuthManagerWithStore restores loopback sessions from storeDir so a
// browser cookie survives `nexus web` restart on this machine.
func NewAuthManagerWithStore(listenHost, listenPort, storeDir string) (*AuthManager, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return nil, "", err
	}
	bootstrapToken := hex.EncodeToString(bytes)

	auth := &AuthManager{
		bootstrapToken: bootstrapToken,
		sessions:       make(map[string]*Session),
		listenHost:     listenHost,
		listenPort:     listenPort,
		storeDir:       strings.TrimSpace(storeDir),
		entropy:        rand.Reader,
	}
	auth.mu.Lock()
	auth.loadPersistedLocked()
	auth.mu.Unlock()
	return auth, bootstrapToken, nil
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
	if _, err := io.ReadFull(a.entropyReader(), bytes); err != nil {
		return nil, err
	}
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	if _, err := io.ReadFull(a.entropyReader(), csrfBytes); err != nil {
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
	_ = a.persistLocked()
	return sess, nil
}

func (a *AuthManager) AuthenticateRequest(r *http.Request) *Session {
	var token string
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		token = cookie.Value
	}
	if token == "" {
		if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
		}
	}
	if token == "" {
		token = strings.TrimSpace(r.Header.Get("X-Nexus-Session"))
	}
	if token == "" && (websocket.IsWebSocketUpgrade(r) || strings.EqualFold(r.Header.Get("Upgrade"), "websocket")) {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
		if token == "" {
			token = strings.TrimSpace(r.URL.Query().Get("session"))
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	// If no explicit token or cookie was provided, check if this request originates from the local native desktop shell
	if token == "" && a.desktopSession != nil {
		origin := r.Header.Get("Origin")
		referer := r.Header.Get("Referer")

		if originpolicy.IsTrustedDesktopRequest(r.Host, origin, referer) {
			// Ensure desktop session still exists and has not expired or idled out
			if _, exists := a.sessions[a.desktopSession.ID]; !exists {
				a.desktopSession = nil
				return nil
			}
			now := time.Now()
			if now.After(a.desktopSession.ExpiresAt) || now.Sub(a.desktopSession.LastActiveAt) > sessionIdleTTL {
				delete(a.sessions, a.desktopSession.ID)
				a.desktopSession = nil
				_ = a.persistLocked()
				return nil
			}
			a.desktopSession.LastActiveAt = now
			if a.storeDir != "" && now.Sub(a.lastPersist) >= sessionPersistMinGap {
				_ = a.persistLocked()
			}
			return a.desktopSession
		}
	}

	if token == "" {
		return nil
	}

	sess, ok := a.sessions[token]
	if !ok {
		return nil
	}

	now := time.Now()
	// Check absolute expiry and idle TTL (A8 security requirement)
	if now.After(sess.ExpiresAt) || now.Sub(sess.LastActiveAt) > sessionIdleTTL {
		if a.desktopSession != nil && a.desktopSession.ID == token {
			a.desktopSession = nil
		}
		delete(a.sessions, token)
		_ = a.persistLocked()
		return nil
	}

	sess.LastActiveAt = now
	if a.storeDir != "" && now.Sub(a.lastPersist) >= sessionPersistMinGap {
		_ = a.persistLocked()
	}
	return sess
}

// RevokeSession destroys an existing authenticated session (logout).
func (a *AuthManager) RevokeSession(sessionID string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.desktopSession != nil && a.desktopSession.ID == sessionID {
		a.desktopSession = nil
	}
	delete(a.sessions, sessionID)
	_ = a.persistLocked()
}

// RotateSession atomically creates a new session ID/CSRF token and deletes the old one.
func (a *AuthManager) RotateSession(oldSessionID string) (*Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	bytes := make([]byte, 32)
	if _, err := io.ReadFull(a.entropyReader(), bytes); err != nil {
		return nil, err
	}
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	if _, err := io.ReadFull(a.entropyReader(), csrfBytes); err != nil {
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
	if a.desktopSession != nil && a.desktopSession.ID == oldSessionID {
		a.desktopSession = sess
	}
	_ = a.persistLocked()
	return sess, nil
}

func (a *AuthManager) ExchangeBootstrapToken(token string) (*Session, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if token == "" || token != a.bootstrapToken {
		return nil, false
	}
	// Remote/private binds keep one-time consume. Loopback reuses the printed
	// Bootstrap URL for the life of the process so local re-auth does not
	// require restarting `nexus web` after a cookie loss or accidental consume.
	loopback := a.isLoopbackListen()
	if a.usedBootstrap && !loopback {
		return nil, false
	}

	bytes := make([]byte, 32)
	if _, err := io.ReadFull(a.entropyReader(), bytes); err != nil {
		return nil, false
	}
	sessID := hex.EncodeToString(bytes)

	csrfBytes := make([]byte, 16)
	if _, err := io.ReadFull(a.entropyReader(), csrfBytes); err != nil {
		return nil, false
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
	if !loopback {
		a.usedBootstrap = true
	}
	a.sessions[sessID] = sess
	_ = a.persistLocked()
	return sess, true
}

func isLoopbackHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" || strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func (a *AuthManager) isLoopbackListen() bool {
	return isLoopbackHost(a.listenHost)
}

func (a *AuthManager) entropyReader() io.Reader {
	if a.entropy != nil {
		return a.entropy
	}
	return rand.Reader
}
