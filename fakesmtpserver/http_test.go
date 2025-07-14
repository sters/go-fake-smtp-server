package fakesmtpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestHandleSearchEndpoint(t *testing.T) {
	// Setup test backend with sample data
	originalBackend := sharedBackend
	defer func() { sharedBackend = originalBackend }()

	testBackend := &smtpBackend{}
	setupTestData(testBackend)
	sharedBackend = testBackend

	tests := []struct {
		name           string
		field          string
		email          string
		method         string
		expectedStatus int
		expectedCount  int
		description    string
	}{
		{
			name:           "valid_to_search",
			field:          "to",
			email:          "recipient1@example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			description:    "Should find email in To field",
		},
		{
			name:           "valid_from_search",
			field:          "from",
			email:          "sender2@example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			description:    "Should find email in From field",
		},
		{
			name:           "valid_cc_search",
			field:          "cc",
			email:          "cc@example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			description:    "Should find email in CC field",
		},
		{
			name:           "no_results",
			field:          "to",
			email:          "notfound@example.com",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedCount:  0,
			description:    "Should return empty results for non-existent email",
		},
		{
			name:           "case_insensitive",
			field:          "from",
			email:          "SENDER1@EXAMPLE.COM",
			method:         "GET",
			expectedStatus: http.StatusOK,
			expectedCount:  1,
			description:    "Should be case insensitive",
		},
		{
			name:           "invalid_method",
			field:          "to",
			email:          "test@example.com",
			method:         "POST",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedCount:  0,
			description:    "Should reject non-GET methods",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request
			reqURL := fmt.Sprintf("/search/%s?email=%s", tt.field, url.QueryEscape(tt.email))
			req := httptest.NewRequest(tt.method, reqURL, nil)
			w := httptest.NewRecorder()

			// Call handler
			handler := handleSearchEndpoint(tt.field)
			handler(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("handleSearchEndpoint() status = %d, want %d. %s", w.Code, tt.expectedStatus, tt.description)
			}

			// Check content type for successful requests
			if tt.expectedStatus == http.StatusOK {
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("handleSearchEndpoint() Content-Type = %s, want application/json", contentType)
				}

				// Parse response and check result count
				var results []smtpView
				if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if len(results) != tt.expectedCount {
					t.Errorf("handleSearchEndpoint() returned %d results, want %d. %s", len(results), tt.expectedCount, tt.description)
				}
			}

			// Check error responses have proper format
			if tt.expectedStatus >= 400 {
				var errorResp map[string]string
				if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
					t.Fatalf("Failed to unmarshal error response: %v", err)
				}

				if _, exists := errorResp["error"]; !exists {
					t.Errorf("Error response missing 'error' field")
				}
			}
		})
	}
}

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

func TestSearchEndpointMissingEmailParam(t *testing.T) {
	// Test all search endpoints with missing email parameter
	endpoints := []string{"to", "cc", "bcc", "from"}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("search_%s_missing_email", endpoint), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/search/"+endpoint, nil)
			w := httptest.NewRecorder()

			handler := handleSearchEndpoint(endpoint)
			handler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d for missing email param, got %d", http.StatusBadRequest, w.Code)
			}

			var errorResp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v", err)
			}

			if !strings.Contains(errorResp["error"], "missing required parameter") {
				t.Errorf("Expected error message about missing parameter, got: %s", errorResp["error"])
			}
		})
	}
}

func TestSearchEndpointInvalidEmailParam(t *testing.T) {
	// Test all search endpoints with invalid email parameter
	endpoints := []string{"to", "cc", "bcc", "from"}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("search_%s_invalid_email", endpoint), func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/search/%s?email=invalid-email", endpoint), nil)
			w := httptest.NewRecorder()

			handler := handleSearchEndpoint(endpoint)
			handler(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("Expected status %d for invalid email param, got %d", http.StatusBadRequest, w.Code)
			}

			var errorResp map[string]string
			if err := json.Unmarshal(w.Body.Bytes(), &errorResp); err != nil {
				t.Fatalf("Failed to unmarshal error response: %v", err)
			}

			if !strings.Contains(errorResp["error"], "invalid email format") {
				t.Errorf("Expected error message about invalid email format, got: %s", errorResp["error"])
			}
		})
	}
}

