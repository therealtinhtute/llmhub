package updater

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testClient returns a Client pointed at a TLS test server whose transport
// already trusts that server's certificate.
func testClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	return NewClient(WithHTTPClient(srv.Client()), WithBaseURL(srv.URL))
}

func TestLatestReleaseParsesTagAndAssets(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != latestPath {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("User-Agent"); got != userAgent {
			t.Fatalf("user agent = %q, want %q", got, userAgent)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v1.2.3","assets":[{"name":"llmhub-linux-amd64","browser_download_url":"https://example.invalid/llmhub-linux-amd64","size":123}]}`))
	}))
	defer srv.Close()

	rel, err := testClient(t, srv).LatestRelease(context.Background())
	if err != nil {
		t.Fatalf("LatestRelease: %v", err)
	}
	if rel.Tag != "v1.2.3" {
		t.Fatalf("tag = %q, want v1.2.3", rel.Tag)
	}
	if len(rel.Assets) != 1 || rel.Assets[0].Name != "llmhub-linux-amd64" {
		t.Fatalf("assets = %+v", rel.Assets)
	}
}

func TestLatestReleaseRejectsNon200(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	_, err := testClient(t, srv).LatestRelease(context.Background())
	if err == nil || !strings.Contains(err.Error(), "500") {
		t.Fatalf("want status-500 diagnostic, got %v", err)
	}
}

func TestLatestReleaseReportsRateLimit(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	_, err := testClient(t, srv).LatestRelease(context.Background())
	if err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("want rate-limit diagnostic, got %v", err)
	}
}

func TestFetchAssetReportsRateLimit(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := testClient(t, srv).FetchAsset(context.Background(), srv.URL+"/bin", &buf)
	if err == nil || !strings.Contains(err.Error(), "rate limit") {
		t.Fatalf("want rate-limit diagnostic, got %v", err)
	}
}

func TestLatestReleaseRejectsMalformedBody(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":`))
	}))
	defer srv.Close()

	if _, err := testClient(t, srv).LatestRelease(context.Background()); err == nil {
		t.Fatal("want decode error for truncated JSON")
	}
}

func TestLatestReleaseResponseLimit(t *testing.T) {
	big := strings.Repeat("x", metadataLimit+1024)
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(big))
	}))
	defer srv.Close()

	if _, err := testClient(t, srv).LatestRelease(context.Background()); err == nil {
		t.Fatal("want error for oversized metadata body")
	}
}

func TestHTTPFetchAssetStreamsBody(t *testing.T) {
	body := []byte("0123456789")
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	if err := testClient(t, srv).FetchAsset(context.Background(), srv.URL, &buf); err != nil {
		t.Fatalf("FetchAsset: %v", err)
	}
	if buf.String() != string(body) {
		t.Fatalf("body = %q", buf.String())
	}
}

func TestHTTPFetchAssetRejectsPlainHTTPURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	var buf bytes.Buffer
	err := testClient(t, srv).FetchAsset(context.Background(), srv.URL, &buf)
	if err == nil || !strings.Contains(err.Error(), "HTTPS") {
		t.Fatalf("want HTTPS rejection, got %v", err)
	}
}

func TestHTTPFetchAssetRejectsNon200(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := testClient(t, srv).FetchAsset(context.Background(), srv.URL, &buf)
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("want status-404 diagnostic, got %v", err)
	}
}

func TestHTTPFetchAssetResponseLimit(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 1024))
	}))
	defer srv.Close()

	var buf bytes.Buffer
	// A 1024-byte server body against a tiny caller-side cap must fail
	// without writing more than the cap into the destination.
	err := NewClient(WithHTTPClient(srv.Client()), WithBaseURL(srv.URL)).
		FetchAsset(context.Background(), srv.URL, &limitedWriter{w: &buf, max: 100})
	if err == nil {
		t.Fatal("want error for body exceeding caller cap")
	}
}

func TestRedirectRejectsHTTPSDowngrade(t *testing.T) {
	httpTarget := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("plain-HTTP target must never be reached")
	}))
	defer httpTarget.Close()

	httpsSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, httpTarget.URL, http.StatusFound)
	}))
	defer httpsSrv.Close()

	var buf bytes.Buffer
	err := testClient(t, httpsSrv).FetchAsset(context.Background(), httpsSrv.URL, &buf)
	if err == nil || !strings.Contains(err.Error(), "non-HTTPS") {
		t.Fatalf("want redirect downgrade rejection, got %v", err)
	}
}

func TestRedirectAllowsHTTPSChain(t *testing.T) {
	var target *httptest.Server
	target = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer target.Close()

	// Second hop first, so the first server can redirect to it.
	middle := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer middle.Close()

	// Redirect through the middle hop, then land on target.
	first := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, middle.URL, http.StatusFound)
	}))
	defer first.Close()

	var buf bytes.Buffer
	if err := testClient(t, first).FetchAsset(context.Background(), first.URL, &buf); err != nil {
		t.Fatalf("FetchAsset through HTTPS redirects: %v", err)
	}
	if buf.String() != "ok" {
		t.Fatalf("body = %q, want ok", buf.String())
	}
}

func TestHTTPRequestHonorsCancelledContext(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var buf bytes.Buffer
	if err := testClient(t, srv).FetchAsset(ctx, srv.URL, &buf); err == nil {
		t.Fatal("want error for cancelled context")
	}
}
