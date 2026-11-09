package wsproxy

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// NewDialer returns a *websocket.Dialer configured to route traffic through proxyURL.
// Supports nil (direct), http:// (standard ProxyURL), and https:// (TLS CONNECT tunnel).
func NewDialer(proxyURL *url.URL, handshakeTimeout time.Duration, customTLS *tls.Config) *websocket.Dialer {
	if handshakeTimeout <= 0 {
		handshakeTimeout = 10 * time.Second
	}

	dialer := &websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
		TLSClientConfig:  customTLS,
	}

	if proxyURL == nil {
		return dialer
	}

	if strings.EqualFold(proxyURL.Scheme, "https") {
		dialer.NetDialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialHTTPSProxy(ctx, proxyURL, addr, customTLS)
		}
		return dialer
	}

	dialer.Proxy = http.ProxyURL(proxyURL)
	return dialer
}

func dialHTTPSProxy(ctx context.Context, proxyURL *url.URL, targetAddr string, customTLS *tls.Config) (net.Conn, error) {
	proxyHost := proxyURL.Host
	hostname := proxyURL.Hostname()

	tlsConfig := &tls.Config{
		ServerName: hostname,
	}
	if customTLS != nil {
		tlsConfig = customTLS.Clone()
		if tlsConfig.ServerName == "" {
			tlsConfig.ServerName = hostname
		}
	}

	tlsDialer := &tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout: 10 * time.Second,
		},
		Config: tlsConfig,
	}

	conn, err := tlsDialer.DialContext(ctx, "tcp", proxyHost)
	if err != nil {
		return nil, fmt.Errorf("tls dial proxy %s: %w", proxyHost, err)
	}

	// Prepare HTTP CONNECT request
	connectReq := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", targetAddr, targetAddr)
	if proxyURL.User != nil {
		auth := base64.StdEncoding.EncodeToString([]byte(proxyURL.User.String()))
		connectReq += fmt.Sprintf("Proxy-Authorization: Basic %s\r\n", auth)
	}
	connectReq += "\r\n"

	if _, err := conn.Write([]byte(connectReq)); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("write CONNECT to proxy: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, &http.Request{Method: http.MethodConnect})
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("read proxy CONNECT response: %w", err)
	}
	_ = resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, fmt.Errorf("proxy CONNECT rejected: %s", resp.Status)
	}

	if br.Buffered() > 0 {
		return &bufferedConn{Conn: conn, br: br}, nil
	}
	return conn, nil
}

type bufferedConn struct {
	net.Conn
	br *bufio.Reader
}

func (b *bufferedConn) Read(p []byte) (int, error) {
	return b.br.Read(p)
}
