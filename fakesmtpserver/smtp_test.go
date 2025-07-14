package fakesmtpserver

import (
	"net/mail"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-smtp"
)

func TestSearchByField(t *testing.T) {
	// Create a test backend with sample data
	backend := &smtpBackend{}

	// Create test sessions with different email scenarios
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

	// Test session with CC and BCC
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

	tests := []struct {
		name          string
		field         string
		email         string
		expectedCount int
		description   string
	}{
		{
			name:          "search_to_field_found",
			field:         "to",
			email:         "recipient1@example.com",
			expectedCount: 1,
			description:   "Should find email in To field",
		},
		{
			name:          "search_to_field_not_found",
			field:         "to",
			email:         "notfound@example.com",
			expectedCount: 0,
			description:   "Should not find non-existent email in To field",
		},
		{
			name:          "search_from_field_found",
			field:         "from",
			email:         "sender2@example.com",
			expectedCount: 1,
			description:   "Should find email in From field",
		},
		{
			name:          "search_cc_field_found",
			field:         "cc",
			email:         "cc@example.com",
			expectedCount: 1,
			description:   "Should find email in CC field",
		},
		{
			name:          "search_bcc_smtp_only",
			field:         "to",
			email:         "bcc@example.com",
			expectedCount: 1,
			description:   "Should find BCC email in SMTP RCPT TO (not in headers)",
		},
		{
			name:          "case_insensitive_search",
			field:         "from",
			email:         "SENDER1@EXAMPLE.COM",
			expectedCount: 1,
			description:   "Search should be case insensitive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := backend.SearchByField(tt.field, tt.email)
			if err != nil {
				t.Fatalf("SearchByField() error = %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("SearchByField() = %d results, want %d. %s", len(results), tt.expectedCount, tt.description)
			}

			// Verify that results contain the searched email
			if tt.expectedCount > 0 && len(results) > 0 {
				found := false
				for _, result := range results {
					if tt.field == "to" && (containsEmailInAddresses(result.ToAddressList, strings.ToLower(tt.email)) ||
						containsEmailInStrings(result.SMTPTo, strings.ToLower(tt.email))) {
						found = true

						break
					}
					if tt.field == "from" && (containsEmailInAddresses(result.FromAddressList, strings.ToLower(tt.email)) ||
						strings.EqualFold(result.SMTPFrom, tt.email)) {
						found = true

						break
					}
					if tt.field == "cc" && containsEmailInAddresses(result.CcAddressList, strings.ToLower(tt.email)) {
						found = true

						break
					}
				}
				if !found {
					t.Errorf("SearchByField() results don't contain expected email %s in field %s", tt.email, tt.field)
				}
			}
		})
	}
}