func TestSearchEndpointDualSourceLogic(t *testing.T) {
	// Test that search finds emails in both headers and SMTP transaction data
	originalBackend := sharedBackend
	defer func() { sharedBackend = originalBackend }()

	testBackend := &smtpBackend{}

	// Create a session where SMTP RCPT TO has an email not in headers (BCC scenario)
	session := &smtpSession{
		data:         createTestEmailData("sender@example.com", "recipient@example.com", "Test Subject"),
		receivedTime: time.Now(),
		mailFrom:     "smtp-sender@example.com",                                 // Different from header
		rcptTo:       []string{"recipient@example.com", "bcc-only@example.com"}, // BCC not in headers
		clientAddr:   "192.168.1.100:12345",
		clientHost:   "client.example.com",
		tlsUsed:      true,
	}

	testBackend.sessions = []*smtpSession{session}
	sharedBackend = testBackend

	tests := []struct {
		name        string
		field       string
		email       string
		expectFound bool
		description string
	}{
		{
			name:        "find_smtp_from_not_in_header",
			field:       "from",
			email:       "smtp-sender@example.com",
			expectFound: true,
			description: "Should find SMTP MAIL FROM even if different from header",
		},
		{
			name:        "find_bcc_in_smtp_rcpt",
			field:       "to",
			email:       "bcc-only@example.com",
			expectFound: true,
			description: "Should find BCC recipient in SMTP RCPT TO",
		},
		{
			name:        "find_header_from",
			field:       "from",
			email:       "sender@example.com",
			expectFound: true,
			description: "Should find sender from email header",
		},
		{
			name:        "find_header_to",
			field:       "to",
			email:       "recipient@example.com",
			expectFound: true,
			description: "Should find recipient from both header and SMTP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqURL := fmt.Sprintf("/search/%s?email=%s", tt.field, url.QueryEscape(tt.email))
			req := httptest.NewRequest(http.MethodGet, reqURL, nil)
			w := httptest.NewRecorder()

			handler := handleSearchEndpoint(tt.field)
			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var results []smtpView
			if err := json.Unmarshal(w.Body.Bytes(), &results); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			found := len(results) > 0
			if found != tt.expectFound {
				t.Errorf("%s: found=%v, expectFound=%v. %s", tt.name, found, tt.expectFound, tt.description)
			}
		})
	}
}

// Helper function to setup test data.
func setupTestData(backend *smtpBackend) {
	session1 := &smtpSession{
		data:         createTestEmailData("sender1@example.com", "recipient1@example.com", "Test Subject 1"),
		receivedTime: time.Now(),
		mailFrom:     "sender1@example.com",
		rcptTo:       []string{"recipient1@example.com", "recipient2@example.com"},
		clientAddr:   "192.168.1.100:12345",
		clientHost:   "client1.example.com",
		tlsUsed:      true,
	}

	session2 := &smtpSession{
		data:         createTestEmailData("sender2@example.com", "recipient3@example.com", "Test Subject 2"),
		receivedTime: time.Now(),
		mailFrom:     "sender2@example.com",
		rcptTo:       []string{"recipient3@example.com"},
		clientAddr:   "192.168.1.101:12346",
		clientHost:   "client2.example.com",
		tlsUsed:      false,
	}

	session3 := &smtpSession{
		data:         createTestEmailWithCC("sender3@example.com", "recipient4@example.com", "cc@example.com", "Test Subject 3"),
		receivedTime: time.Now(),
		mailFrom:     "sender3@example.com",
		rcptTo:       []string{"recipient4@example.com", "cc@example.com", "bcc@example.com"},
		clientAddr:   "192.168.1.102:12347",
		clientHost:   "client3.example.com",
		tlsUsed:      true,
	}

	backend.sessions = []*smtpSession{session1, session2, session3}
}
