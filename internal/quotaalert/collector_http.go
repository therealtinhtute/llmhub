package quotaalert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultCollectorHTTPTimeout      = 15 * time.Second
	DefaultCollectorMaxResponseBytes = 1 << 20
)

// CollectorRefreshFunc refreshes an auth snapshot after an authentication challenge.
type CollectorRefreshFunc func(context.Context, AuthSnapshot) error

// CollectorHTTPConfig describes one fixed-host collector HTTP boundary.
type CollectorHTTPConfig struct {
	BaseURL      string
	AllowedHosts []string
	Timeout      time.Duration
	MaxBodyBytes int64
	Client       *http.Client
}

// CollectorHTTPClient performs bounded fixed-host JSON requests for collectors.
type CollectorHTTPClient struct {
	baseURL      *url.URL
	allowedHosts map[string]struct{}
	timeout      time.Duration
	maxBodyBytes int64
	client       *http.Client
}

// NewCollectorHTTPClient validates and creates a collector HTTP helper.
func NewCollectorHTTPClient(config CollectorHTTPConfig) (*CollectorHTTPClient, error) {
	base, err := url.Parse(strings.TrimSpace(config.BaseURL))
	if err != nil || base == nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("quota collector base URL is invalid")
	}
	if base.Scheme != "https" && base.Scheme != "http" {
		return nil, fmt.Errorf("quota collector base URL scheme is invalid")
	}
	base.RawQuery = ""
	base.Fragment = ""

	allowed := make(map[string]struct{}, len(config.AllowedHosts)+1)
	baseHost := canonicalHost(base.Host)
	allowed[baseHost] = struct{}{}
	for _, host := range config.AllowedHosts {
		host = canonicalHost(host)
		if host == "" {
			return nil, fmt.Errorf("quota collector allowed host is invalid")
		}
		allowed[host] = struct{}{}
	}
	if _, ok := allowed[baseHost]; !ok {
		return nil, fmt.Errorf("quota collector base host is not allowed")
	}

	timeout := config.Timeout
	if timeout == 0 {
		timeout = DefaultCollectorHTTPTimeout
	}
	if timeout < time.Second || timeout > time.Minute {
		return nil, fmt.Errorf("quota collector HTTP timeout must be between 1s and 1m")
	}
	maxBodyBytes := config.MaxBodyBytes
	if maxBodyBytes == 0 {
		maxBodyBytes = DefaultCollectorMaxResponseBytes
	}
	if maxBodyBytes < 1 || maxBodyBytes > 8<<20 {
		return nil, fmt.Errorf("quota collector response bound is invalid")
	}
	client := config.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &CollectorHTTPClient{
		baseURL:      base,
		allowedHosts: allowed,
		timeout:      timeout,
		maxBodyBytes: maxBodyBytes,
		client:       client,
	}, nil
}

// JSON sends one relative-path request and decodes a bounded JSON response.
func (c *CollectorHTTPClient) JSON(ctx context.Context, auth AuthSnapshot, method, path string, headers map[string]string, out any, refresh CollectorRefreshFunc) error {
	return c.JSONBody(ctx, auth, method, path, headers, nil, out, refresh)
}

// JSONBody sends one relative-path request with an optional JSON body and decodes a bounded JSON response.
func (c *CollectorHTTPClient) JSONBody(ctx context.Context, auth AuthSnapshot, method, path string, headers map[string]string, body any, out any, refresh CollectorRefreshFunc) error {
	if c == nil {
		return fmt.Errorf("quota collector HTTP client is nil")
	}
	if out == nil {
		return fmt.Errorf("quota collector JSON target is nil")
	}
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("quota collector request body is invalid")
		}
		if int64(len(bodyBytes)) > c.maxBodyBytes {
			return fmt.Errorf("quota collector request body exceeds %d bytes", c.maxBodyBytes)
		}
	}
	for attempt := 0; attempt < 2; attempt++ {
		status, err := c.doJSON(ctx, auth, method, path, headers, bodyBytes, out)
		if err == nil {
			return nil
		}
		if status != http.StatusUnauthorized || attempt == 1 || refresh == nil {
			return err
		}
		if refreshErr := refresh(ctx, auth); refreshErr != nil {
			return fmt.Errorf("quota collector refresh failed: %s", RedactCollectorError(refreshErr, auth))
		}
	}
	return nil
}

func (c *CollectorHTTPClient) doJSON(ctx context.Context, auth AuthSnapshot, method, path string, headers map[string]string, body []byte, out any) (int, error) {
	requestURL, err := c.resolve(path)
	if err != nil {
		return 0, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var requestBody io.Reader
	if len(body) > 0 {
		requestBody = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(requestCtx, method, requestURL.String(), requestBody)
	if err != nil {
		return 0, fmt.Errorf("quota collector request is invalid")
	}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	client := c.clientFor(auth)
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("quota collector request failed: %s", RedactCollectorError(err, auth))
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		return resp.StatusCode, fmt.Errorf("quota collector request returned HTTP %d", resp.StatusCode)
	}
	reader := io.LimitReader(resp.Body, c.maxBodyBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return resp.StatusCode, fmt.Errorf("quota collector response read failed")
	}
	if int64(len(data)) > c.maxBodyBytes {
		return resp.StatusCode, fmt.Errorf("quota collector response exceeds %d bytes", c.maxBodyBytes)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return resp.StatusCode, fmt.Errorf("quota collector response JSON is invalid")
	}
	return resp.StatusCode, nil
}

func (c *CollectorHTTPClient) resolve(path string) (*url.URL, error) {
	path = strings.TrimSpace(path)
	parsed, err := url.Parse(path)
	if err != nil || parsed == nil || parsed.IsAbs() || parsed.Host != "" || !strings.HasPrefix(path, "/") {
		return nil, fmt.Errorf("quota collector request path must be relative to its fixed host")
	}
	resolved := *c.baseURL
	resolved.Path = strings.TrimRight(c.baseURL.Path, "/") + parsed.Path
	resolved.RawQuery = parsed.RawQuery
	resolved.Fragment = ""
	if _, ok := c.allowedHosts[canonicalHost(resolved.Host)]; !ok {
		return nil, fmt.Errorf("quota collector request host is not allowed")
	}
	return &resolved, nil
}

func (c *CollectorHTTPClient) clientFor(auth AuthSnapshot) *http.Client {
	if auth == nil || strings.TrimSpace(auth.ProxyURL()) == "" {
		return c.client
	}
	proxyURL, err := url.Parse(strings.TrimSpace(auth.ProxyURL()))
	if err != nil || proxyURL.Scheme == "" || proxyURL.Host == "" {
		return c.client
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(proxyURL)
	return &http.Client{Transport: transport, Timeout: c.timeout}
}

func canonicalHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return ""
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	return strings.Trim(host, ".")
}

// RedactCollectorError removes known credential-like snapshot values from errors.
func RedactCollectorError(err error, auth AuthSnapshot) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if auth == nil {
		return message
	}
	for _, key := range []string{"access_token", "refresh_token", "id_token", "api_key", "token", "authorization", "cookie"} {
		if value, ok := auth.Attribute(key); ok {
			message = redactValue(message, value)
		}
		if value, ok := auth.Metadata(key); ok {
			if text, textOK := value.(string); textOK {
				message = redactValue(message, text)
			}
		}
	}
	return message
}

func redactValue(message, value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 4 {
		return message
	}
	return strings.ReplaceAll(message, value, "[redacted]")
}
