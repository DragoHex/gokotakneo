package gokotakneo_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/DragoHex/gokotakneo"
)

func TestTransport_ExecuteJData(t *testing.T) {
	t.Run("Positive: Correct Form Encoding and Header Injection", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check Headers
			if r.Header.Get("Auth") != "test-token" {
				t.Errorf("Header Auth = %q, want 'test-token'", r.Header.Get("Auth"))
			}
			if r.Header.Get("Sid") != "test-sid" {
				t.Errorf("Header Sid = %q, want 'test-sid'", r.Header.Get("Sid"))
			}
			if r.Header.Get("neo-fin-key") != "neotradeapi" {
				t.Errorf("Header neo-fin-key = %q, want 'neotradeapi'", r.Header.Get("neo-fin-key"))
			}
			if !strings.Contains(r.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
				t.Errorf("Content-Type = %q, want form-urlencoded", r.Header.Get("Content-Type"))
			}

			// Check Body
			body, _ := io.ReadAll(r.Body)
			values, err := url.ParseQuery(string(body))
			if err != nil {
				t.Fatalf("failed to parse form body: %v", err)
			}
			jData := values.Get("jData")
			if !strings.Contains(jData, `"symbol":"INFY-EQ"`) {
				t.Errorf("jData = %q, expected to contain symbol INFY-EQ", jData)
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"stat": "Ok", "stCode": 200, "nOrdNo": "250720000007242"}`))
		}))
		defer server.Close()

		tr := gokotakneo.NewDefaultTransport(server.URL)
		tr.SetSessionHeaders("test-token", "test-sid")

		var out struct {
			Stat   string `json:"stat"`
			StCode int    `json:"stCode"`
			NOrdNo string `json:"nOrdNo"`
		}

		payload := map[string]string{"symbol": "INFY-EQ", "tag": "my & special <tag>"}
		err := tr.ExecuteJData(context.Background(), http.MethodPost, "/quick/order/rule/ms/place", payload, &out)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if out.NOrdNo != "250720000007242" {
			t.Errorf("NOrdNo = %q, want '250720000007242'", out.NOrdNo)
		}
	})

	t.Run("Negative: Server Returns Kotak Error in HTTP 200", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"stat": "Not_Ok", "stCode": 1004, "emsg": "Insufficient balance."}`))
		}))
		defer server.Close()

		tr := gokotakneo.NewDefaultTransport(server.URL)
		err := tr.ExecuteJData(context.Background(), http.MethodPost, "/place", map[string]string{}, nil)
		if err == nil {
			t.Fatalf("expected error from stat: Not_Ok, got nil")
		}

		if !strings.Contains(err.Error(), "Insufficient balance") {
			t.Errorf("error = %v, want to contain 'Insufficient balance'", err)
		}
	})
}
