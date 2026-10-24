package gokotakneo

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// Domain sentinel errors.
var (
	ErrAuthExpired        = errors.New("authentication expired or invalid credentials")
	ErrInsufficientFunds  = errors.New("insufficient balance")
	ErrOrderNotModifiable = errors.New("order cannot be modified as it is already executed or cancelled")
	ErrNotFound           = errors.New("resource or order not found")
	ErrInvalidInput       = errors.New("invalid request parameters")
	ErrBrokerUnavailable  = errors.New("kotak-neo gateway unreachable or timed out")
)

// APIError captures detailed error responses from Kotak-Neo APIs.
type APIError struct {
	StatusCode int    `json:"stCode"`
	Status     string `json:"stat"`
	Message    string `json:"emsg"`
	Err        error  `json:"-"`
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("kotak-neo api error [%d]: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("kotak-neo api error [%d]: %v", e.StatusCode, e.Err)
}

func (e *APIError) Unwrap() error {
	return e.Err
}

// ParseAPIError parses HTTP status and body bytes into a strongly typed APIError.
func ParseAPIError(httpStatus int, body []byte) error {
	// 1. Try parsing Kotak standard trading error: {"stat": "Not_Ok", "stCode": 1004, "emsg": "..."}
	var kotakErr struct {
		Stat   string `json:"stat"`
		StCode int    `json:"stCode"`
		EMsg   string `json:"emsg"`
	}
	if err := json.Unmarshal(body, &kotakErr); err == nil && (kotakErr.Stat == "Not_Ok" || kotakErr.StCode > 0) {
		apiErr := &APIError{
			StatusCode: kotakErr.StCode,
			Status:     kotakErr.Stat,
			Message:    kotakErr.EMsg,
		}
		apiErr.Err = mapKotakCodeToSentinel(kotakErr.StCode, kotakErr.EMsg)
		if apiErr.Message == "" {
			apiErr.Message = apiErr.Err.Error()
		}
		return apiErr
	}

	// 2. Try parsing Auth error: {"status": "error", "message": "...", "errorCode": "401"}
	var authErr struct {
		Status    string `json:"status"`
		Message   string `json:"message"`
		ErrorCode string `json:"errorCode"`
	}
	if err := json.Unmarshal(body, &authErr); err == nil && authErr.Status == "error" {
		code := httpStatus
		if authErr.ErrorCode == "401" {
			code = 401
		}
		return &APIError{
			StatusCode: code,
			Status:     authErr.Status,
			Message:    authErr.Message,
			Err:        ErrAuthExpired,
		}
	}

	// 3. Fallback for non-JSON or HTML gateway errors (e.g. 502/504)
	var sentinel error
	switch httpStatus {
	case http.StatusUnauthorized, http.StatusForbidden:
		sentinel = ErrAuthExpired
	case http.StatusNotFound:
		sentinel = ErrNotFound
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		sentinel = ErrInvalidInput
	default:
		sentinel = ErrBrokerUnavailable
	}

	rawSnippet := strings.TrimSpace(string(body))
	if len(rawSnippet) > 100 {
		rawSnippet = rawSnippet[:100] + "..."
	}

	msg := fmt.Sprintf("HTTP %d: %s", httpStatus, rawSnippet)
	if strings.HasPrefix(rawSnippet, "<") || !json.Valid(body) {
		msg = fmt.Sprintf("failed to parse gateway response (HTTP %d): %s", httpStatus, rawSnippet)
	}

	return &APIError{
		StatusCode: httpStatus,
		Message:    msg,
		Err:        sentinel,
	}
}

func mapKotakCodeToSentinel(stCode int, msg string) error {
	switch stCode {
	case 1004:
		return ErrInsufficientFunds
	case 1006:
		return ErrOrderNotModifiable
	case 401:
		return ErrAuthExpired
	case 404:
		return ErrNotFound
	default:
		if strings.Contains(strings.ToLower(msg), "insufficient") {
			return ErrInsufficientFunds
		}
		if strings.Contains(strings.ToLower(msg), "already executed") {
			return ErrOrderNotModifiable
		}
		return ErrInvalidInput
	}
}
