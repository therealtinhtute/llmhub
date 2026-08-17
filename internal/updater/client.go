// Package updater discovers, verifies, probes, and stages LLMHub release
// binaries for a later root-run apply step. All remote data is untrusted.
package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Production endpoints and bounds. Kept private; tests override the base URL
// and HTTP client through options.
const (
	apiBaseURL    = "https://api.github.com/repos/therealtinhtute/llmhub"
	latestPath    = "/releases/latest"
	userAgent     = "LLMHub"
	metadataLimit = 4 << 20 // release metadata JSON
	assetLimit    = 256 << 20 // release binary or checksums manifest
	metaTimeout   = 10 * time.Second
	assetTimeout  = 5 * time.Minute
)

// Asset is a single entry in a GitHub release's assets list.
type Asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

// Release is the untrusted latest-release metadata.
type Release struct {
	Tag    string  `json:"tag_name"`
	Assets []Asset `json:"assets"`
}

// Client fetches release metadata and assets over HTTPS with bounded
// redirects, timeouts, and response sizes.
type Client struct {
	http      *http.Client
	baseURL   string
	userAgent string
}

// Option configures a Client. Only the seams tests need are exposed.
type Option func(*Client)

// WithHTTPClient replaces the default HTTP client (test seam).
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) { cl.http = c }
}

// WithBaseURL overrides the release API base URL (test seam).
func WithBaseURL(base string) Option {
	return func(cl *Client) { cl.baseURL = strings.TrimRight(base, "/") }
}

// NewClient returns a Client using the default production endpoint. The
// default transport honors HTTP(S)_PROXY and NO_PROXY environment variables.
func NewClient(opts ...Option) *Client {
	c := &Client{
		http:      &http.Client{Timeout: assetTimeout},
		baseURL:   apiBaseURL,
		userAgent: userAgent,
	}
	for _, opt := range opts {
		opt(c)
	}
	// Redirects must stay on HTTPS: reject any downgrade to plain HTTP.
	c.http.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("redirect to non-HTTPS URL %q rejected", req.URL.String())
		}
		if len(via) >= 10 {
			return errors.New("too many redirects")
		}
		return nil
	}
	return c
}

// LatestRelease fetches and decodes the latest stable release metadata.
func (c *Client) LatestRelease(ctx context.Context) (Release, error) {
	var rel Release
	err := c.get(ctx, c.baseURL+latestPath, metaTimeout, metadataLimit, &rel, nil)
	return rel, err
}

// FetchAsset streams the asset at u into dst under the asset timeout and
// size limit. It verifies the response status and rejects non-HTTPS URLs.
func (c *Client) FetchAsset(ctx context.Context, u string, dst io.Writer) error {
	parsed, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid asset URL: %w", err)
	}
	if parsed.Scheme != "https" {
		return fmt.Errorf("asset URL must be HTTPS, got %q", parsed.Scheme)
	}
	return c.get(ctx, u, assetTimeout, assetLimit, nil, dst)
}

// ReleaseForTag fetches metadata for one specific release tag. The root
// apply step uses it to re-fetch checksums.txt for the staged version
// instead of trusting anything already on disk (R9).
func (c *Client) ReleaseForTag(ctx context.Context, tag string) (Release, error) {
	var rel Release
	err := c.get(ctx, c.baseURL+"/releases/tags/"+url.PathEscape(tag), metaTimeout, metadataLimit, &rel, nil)
	return rel, err
}

// FetchMetadata streams a small release-metadata document (e.g. the
// checksums.txt manifest) into dst under the metadata size limit.
func (c *Client) FetchMetadata(ctx context.Context, u string, dst io.Writer) error {
	return c.get(ctx, u, metaTimeout, metadataLimit, nil, dst)
}

// get performs a bounded GET. When out is non-nil the JSON body is decoded
// into it; otherwise the raw body is copied to dst. A body larger than limit
// fails without buffering past the limit.
func (c *Client) get(ctx context.Context, u string, timeout time.Duration, limit int64, out any, dst io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("unexpected HTTP status %d from %s", resp.StatusCode, u)
	}

	if out != nil {
		body := io.LimitReader(resp.Body, limit+1)
		if err := json.NewDecoder(body).Decode(out); err != nil {
			return fmt.Errorf("decoding response body: %w", err)
		}
		return nil
	}
	if _, err := io.Copy(&limitedWriter{w: dst, max: limit}, resp.Body); err != nil {
		return err
	}
	return nil
}

// limitedWriter fails once more than max bytes would be written.
type limitedWriter struct {
	w   io.Writer
	n   int64
	max int64
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	if lw.n+int64(len(p)) > lw.max {
		return 0, fmt.Errorf("response body exceeds %d-byte limit", lw.max)
	}
	n, err := lw.w.Write(p)
	lw.n += int64(n)
	return n, err
}
