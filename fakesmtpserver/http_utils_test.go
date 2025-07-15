package fakesmtpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestParseEmailParameter(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		want        string
		wantErr     bool
		expectedErr error
	}{
		{
			name:    "valid_email",
			query:   "email=test@example.com",
			want:    "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid_email_with_spaces",
			query:   "email= test@example.com ",
			want:    "test@example.com",
			wantErr: false,
		},
		{
			name:        "missing_email_param",
			query:       "other=value",
			want:        "",
			wantErr:     true,
			expectedErr: ErrMissingEmailParam,
		},
		{
			name:        "empty_email_param",
			query:       "email=",
			want:        "",
			wantErr:     true,
			expectedErr: ErrMissingEmailParam,
		},
		{
			name:        "invalid_email_format",
			query:       "email=invalid-email",
			want:        "",
			wantErr:     true,
			expectedErr: ErrInvalidEmailFormat,
		},
		{
			name:    "url_encoded_email",
			query:   "email=test%40example.com",
			want:    "test@example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse query string
			values, err := url.ParseQuery(tt.query)
			if err != nil {
				t.Fatalf("Failed to parse query: %v", err)
			}

			got, err := parseEmailParameter(values)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseEmailParameter() error = %v, wantErr %v", err, tt.wantErr)

				return
			}

			if tt.expectedErr != nil && !errors.Is(err, tt.expectedErr) {
				t.Errorf("parseEmailParameter() error = %v, want %v", err, tt.expectedErr)
			}

			if got != tt.want {
				t.Errorf("parseEmailParameter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteJSONError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		message    string
	}{
		{
			name:       "bad_request",
			statusCode: http.StatusBadRequest,
			message:    "invalid request",
		},
		{
			name:       "not_found",
			statusCode: http.StatusNotFound,
			message:    "not found",
		},
		{
			name:       "internal_error",
			statusCode: http.StatusInternalServerError,
			message:    "internal server error",
		},
		{
			name:       "method_not_allowed",
			statusCode: http.StatusMethodNotAllowed,
			message:    "method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeJSONError(w, tt.statusCode, tt.message)

			// Check status code
			if w.Code != tt.statusCode {
				t.Errorf("writeJSONError() status = %d, want %d", w.Code, tt.statusCode)
			}

			// Check content type
			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("writeJSONError() Content-Type = %s, want application/json", contentType)
			}

			// Check response body
			var errorResp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v", err)
			}

			if errorResp["error"] != tt.message {
				t.Errorf("writeJSONError() error message = %s, want %s", errorResp["error"], tt.message)
			}
		})
	}
}