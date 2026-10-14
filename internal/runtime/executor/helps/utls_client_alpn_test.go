package helps

import (
	stdtls "crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	tls "github.com/refraction-networking/utls"
)

// alpnCloseTracker observes when the underlying TCP connection is closed.
type alpnCloseTracker struct {
	net.Conn
	closed chan struct{}
	once   sync.Once
}

func (c *alpnCloseTracker) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { close(c.closed) })
	return err
}

func newALPNTestServer(t *testing.T, protocols []string, enableH2 bool) *httptest.Server {
	t.Helper()
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("X-Echo-Proto", r.Proto)
		w.Header().Set("X-Echo-Test", r.Header.Get("X-Test"))
		_, _ = io.WriteString(w, "alpn-ok:"+r.Proto+":"+string(body))
	}))
	server.EnableHTTP2 = enableH2
	server.TLS = &stdtls.Config{NextProtos: protocols}
	server.StartTLS()
	t.Cleanup(server.Close)
	return server
}

// trustingALPNDialer dials with the test server's certificate trusted and
// tracks connection closure.
func trustingALPNDialer(t *testing.T, server *httptest.Server, tracked chan *alpnCloseTracker) func(host, addr string) (*tls.UConn, error) {
	t.Helper()
	pool := x509.NewCertPool()
	pool.AddCert(server.Certificate())
	return func(host, addr string) (*tls.UConn, error) {
		raw, err := net.Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		tracker := &alpnCloseTracker{Conn: raw, closed: make(chan struct{})}
		if tracked != nil {
			select {
			case tracked <- tracker:
			default:
			}
		}
		conn := tls.UClient(tracker, &tls.Config{ServerName: host, RootCAs: pool}, tls.HelloChrome_Auto)
		if err := conn.Handshake(); err != nil {
			raw.Close()
			return nil, err
		}
		return conn, nil
	}
}

func newALPNTestRequest(t *testing.T, server *httptest.Server) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, server.URL+"/calls?intent=test", strings.NewReader("intent=test"))
	if err != nil {
		t.Fatal(err)
	}
	req.RequestURI = ""
	req.Header.Set("X-Test", "preserved")
	return req
}

func TestUtlsRoundTripperUsesH2WhenNegotiated(t *testing.T) {
	server := newALPNTestServer(t, []string{"h2"}, true)
	rt := newUtlsRoundTripper("")
	rt.dialTLS = trustingALPNDialer(t, server, nil)

	resp, err := rt.RoundTrip(newALPNTestRequest(t, server))
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d body=%s", resp.StatusCode, http.StatusOK, body)
	}
	if resp.ProtoMajor != 2 {
		t.Fatalf("ProtoMajor = %d, want 2 body=%s", resp.ProtoMajor, body)
	}
	if !strings.Contains(string(body), "alpn-ok:HTTP/2.0:intent=test") {
		t.Fatalf("unexpected body %q", body)
	}

	// A second request must reuse the pooled HTTP/2 connection.
	resp2, err := rt.RoundTrip(newALPNTestRequest(t, server))
	if err != nil {
		t.Fatalf("second RoundTrip() error = %v", err)
	}
	io.Copy(io.Discard, resp2.Body)
	resp2.Body.Close()
	rt.mu.Lock()
	pooled := len(rt.connections)
	rt.mu.Unlock()
	if pooled != 1 {
		t.Fatalf("pooled connections = %d, want 1", pooled)
	}
}

func TestUtlsRoundTripperFallsBackToHTTP1(t *testing.T) {
	for _, tc := range []struct {
		name      string
		protocols []string
	}{
		{name: "http/1.1", protocols: []string{"http/1.1"}},
		{name: "empty-alpn", protocols: []string{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := newALPNTestServer(t, tc.protocols, false)
			tracked := make(chan *alpnCloseTracker, 4)
			rt := newUtlsRoundTripper("")
			rt.dialTLS = trustingALPNDialer(t, server, tracked)

			resp, err := rt.RoundTrip(newALPNTestRequest(t, server))
			if err != nil {
				t.Fatalf("RoundTrip() error = %v", err)
			}
			body, _ := io.ReadAll(resp.Body)
			if resp.ProtoMajor != 1 {
				t.Fatalf("ProtoMajor = %d, want 1 body=%s", resp.ProtoMajor, body)
			}
			if resp.Header.Get("X-Echo-Test") != "preserved" {
				t.Fatalf("X-Test not preserved: headers=%v body=%s", resp.Header, body)
			}
			if !strings.Contains(string(body), "alpn-ok:HTTP/1.1:intent=test") {
				t.Fatalf("unexpected body %q", body)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatalf("Body.Close() error = %v", err)
			}

			// HTTP/1.1 connections are never pooled.
			rt.mu.Lock()
			pooled := len(rt.connections)
			rt.mu.Unlock()
			if pooled != 0 {
				t.Fatalf("pooled connections = %d, want 0", pooled)
			}

			// The single-use connection is released when the body closes.
			select {
			case tracker := <-tracked:
				select {
				case <-tracker.closed:
				case <-time.After(5 * time.Second):
					t.Fatal("HTTP/1.1 connection was not closed after body close")
				}
			case <-time.After(5 * time.Second):
				t.Fatal("no connection was dialed")
			}
		})
	}
}

func TestUtlsCreateConnectionReturnsFallbackForHTTP1(t *testing.T) {
	server := newALPNTestServer(t, []string{"http/1.1"}, false)
	rt := newUtlsRoundTripper("")
	rt.dialTLS = trustingALPNDialer(t, server, nil)

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, err = rt.createConnection(serverURL.Hostname(), serverURL.Host)
	var fallback *http1FallbackConn
	if !errors.As(err, &fallback) || fallback.conn == nil {
		t.Fatalf("createConnection() err = %v, want *http1FallbackConn with live conn", err)
	}
	defer fallback.conn.Close()
	if got := fallback.conn.ConnectionState().NegotiatedProtocol; got != "http/1.1" {
		t.Fatalf("negotiated protocol = %q, want http/1.1", got)
	}
}
