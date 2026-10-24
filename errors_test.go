package gokotakneo_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/DragoHex/gokotakneo"
)

func TestParseError_Taxonomy(t *testing.T) {
	tests := []struct {
		name          string
		httpStatus    int
		body          []byte
		wantSentinel  error
		wantCode      int
		wantMsgSubstr string
	}{
		{
			name:          "Positive: Kotak RMS Insufficient Balance (1004)",
			httpStatus:    http.StatusOK,
			body:          []byte(`{"stat": "Not_Ok", "emsg": "Insufficient balance.", "stCode": 1004}`),
			wantSentinel:  gokotakneo.ErrInsufficientFunds,
			wantCode:      1004,
			wantMsgSubstr: "Insufficient balance",
		},
		{
			name:          "Positive: Kotak Order Not Modifiable (1006)",
			httpStatus:    http.StatusOK,
			body:          []byte(`{"stat": "Not_Ok", "emsg": "Order cannot be modified as it is already executed.", "stCode": 1006}`),
			wantSentinel:  gokotakneo.ErrOrderNotModifiable,
			wantCode:      1006,
			wantMsgSubstr: "already executed",
		},
		{
			name:          "Positive: Auth 401 Error Envelope",
			httpStatus:    http.StatusUnauthorized,
			body:          []byte(`{"status": "error", "message": "Invalid MPIN.", "errorCode": "401"}`),
			wantSentinel:  gokotakneo.ErrAuthExpired,
			wantCode:      401,
			wantMsgSubstr: "Invalid MPIN",
		},
		{
			name:          "Negative: HTML Gateway 502 Bad Gateway",
			httpStatus:    http.StatusBadGateway,
			body:          []byte(`<html><body>502 Bad Gateway</body></html>`),
			wantSentinel:  gokotakneo.ErrBrokerUnavailable,
			wantCode:      502,
			wantMsgSubstr: "Bad Gateway",
		},
		{
			name:          "Edge Case: Kotak Error JSON with empty emsg",
			httpStatus:    http.StatusBadRequest,
			body:          []byte(`{"stat": "Not_Ok", "stCode": 1004, "emsg": ""}`),
			wantSentinel:  gokotakneo.ErrInsufficientFunds,
			wantCode:      1004,
			wantMsgSubstr: "Insufficient balance",
		},
		{
			name:          "Edge Case: Truncated JSON",
			httpStatus:    http.StatusInternalServerError,
			body:          []byte(`{"stat": "Not_Ok", "stCode":`),
			wantSentinel:  gokotakneo.ErrBrokerUnavailable,
			wantCode:      500,
			wantMsgSubstr: "failed to parse",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := gokotakneo.ParseAPIError(tt.httpStatus, tt.body)
			if err == nil {
				t.Fatalf("expected error, got nil")
			}

			if !errors.Is(err, tt.wantSentinel) {
				t.Errorf("errors.Is(err, %v) = false, got err = %v", tt.wantSentinel, err)
			}

			var apiErr *gokotakneo.APIError
			if errors.As(err, &apiErr) {
				if apiErr.StatusCode != tt.wantCode {
					t.Errorf("StatusCode = %d, want %d", apiErr.StatusCode, tt.wantCode)
				}
			} else {
				t.Errorf("expected *APIError, got %T", err)
			}
		})
	}
}
