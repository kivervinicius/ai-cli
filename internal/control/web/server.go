package web

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type ServerOptions struct {
	Host   string
	Port   int
	NoOpen bool
}

type Server struct {
	httpServer *http.Server
	listener   net.Listener
	auth       *AuthManager
	api        *APIHandler
	terminal   *TerminalHub
	bootstrap  string
	url        string
}

func NewServer(opts ServerOptions) (*Server, error) {
	if opts.Host == "" {
		opts.Host = "127.0.0.1"
	}

	addr := net.JoinHostPort(opts.Host, strconv.Itoa(opts.Port))
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to bind to %s: %w", addr, err)
	}

	tcpAddr := l.Addr().(*net.TCPAddr)
	portStr := strconv.Itoa(tcpAddr.Port)

	auth, bootstrapToken, err := NewAuthManager(opts.Host, portStr)
	if err != nil {
		_ = l.Close()
		return nil, fmt.Errorf("failed to initialize auth manager: %w", err)
	}

	api := NewAPIHandler(auth)
	terminalHub := NewTerminalHub(auth)

	s := &Server{
		listener:  l,
		auth:      auth,
		api:       api,
		terminal:  terminalHub,
		bootstrap: bootstrapToken,
		url:       fmt.Sprintf("http://%s:%d", opts.Host, tcpAddr.Port),
	}

	mux := http.NewServeMux()

	// REST API Routes
	mux.HandleFunc("/api/v1/health", api.handleHealth)
	mux.HandleFunc("/api/v1/session", api.handleSession)
	mux.HandleFunc("/api/v1/workspaces", s.authMiddleware(api.handleWorkspaces))
	mux.HandleFunc("/api/v1/runtimes", s.authMiddleware(api.handleRuntimes))
	mux.HandleFunc("/api/v1/runtimes/", s.routeRuntime)
	mux.HandleFunc("/api/v1/providers", s.authMiddleware(api.handleProviders))
	mux.HandleFunc("/api/v1/profiles", s.authMiddleware(api.handleProfiles))
	mux.HandleFunc("/api/v1/events", s.authMiddleware(api.handleEvents))

	// Static Files & SPA Routing
	distFS, distErr := DistFileSystem()
	var fileServer http.Handler
	if distErr == nil {
		fileServer = http.FileServer(distFS)
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// 1. Check for one-time bootstrap token query parameter: ?token=...
		if token := r.URL.Query().Get("token"); token != "" {
			sess, ok := s.auth.ExchangeBootstrapToken(token)
			if ok && sess != nil {
				http.SetCookie(w, &http.Cookie{
					Name:     sessionCookieName,
					Value:    sess.ID,
					Path:     "/",
					HttpOnly: true,
					SameSite: http.SameSiteStrictMode,
				})
				// Redirect to clean URL without the token
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
		}

		// 2. Serve static SPA assets
		if fileServer != nil {
			// If file exists serve it, otherwise serve index.html (SPA routing)
			f, err := distFS.Open(strings.TrimPrefix(r.URL.Path, "/"))
			if err == nil {
				_ = f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}
			// Fallback to index.html
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<h1>AI Control Center</h1><p>Web frontend initializing...</p>"))
	})

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	return s, nil
}

func (s *Server) routeRuntime(w http.ResponseWriter, r *http.Request) {
	// Check for WebSocket terminal route: /api/v1/runtimes/:id/terminal
	if strings.HasSuffix(r.URL.Path, "/terminal") {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/runtimes/"), "/")
		if len(parts) == 2 && parts[1] == "terminal" {
			if !s.auth.ValidateOrigin(r) {
				writeError(w, http.StatusForbidden, "invalid origin")
				return
			}
			if s.auth.AuthenticateRequest(r) == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			runtimeID := parts[0]
			s.terminal.HandleWebSocket(w, r, runtimeID)
			return
		}
	}

	// Normal REST API runtime action
	s.authMiddleware(s.api.handleRuntimeDetail)(w, r)
}

func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.ValidateOrigin(r) {
			writeError(w, http.StatusForbidden, "invalid origin")
			return
		}

		// Enforce authentication on all API routes, including GET requests
		sess := s.auth.AuthenticateRequest(r)
		if sess == nil {
			writeError(w, http.StatusUnauthorized, "authentication required")
			return
		}

		// Non-GET requests require CSRF token validation
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions {
			csrf := r.Header.Get(csrfHeaderName)
			if csrf == "" || csrf != sess.CSRFToken {
				writeError(w, http.StatusForbidden, "invalid CSRF token")
				return
			}
		}

		next(w, r)
	}
}

func (s *Server) URL() string {
	return s.url
}

func (s *Server) BootstrapURL() string {
	return fmt.Sprintf("%s/?token=%s", s.url, s.bootstrap)
}

func (s *Server) Start() error {
	return s.httpServer.Serve(s.listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func OpenBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{url}
	}
	return exec.Command(cmd, args...).Start()
}