func TestSearchByFieldErrors(t *testing.T) {
	backend := &smtpBackend{}

	tests := []struct {
		name    string
		field   string
		email   string
		wantErr bool
	}{
		{
			name:    "invalid_field",
			field:   "invalid",
			email:   "test@example.com",
			wantErr: true,
		},
		{
			name:    "valid_field_to",
			field:   "to",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid_field_from",
			field:   "from",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid_field_cc",
			field:   "cc",
			email:   "test@example.com",
			wantErr: false,
		},
		{
			name:    "valid_field_bcc",
			field:   "bcc",
			email:   "test@example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := backend.SearchByField(tt.field, tt.email)
			if (err != nil) != tt.wantErr {
				t.Errorf("SearchByField() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContainsEmailInAddresses(t *testing.T) {
	addresses := []*mail.Address{
		{Name: "John Doe", Address: "john@example.com"},
		{Name: "Jane Smith", Address: "jane@example.com"},
		{Name: "", Address: "no-name@example.com"},
	}

	tests := []struct {
		name        string
		addresses   []*mail.Address
		searchEmail string
		want        bool
	}{
		{
			name:        "email_found",
			addresses:   addresses,
			searchEmail: "john@example.com",
			want:        true,
		},
		{
			name:        "email_not_found",
			addresses:   addresses,
			searchEmail: "notfound@example.com",
			want:        false,
		},
		{
			name:        "case_insensitive_found",
			addresses:   addresses,
			searchEmail: "JANE@EXAMPLE.COM",
			want:        true,
		},
		{
			name:        "empty_addresses",
			addresses:   []*mail.Address{},
			searchEmail: "test@example.com",
			want:        false,
		},
		{
			name:        "nil_addresses",
			addresses:   nil,
			searchEmail: "test@example.com",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsEmailInAddresses(tt.addresses, strings.ToLower(tt.searchEmail)); got != tt.want {
				t.Errorf("containsEmailInAddresses() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContainsEmailInStrings(t *testing.T) {
	emails := []string{
		"test1@example.com",
		"test2@example.com",
		"test3@example.com",
	}

	tests := []struct {
		name        string
		emails      []string
		searchEmail string
		want        bool
	}{
		{
			name:        "email_found",
			emails:      emails,
			searchEmail: "test2@example.com",
			want:        true,
		},
		{
			name:        "email_not_found",
			emails:      emails,
			searchEmail: "notfound@example.com",
			want:        false,
		},
		{
			name:        "case_insensitive_found",
			emails:      emails,
			searchEmail: "TEST1@EXAMPLE.COM",
			want:        true,
		},
		{
			name:        "empty_emails",
			emails:      []string{},
			searchEmail: "test@example.com",
			want:        false,
		},
		{
			name:        "nil_emails",
			emails:      nil,
			searchEmail: "test@example.com",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := containsEmailInStrings(tt.emails, strings.ToLower(tt.searchEmail)); got != tt.want {
				t.Errorf("containsEmailInStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAddressHeader(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   bool
	}{
		{"to_header", "to", true},
		{"cc_header", "cc", true},
		{"bcc_header", "bcc", true},
		{"from_header", "from", true},
		{"case_insensitive_to", "TO", true},
		{"case_insensitive_from", "FROM", true},
		{"subject_header", "subject", false},
		{"date_header", "date", false},
		{"empty_header", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAddressHeader(tt.header); got != tt.want {
				t.Errorf("isAddressHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSMTPSessionDataCapture(t *testing.T) {
	// Test that SMTP session methods properly capture data
	session := &smtpSession{
		receivedTime: time.Now(),
		rcptTo:       make([]string, 0),
		rcptOpts:     make([]*smtp.RcptOptions, 0),
	}

	// Test Mail method
	testFrom := "sender@example.com"
	mailOpts := &smtp.MailOptions{Size: 1024}
	err := session.Mail(testFrom, mailOpts)
	if err != nil {
		t.Errorf("Mail() error = %v", err)
	}
	if session.mailFrom != testFrom {
		t.Errorf("Mail() mailFrom = %v, want %v", session.mailFrom, testFrom)
	}
	if session.mailOpts != mailOpts {
		t.Errorf("Mail() mailOpts not captured correctly")
	}

	// Test Rcpt method
	testTo1 := "recipient1@example.com"
	testTo2 := "recipient2@example.com"
	rcptOpts := &smtp.RcptOptions{}

	err = session.Rcpt(testTo1, rcptOpts)
	if err != nil {
		t.Errorf("Rcpt() error = %v", err)
	}
	err = session.Rcpt(testTo2, rcptOpts)
	if err != nil {
		t.Errorf("Rcpt() error = %v", err)
	}

	if len(session.rcptTo) != 2 {
		t.Errorf("Rcpt() rcptTo length = %d, want 2", len(session.rcptTo))
	}
	if session.rcptTo[0] != testTo1 || session.rcptTo[1] != testTo2 {
		t.Errorf("Rcpt() rcptTo = %v, want [%s, %s]", session.rcptTo, testTo1, testTo2)
	}
}

// Helper functions to create test email data.
func createTestEmailData(from, to, subject string) string {
	return "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		"This is a test email body.\r\n"
}

func createTestEmailWithCC(from, to, cc, subject string) string {
	return "From: " + from + "\r\n" +
		"To: " + to + "\r\n" +
		"Cc: " + cc + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"\r\n" +
		"This is a test email body with CC.\r\n"
}
