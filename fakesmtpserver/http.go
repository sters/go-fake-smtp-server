package fakesmtpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrMissingEmailParam is returned when the email parameter is missing from the request.
	ErrMissingEmailParam = errors.New("missing required parameter: email")
	// ErrInvalidEmailFormat is returned when the email parameter has an invalid format.
	ErrInvalidEmailFormat = errors.New("invalid email format")
)

const ViewAddr = "127.0.0.1:11080"

func StartViewServer() error {
	mux := http.NewServeMux()

	// Existing endpoint - get all emails
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err := enc.Encode(sharedBackend.GetAllData())
		if err != nil {
			slog.Info("err", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = w.Write(buf.Bytes())
		w.WriteHeader(http.StatusOK)
	})

	// Search endpoints
	mux.HandleFunc("/search/to", handleSearchEndpoint("to"))
	mux.HandleFunc("/search/cc", handleSearchEndpoint("cc"))
	mux.HandleFunc("/search/bcc", handleSearchEndpoint("bcc"))
	mux.HandleFunc("/search/from", handleSearchEndpoint("from"))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	l, err := ln()
	if err != nil {
		return err
	}

	slog.Info("Starting HTTP server", "addr", l.Addr().String())
	if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve error: %w", err)
	}

	return nil
}

func ln() (net.Listener, error) {
	ln, err := net.Listen("tcp", ViewAddr)
	if err != nil {
		return nil, fmt.Errorf("listen error: %w", err)
	}

	return ln, nil
}

// handleSearchEndpoint returns a handler function for the specified search field.
func handleSearchEndpoint(field string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only allow GET requests
		if r.Method != http.MethodGet {
			writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")

			return
		}

		// Parse and validate email parameter
		email, err := parseEmailParameter(r.URL.Query())
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, err.Error())

			return
		}

		// Perform search
		results, err := sharedBackend.SearchByField(field, email)
		if err != nil {
			slog.Info("search error", "field", field, "email", email, "error", err)
			writeJSONError(w, http.StatusInternalServerError, "search failed")

			return
		}

		// Return results as JSON
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		if err := enc.Encode(results); err != nil {
			slog.Info("encoding error", "error", err)
			writeJSONError(w, http.StatusInternalServerError, "encoding failed")

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(buf.Bytes())
	}
}

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
