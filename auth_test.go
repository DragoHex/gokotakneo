package gokotakneo_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DragoHex/gokotakneo"
)

func TestAuth_Lifecycle(t *testing.T) {
	t.Run("Positive: Complete 2-Step Login and Gateway Discovery", func(t *testing.T) {
		var step1Received, step2Received bool

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")

			switch r.URL.Path {
			case "/login/1.0/tradeApiLogin":
				step1Received = true
				if r.Header.Get("Authorization") != "portal-token-123" {
					t.Errorf("Step 1 Authorization = %q, want 'portal-token-123'", r.Header.Get("Authorization"))
				}
				if r.Header.Get("neo-fin-key") != "neotradeapi" {
					t.Errorf("Step 1 neo-fin-key = %q, want 'neotradeapi'", r.Header.Get("neo-fin-key"))
				}

				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				json.Unmarshal(body, &req)
				if req["mobileNumber"] != "+919876543210" || req["ucc"] != "CLIENT1" || req["totp"] != "123456" {
					t.Errorf("Step 1 unexpected payload: %v", req)
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"data": {
						"token": "view-jwt-token",
						"sid": "view-sid-xyz",
						"rid": "req-1",
						"kType": "View",
						"status": "success",
						"greetingName": "Trader Joe"
					}
				}`))

			case "/login/1.0/tradeApiValidate":
				step2Received = true
				if r.Header.Get("Auth") != "view-jwt-token" {
					t.Errorf("Step 2 Auth = %q, want 'view-jwt-token'", r.Header.Get("Auth"))
				}
				if r.Header.Get("sid") != "view-sid-xyz" {
					t.Errorf("Step 2 sid = %q, want 'view-sid-xyz'", r.Header.Get("sid"))
				}

				body, _ := io.ReadAll(r.Body)
				var req map[string]string
				json.Unmarshal(body, &req)
				if req["mpin"] != "654321" {
					t.Errorf("Step 2 unexpected mpin: %q", req["mpin"])
				}

				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"data": {
						"token": "trade-jwt-token-999",
						"sid": "trade-sid-abc",
						"baseUrl": "https://cis.kotaksecurities.com",
						"feedUrl": "https://feed.kotaksecurities.com/apifeed",
						"rtUrl": "https://rt.kotaksecurities.com/realtime",
						"kType": "Trade",
						"status": "success"
					}
				}`))

			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token-123", gokotakneo.WithBaseURL(server.URL))

		if client.IsAuthenticated() {
			t.Errorf("client should not be authenticated initially")
		}

		// Step 1: Login with TOTP
		viewSession, err := client.LoginTOTP(context.Background(), "+919876543210", "CLIENT1", "123456")
		if err != nil {
			t.Fatalf("LoginTOTP failed: %v", err)
		}
		if viewSession.Token != "view-jwt-token" || viewSession.Sid != "view-sid-xyz" {
			t.Errorf("unexpected view session: %+v", viewSession)
		}

		// Step 2: Validate MPIN
		tradeSession, err := client.ValidateMPIN(context.Background(), viewSession.Token, viewSession.Sid, "654321")
		if err != nil {
			t.Fatalf("ValidateMPIN failed: %v", err)
		}

		if !step1Received || !step2Received {
			t.Errorf("expected both steps to execute: step1=%v, step2=%v", step1Received, step2Received)
		}

		if tradeSession.Token != "trade-jwt-token-999" {
			t.Errorf("trade token = %q, want 'trade-jwt-token-999'", tradeSession.Token)
		}

		if !client.IsAuthenticated() {
			t.Errorf("client should be authenticated after successful Step 2")
		}

		// Dynamic Gateways Updated
		if client.BaseURL() != "https://cis.kotaksecurities.com" {
			t.Errorf("client BaseURL = %q, want 'https://cis.kotaksecurities.com'", client.BaseURL())
		}
		if client.FeedURL() != "https://feed.kotaksecurities.com/apifeed" {
			t.Errorf("client FeedURL = %q, want 'https://feed.kotaksecurities.com/apifeed'", client.FeedURL())
		}
		if client.RTURL() != "https://rt.kotaksecurities.com/realtime" {
			t.Errorf("client RTURL = %q, want 'https://rt.kotaksecurities.com/realtime'", client.RTURL())
		}
	})

	t.Run("Negative: Step 1 Invalid TOTP (401)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "Invalid credentials or TOTP.", "errorCode": "401"}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		_, err := client.LoginTOTP(context.Background(), "+919876543210", "CLIENT1", "000000")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, gokotakneo.ErrAuthExpired) {
			t.Errorf("expected ErrAuthExpired, got %v", err)
		}
		if client.IsAuthenticated() {
			t.Errorf("client should not be authenticated")
		}
	})

	t.Run("Negative: Step 2 Invalid MPIN (401)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status": "error", "message": "Invalid MPIN.", "errorCode": "401"}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		_, err := client.ValidateMPIN(context.Background(), "view-tok", "view-sid", "000000")
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !errors.Is(err, gokotakneo.ErrAuthExpired) {
			t.Errorf("expected ErrAuthExpired, got %v", err)
		}
		if client.IsAuthenticated() {
			t.Errorf("client should not be authenticated")
		}
	})

	t.Run("Edge Case: Auto-trim whitespace in inputs", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			var req map[string]string
			json.Unmarshal(body, &req)

			if req["mobileNumber"] != "+919876543210" || req["ucc"] != "CLIENT1" || req["totp"] != "123456" {
				t.Errorf("whitespace was not trimmed: %v", req)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"token": "t", "sid": "s"}}`))
		}))
		defer server.Close()

		client := gokotakneo.New("portal-token", gokotakneo.WithBaseURL(server.URL))
		_, err := client.LoginTOTP(context.Background(), "  +919876543210  ", " CLIENT1 ", " 123456 ")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("Positive: SetSessionTokens Updates Authenticated State", func(t *testing.T) {
		client := gokotakneo.New("portal-token")
		if client.IsAuthenticated() {
			t.Errorf("new client should not be authenticated")
		}

		client.SetSessionTokens("valid-trade-token", "valid-trade-sid")
		if !client.IsAuthenticated() {
			t.Errorf("client with non-empty session tokens must be authenticated")
		}

		client.SetSessionTokens("", "")
		if client.IsAuthenticated() {
			t.Errorf("client with empty session tokens must not be authenticated")
		}
	})
}
