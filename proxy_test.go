package gokotakneo

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Positive Cases: URL construction & defaults
func TestProxyConfig_URL_PositiveCases(t *testing.T) {
	tests := []struct {
		name     string
		cfg      ProxyConfig
		wantURL  string
		wantUser string
		wantPass string
	}{
		{
			name: "Standard HTTPS with full credentials",
			cfg: ProxyConfig{
				Scheme:       "https",
				Host:         "dc46-mum-01.algoip.in",
				Port:         443,
				ClientID:     "algo_user",
				ClientSecret: "secret_pass",
			},
			wantURL:  "https://algo_user:secret_pass@dc46-mum-01.algoip.in:443",
			wantUser: "algo_user",
			wantPass: "secret_pass",
		},
		{
			name: "Default Scheme (empty defaults to https) and Default Port (0 defaults to 443)",
			cfg: ProxyConfig{
				Host:         "staticip.in",
				ClientID:     "user1",
				ClientSecret: "pass1",
			},
			wantURL:  "https://user1:pass1@staticip.in:443",
			wantUser: "user1",
			wantPass: "pass1",
		},
		{
			name: "HTTP Scheme with default port 80",
			cfg: ProxyConfig{
				Scheme:   "http",
				Host:     "127.0.0.1",
				ClientID: "local_user",
			},
			wantURL:  "http://local_user@127.0.0.1:80",
			wantUser: "local_user",
			wantPass: "",
		},
		{
			name: "HTTP Scheme with custom port and credentials",
			cfg: ProxyConfig{
				Scheme:       "http",
				Host:         "proxy.internal",
				Port:         8080,
				ClientID:     "agent",
				ClientSecret: "token99",
			},
			wantURL:  "http://agent:token99@proxy.internal:8080",
			wantUser: "agent",
			wantPass: "token99",
		},
		{
			name: "Unauthenticated proxy",
			cfg: ProxyConfig{
				Scheme: "https",
				Host:   "squid.local",
				Port:   3128,
			},
			wantURL:  "https://squid.local:3128",
			wantUser: "",
			wantPass: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := tc.cfg.URL()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if u.String() != tc.wantURL {
				t.Errorf("u.String() = %q, want %q", u.String(), tc.wantURL)
			}
			if tc.wantUser != "" {
				if u.User == nil || u.User.Username() != tc.wantUser {
					t.Errorf("username = %v, want %q", u.User, tc.wantUser)
				}
				pass, _ := u.User.Password()
				if pass != tc.wantPass {
					t.Errorf("password = %q, want %q", pass, tc.wantPass)
				}
			}
		})
	}
}

// Negative & Edge Cases: Host normalization, trimming, port in host, invalid inputs
func TestProxyConfig_URL_NegativeAndEdgeCases(t *testing.T) {
	t.Run("Edge: Host already contains port", func(t *testing.T) {
		cfg := ProxyConfig{
			Host:         "dc46-mum-01.algoip.in:8443",
			ClientID:     "u",
			ClientSecret: "p",
		}
		u, err := cfg.URL()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Host != "dc46-mum-01.algoip.in:8443" {
			t.Errorf("u.Host = %q, want 'dc46-mum-01.algoip.in:8443'", u.Host)
		}
	})

	t.Run("Edge: Host contains protocol prefix accidentally", func(t *testing.T) {
		cfg := ProxyConfig{
			Host:         "https://dc46-mum-01.algoip.in",
			Port:         443,
			ClientID:     "u",
			ClientSecret: "p",
		}
		u, err := cfg.URL()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Host != "dc46-mum-01.algoip.in:443" {
			t.Errorf("u.Host = %q, want 'dc46-mum-01.algoip.in:443'", u.Host)
		}
		if u.Scheme != "https" {
			t.Errorf("u.Scheme = %q, want 'https'", u.Scheme)
		}
	})

	t.Run("Edge: Host and Scheme have surrounding whitespace", func(t *testing.T) {
		cfg := ProxyConfig{
			Scheme:       "  http  ",
			Host:         "  10.0.0.1  ",
			Port:         8080,
			ClientID:     "u",
			ClientSecret: "p",
		}
		u, err := cfg.URL()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if u.Scheme != "http" || u.Host != "10.0.0.1:8080" {
			t.Errorf("got %s://%s, want http://10.0.0.1:8080", u.Scheme, u.Host)
		}
	})

	t.Run("Edge: Special characters in password (URL encoded)", func(t *testing.T) {
		cfg := ProxyConfig{
			Host:         "proxy.internal",
			Port:         443,
			ClientID:     "user@domain.com",
			ClientSecret: "p@$$:w0rd!#",
		}
		u, err := cfg.URL()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		pass, _ := u.User.Password()
		if pass != "p@$$:w0rd!#" {
			t.Errorf("password = %q, want 'p@$$:w0rd!#'", pass)
		}
		if u.User.Username() != "user@domain.com" {
			t.Errorf("username = %q, want 'user@domain.com'", u.User.Username())
		}
	})

	t.Run("Negative: Empty host returns ErrInvalidInput", func(t *testing.T) {
		cfg := ProxyConfig{
			Host: "   ",
		}
		_, err := cfg.URL()
		if err == nil {
			t.Fatal("expected error on empty host, got nil")
		}
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})

	t.Run("Negative: Unsupported scheme returns ErrInvalidInput", func(t *testing.T) {
		cfg := ProxyConfig{
			Scheme: "socks5",
			Host:   "127.0.0.1",
		}
		_, err := cfg.URL()
		if err == nil {
			t.Fatal("expected error on socks5 scheme, got nil")
		}
		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("expected ErrInvalidInput, got %v", err)
		}
	})
}

