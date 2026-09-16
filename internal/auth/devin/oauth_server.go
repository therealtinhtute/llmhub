package devin

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// OAuthServer handles local loopback HTTP callbacks for Devin authentication.
// Ported from upstream CLIProxyAPI internal/auth/devin/devin_auth.go
// (f94752762bb9; ctx-bound wait from 5d0c77cf).
type OAuthServer struct {
	server     *http.Server
	listener   net.Listener
	port       int
	resultChan chan *OAuthResult
	errorChan  chan error
	mu         sync.Mutex
	running    bool
}

// OAuthResult carries the authorization code from the browser callback.
type OAuthResult struct {
	Code  string
	State string
	Error string
}

// NewOAuthServer creates a local loopback server for Devin OAuth.
func NewOAuthServer(port int) *OAuthServer {
	return &OAuthServer{
		port:       port,
		resultChan: make(chan *OAuthResult, 1),
		errorChan:  make(chan error, 1),
	}
}

// Start initiates the local HTTP server. If port is 0, an available ephemeral port is chosen.
func (s *OAuthServer) Start() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return s.port, nil
	}

	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	ln, errLn := net.Listen("tcp", addr)
	if errLn != nil {
		return 0, fmt.Errorf("failed to bind local OAuth server to %s: %w", addr, errLn)
	}

	s.listener = ln
	s.port = ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", s.handleCallback)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/callback" {
			s.handleCallback(w, r)
			return
		}
		http.NotFound(w, r)
	})

	s.server = &http.Server{
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	s.running = true
	go func() {
		if errServe := s.server.Serve(ln); errServe != nil && !errors.Is(errServe, http.ErrServerClosed) {
			log.Debugf("devin oauth server error: %v", errServe)
			select {
			case s.errorChan <- errServe:
			default:
			}
		}
	}()

	return s.port, nil
}

// Stop gracefully stops the server.
func (s *OAuthServer) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}
	s.running = false
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// WaitForCallback waits for the browser redirect or times out.
func (s *OAuthServer) WaitForCallback(timeout time.Duration) (*OAuthResult, error) {
	return s.WaitForCallbackWithContext(context.Background(), timeout)
}

// WaitForCallbackWithContext waits for the browser redirect, context cancellation, or times out.
func (s *OAuthServer) WaitForCallbackWithContext(ctx context.Context, timeout time.Duration) (*OAuthResult, error) {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case res := <-s.resultChan:
		return res, nil
	case err := <-s.errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, errors.New("devin authentication timed out")
	}
}

func (s *OAuthServer) handleCallback(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	code := strings.TrimSpace(q.Get("code"))
	state := strings.TrimSpace(q.Get("state"))
	errStr := strings.TrimSpace(q.Get("error"))
	errDesc := strings.TrimSpace(q.Get("error_description"))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if errStr != "" || code == "" {
		errMsg := errStr
		if errDesc != "" {
			errMsg = fmt.Sprintf("%s: %s", errStr, errDesc)
		}
		if errMsg == "" {
			errMsg = "missing authorization code"
		}
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf(loginFailureHTML, html.EscapeString(errMsg))))
		select {
		case s.resultChan <- &OAuthResult{Error: errMsg}:
		default:
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(loginSuccessHTML))

	select {
	case s.resultChan <- &OAuthResult{Code: code, State: state}:
	default:
	}
}
