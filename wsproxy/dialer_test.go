package wsproxy_test

import (
	"crypto/tls"
	"net/url"
	"testing"
	"time"

	"github.com/DragoHex/gokotakneo/wsproxy"
)

func TestNewDialer(t *testing.T) {
	t.Run("nil proxyURL returns default dialer", func(t *testing.T) {
		d := wsproxy.NewDialer(nil, 5*time.Second, nil)
		if d == nil {
			t.Fatal("expected non-nil dialer")
		}
		if d.Proxy != nil {
			t.Error("expected nil Proxy for direct connection")
		}
		if d.NetDialContext != nil {
			t.Error("expected nil NetDialContext for direct connection")
		}
	})

	t.Run("http proxy sets Proxy function", func(t *testing.T) {
		u, _ := url.Parse("http://127.0.0.1:8080")
		d := wsproxy.NewDialer(u, 5*time.Second, nil)
		if d == nil {
			t.Fatal("expected non-nil dialer")
		}
		if d.Proxy == nil {
			t.Error("expected non-nil Proxy for http proxy")
		}
		if d.NetDialContext != nil {
			t.Error("expected nil NetDialContext for http proxy")
		}
	})

	t.Run("https proxy sets NetDialContext", func(t *testing.T) {
		u, _ := url.Parse("https://user:pass@127.0.0.1:443")
		tlsCfg := &tls.Config{InsecureSkipVerify: true}
		d := wsproxy.NewDialer(u, 5*time.Second, tlsCfg)
		if d == nil {
			t.Fatal("expected non-nil dialer")
		}
		if d.NetDialContext == nil {
			t.Error("expected non-nil NetDialContext for https proxy")
		}
	})
}
