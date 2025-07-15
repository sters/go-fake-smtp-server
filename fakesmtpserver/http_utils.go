package fakesmtpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

var (
	// ErrMissingEmailParam is returned when the email parameter is missing from the request.
	ErrMissingEmailParam = errors.New("missing required parameter: email")
	// ErrInvalidEmailFormat is returned when the email parameter has an invalid format.
	ErrInvalidEmailFormat = errors.New("invalid email format")
)

// parseEmailParameter extracts and validates the email parameter from query string.
func parseEmailParameter(values url.Values) (string, error) {
	email := values.Get("email")
	if email == "" {
		return "", ErrMissingEmailParam
	}

	// Basic email validation
	email = strings.TrimSpace(email)
	if !strings.Contains(email, "@") {
		return "", ErrInvalidEmailFormat
	}

	return email, nil
}

// writeJSONError writes an error response in JSON format.
func writeJSONError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResp := map[string]string{
		"error": message,
	}

	if err := json.NewEncoder(w).Encode(errorResp); err != nil {
		slog.Info("failed to encode error response", "error", err)
	}
}