// Integration Test: Client with WithProxy routing through mock forward proxy
func TestClient_WithProxy_MockProxy_Success(t *testing.T) {
	var capturedProxyAuth string
	var proxyRequestCount atomic.Int32

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"stat":"Ok","data":[{"nOrdNo":"ORD123","trdSym":"TATASTEEL"}]}`))
	}))
	defer targetServer.Close()

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyRequestCount.Add(1)
		capturedProxyAuth = r.Header.Get("Proxy-Authorization")

		expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("proxyuser:proxypass"))
		if capturedProxyAuth != expectedAuth {
			w.WriteHeader(http.StatusProxyAuthRequired)
			_, _ = w.Write([]byte(`Proxy Authentication Required`))
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), r.Method, targetServer.URL+r.URL.Path, r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		for k, v := range r.Header {
			if !strings.HasPrefix(k, "Proxy-") {
				req.Header[k] = v
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		w.WriteHeader(resp.StatusCode)
		_, _ = w.Write([]byte(`{"stat":"Ok","data":[{"nOrdNo":"ORD123","trdSym":"TATASTEEL"}]}`))
	}))
	defer proxyServer.Close()

	host, portStr, _ := net.SplitHostPort(strings.TrimPrefix(proxyServer.URL, "http://"))
	var port int
	if p, err := strconv.Atoi(portStr); err == nil {
		port = p
	}

	cfg := ProxyConfig{
		Scheme:       "http",
		Host:         host,
		Port:         port,
		ClientID:     "proxyuser",
		ClientSecret: "proxypass",
	}

	client := New("mock_token",
		WithBaseURL(targetServer.URL),
		WithProxy(cfg),
	)

	orders, err := client.GetOrders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error through proxy: %v", err)
	}

	if proxyRequestCount.Load() != 1 {
		t.Errorf("proxyRequestCount = %d, want 1", proxyRequestCount.Load())
	}
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("proxyuser:proxypass"))
	if capturedProxyAuth != expectedAuth {
		t.Errorf("captured Proxy-Authorization = %q, want %q", capturedProxyAuth, expectedAuth)
	}
	if len(orders) != 1 || orders[0].OrderNumber != "ORD123" {
		t.Errorf("orders = %+v, want ORD123", orders)
	}
}

// Integration Test: Client with WithProxyURL
func TestClient_WithProxyURL_MockProxy_Success(t *testing.T) {
	var capturedProxyAuth string
	var proxyHits atomic.Int32

	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"stat":"Ok","data":[{"nOrdNo":"ORD456","trdSym":"INFY"}]}`))
	}))
	defer targetServer.Close()

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits.Add(1)
		capturedProxyAuth = r.Header.Get("Proxy-Authorization")

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"stat":"Ok","data":[{"nOrdNo":"ORD456","trdSym":"INFY"}]}`))
	}))
	defer proxyServer.Close()

	proxyHost := strings.TrimPrefix(proxyServer.URL, "http://")
	rawProxyURL := fmt.Sprintf("http://alice:bobsecret@%s", proxyHost)

	client := New("mock_token",
		WithBaseURL(targetServer.URL),
		WithProxyURL(rawProxyURL),
	)

	orders, err := client.GetOrders(context.Background())
	if err != nil {
		t.Fatalf("WithProxyURL request failed: %v", err)
	}

	if proxyHits.Load() != 1 {
		t.Errorf("proxyHits = %d, want 1", proxyHits.Load())
	}
	expectedAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("alice:bobsecret"))
	if capturedProxyAuth != expectedAuth {
		t.Errorf("captured Proxy-Authorization = %q, want %q", capturedProxyAuth, expectedAuth)
	}
	if len(orders) != 1 || orders[0].OrderNumber != "ORD456" {
		t.Errorf("orders = %+v, want ORD456", orders)
	}
}

// Security & Fail-Fast Test: Malformed/Empty proxy MUST fail fast without sending unproxied traffic
func TestClient_WithProxy_FailFast_NoNetworkLeak(t *testing.T) {
	var targetHits atomic.Int32
	targetServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		targetHits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	defer targetServer.Close()

	testCases := []struct {
		name string
		opt  Option
	}{
		{
			name: "WithProxy empty host",
			opt:  WithProxy(ProxyConfig{Host: "   "}),
		},
		{
			name: "WithProxy invalid scheme",
			opt:  WithProxy(ProxyConfig{Host: "127.0.0.1", Scheme: "ftp"}),
		},
		{
			name: "WithProxyURL empty string",
			opt:  WithProxyURL("   "),
		},
		{
			name: "WithProxyURL malformed string",
			opt:  WithProxyURL("::bad-url::"),
		},
		{
			name: "WithProxyURL unsupported scheme",
			opt:  WithProxyURL("gopher://127.0.0.1:8080"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			targetHits.Store(0)
			client := New("mock_token",
				WithBaseURL(targetServer.URL),
				tc.opt,
			)

			_, err := client.GetOrders(context.Background())
			if err == nil {
				t.Fatal("expected error on misconfigured proxy, got nil")
			}
			if !errors.Is(err, ErrInvalidInput) {
				t.Errorf("expected ErrInvalidInput, got %v", err)
			}
			if targetHits.Load() != 0 {
				t.Errorf("SECURITY LEAK: targetServer received %d requests despite invalid proxy", targetHits.Load())
			}
		})
	}
}

// Negative Test: Proxy authentication failure (HTTP 407)
func TestClient_WithProxy_ProxyAuthFailure407(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusProxyAuthRequired)
		_, _ = w.Write([]byte(`Proxy Authentication Required`))
	}))
	defer proxyServer.Close()

	client := New("mock_token",
		WithBaseURL("http://api.kotakneo.com"),
		WithProxyURL(proxyServer.URL),
	)

	_, err := client.GetOrders(context.Background())
	if err == nil {
		t.Fatal("expected error on 407 proxy auth required, got nil")
	}
}

// Negative Test: Unreachable proxy returns ErrBrokerUnavailable
func TestClient_WithProxy_NetworkFailure(t *testing.T) {
	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	proxyURL := proxyServer.URL
	proxyServer.Close()

	client := New("mock_token",
		WithBaseURL("http://api.kotakneo.com"),
		WithProxyURL(proxyURL),
	)

	_, err := client.GetOrders(context.Background())
	if err == nil {
		t.Fatal("expected network error on unreachable proxy, got nil")
	}
	if !errors.Is(err, ErrBrokerUnavailable) {
		t.Errorf("expected ErrBrokerUnavailable, got %v", err)
	}
}

// Concurrency Test: Multiple goroutines using proxy-configured client simultaneously
func TestClient_WithProxy_Concurrency(t *testing.T) {
	var proxyHits atomic.Int32

	proxyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyHits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"stat":"Ok","data":[{"nOrdNo":"ORD789","trdSym":"SBIN"}]}`))
	}))
	defer proxyServer.Close()

	client := New("mock_token",
		WithBaseURL("http://api.kotakneo.com"),
		WithProxyURL(proxyServer.URL),
	)

	var wg sync.WaitGroup
	ctx := context.Background()
	concurrency := 20
	iterations := 10

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				orders, err := client.GetOrders(ctx)
				if err != nil {
					t.Errorf("concurrent request failed: %v", err)
					return
				}
				if len(orders) != 1 || orders[0].OrderNumber != "ORD789" {
					t.Errorf("unexpected orders: %+v", orders)
					return
				}
			}
		}()
	}

	wg.Wait()

	expectedTotal := int32(concurrency * iterations)
	if proxyHits.Load() != expectedTotal {
		t.Errorf("proxyHits = %d, want %d", proxyHits.Load(), expectedTotal)
	}
}

