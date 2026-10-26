package gokotakneo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// HTTPClient defines the transport interface seam for test mocks and proxy injection.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// Transport manages HTTP connections, keep-alives, headers, and jData form-encoding.
type Transport struct {
	client     HTTPClient
	baseURI    string
	neoFinKey  string
	authToken  string
	sessionSid string
	configErr  error
	mu         sync.RWMutex
}

// NewDefaultTransport creates a production-grade transport with pooled keep-alives.
func NewDefaultTransport(baseURI string) *Transport {
	return &Transport{
		baseURI:   strings.TrimRight(baseURI, "/"),
		neoFinKey: "neotradeapi",
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConnsPerHost:   50,
				ResponseHeaderTimeout: 5 * time.Second,
				IdleConnTimeout:       90 * time.Second,
			},
		},
	}
}

// SetBaseURI updates the active target gateway base URL.
func (t *Transport) SetBaseURI(uri string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.baseURI = strings.TrimRight(uri, "/")
}

// SetSessionHeaders updates the session credentials for subsequent trading calls.
func (t *Transport) SetSessionHeaders(token, sid string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.authToken = token
	t.sessionSid = sid
}

// ExecuteJData encodes payload into form-encoded `jData` parameter and executes request.
func (t *Transport) ExecuteJData(ctx context.Context, method, endpoint string, payload interface{}, out interface{}) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal jData payload: %w", err)
	}

	form := url.Values{}
	form.Set("jData", string(jsonBytes))

	return t.ExecuteForm(ctx, method, endpoint, form, out)
}

// ExecuteForm dispatches form-encoded request with standard headers.
func (t *Transport) ExecuteForm(ctx context.Context, method, endpoint string, form url.Values, out interface{}) error {
	body := bytes.NewReader([]byte(form.Encode()))
	headers := http.Header{}
	headers.Set("Content-Type", "application/x-www-form-urlencoded")
	return t.executeRaw(ctx, method, endpoint, body, headers, out)
}

// ExecuteJSON dispatches JSON-encoded request.
func (t *Transport) ExecuteJSON(ctx context.Context, method, endpoint string, body interface{}, headers http.Header, out interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal json body: %w", err)
		}
		bodyReader = bytes.NewReader(b)
	}

	if headers == nil {
		headers = http.Header{}
	}
	headers.Set("Content-Type", "application/json")

	return t.executeRaw(ctx, method, endpoint, bodyReader, headers, out)
}

// ExecuteRawRequest executes arbitrary HTTP requests.
func (t *Transport) executeRaw(ctx context.Context, method, endpoint string, body io.Reader, headers http.Header, out interface{}) error {
	if t.configErr != nil {
		return t.configErr
	}

	t.mu.RLock()
	base := t.baseURI
	token := t.authToken
	sid := t.sessionSid
	finKey := t.neoFinKey
	t.mu.RUnlock()

	fullURL := base + endpoint
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	// Default Headers
	req.Header.Set("accept", "application/json")
	if finKey != "" {
		req.Header.Set("neo-fin-key", finKey)
	}
	if token != "" {
		req.Header.Set("Auth", token)
	}
	if sid != "" {
		req.Header.Set("Sid", sid)
	}

	// Merge Custom Headers
	for k, v := range headers {
		req.Header[k] = v
	}

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrBrokerUnavailable, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// Check if body is a Kotak error response
	if resp.StatusCode >= http.StatusBadRequest || strings.Contains(string(respBytes), `"stat":"Not_Ok"`) || strings.Contains(string(respBytes), `"stat": "Not_Ok"`) {
		return ParseAPIError(resp.StatusCode, respBytes)
	}

	if out != nil {
		return json.Unmarshal(respBytes, out)
	}

	return nil
}