// Option preservation test: custom timeout preserved when WithProxy used
func TestClient_WithProxy_TransportPreservation(t *testing.T) {
	customTimeout := 42 * time.Second
	customHTTP := &http.Client{
		Timeout: customTimeout,
	}

	client := New("mock_token",
		WithHTTPClient(customHTTP),
		WithProxy(ProxyConfig{
			Host: "dc46-mum-01.algoip.in",
			Port: 443,
		}),
	)

	hc, ok := client.transport.client.(*http.Client)
	if !ok {
		t.Fatalf("expected *http.Client, got %T", client.transport.client)
	}
	if hc.Timeout != customTimeout {
		t.Errorf("hc.Timeout = %v, want %v", hc.Timeout, customTimeout)
	}
	tr, ok := hc.Transport.(*http.Transport)
	if !ok || tr.Proxy == nil {
		t.Fatal("expected *http.Transport with non-nil Proxy")
	}
}

// TestRESTClientFor tests standalone client factory
func TestRESTClientFor(t *testing.T) {
	cfg := ProxyConfig{
		Host:         "dc46-mum-01.algoip.in",
		Port:         443,
		ClientID:     "u1",
		ClientSecret: "p1",
	}
	client, err := RESTClientFor(cfg)
	if err != nil {
		t.Fatalf("unexpected error from RESTClientFor: %v", err)
	}
	if client == nil || client.Transport == nil {
		t.Fatal("expected non-nil *http.Client with non-nil Transport")
	}
	tr, ok := client.Transport.(*http.Transport)
	if !ok || tr.Proxy == nil {
		t.Fatal("expected *http.Transport with non-nil Proxy")
	}

	badCfg := ProxyConfig{Host: "   "}
	_, err = RESTClientFor(badCfg)
	if err == nil {
		t.Fatal("expected error on empty host, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}
