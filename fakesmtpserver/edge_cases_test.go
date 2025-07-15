package fakesmtpserver

import (
	"net/mail"
	"strings"
	"testing"
	"time"
)

//nolint:maintidx // Comprehensive test coverage requires high complexity
func TestEdgeCasesAndErrorHandling(t *testing.T) {
	t.Run("empty_backend", func(t *testing.T) {
		backend := &smtpBackend{}
		results, err := backend.SearchByField("to", "test@example.com")
		if err != nil {
			t.Errorf("SearchByField() on empty backend should not error, got: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("SearchByField() on empty backend should return empty results, got %d", len(results))
		}
	})

	t.Run("session_with_empty_data", func(t *testing.T) {
		backend := &smtpBackend{}
		session := &smtpSession{
			data:         "", // Empty email data
			receivedTime: time.Now(),
			mailFrom:     "sender@example.com",
			rcptTo:       []string{"recipient@example.com"},
			clientAddr:   "192.168.1.100:12345",
			clientHost:   "client.example.com",
			tlsUsed:      false,
		}
		backend.sessions = []*smtpSession{session}

		// Should still find in SMTP data even if email data is empty
		results, err := backend.SearchByField("from", "sender@example.com")
		if err != nil {
			t.Errorf("SearchByField() with empty data should not error, got: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("SearchByField() should find SMTP data even with empty email, got %d results", len(results))
		}
	})

	t.Run("session_with_malformed_email_data", func(t *testing.T) {
		backend := &smtpBackend{}
		session := &smtpSession{
			data:         "This is not a valid email format at all!!!", // Malformed email
			receivedTime: time.Now(),
			mailFrom:     "sender@example.com",
			rcptTo:       []string{"recipient@example.com"},
			clientAddr:   "192.168.1.100:12345",
			clientHost:   "client.example.com",
			tlsUsed:      false,
		}
		backend.sessions = []*smtpSession{session}

		// Should still work and find SMTP data
		results, err := backend.SearchByField("from", "sender@example.com")
		if err != nil {
			t.Errorf("SearchByField() with malformed email should not error, got: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("SearchByField() should find SMTP data even with malformed email, got %d results", len(results))
		}

		// Check that the result has error text for malformed email
		if results[0].Text != "cannot parse this mail" {
			t.Errorf("Expected error text for malformed email, got: %s", results[0].Text)
		}
	})

	t.Run("case_sensitivity_comprehensive", func(t *testing.T) {
		backend := &smtpBackend{}
		session := &smtpSession{
			data:         createTestEmailData("Sender@Example.COM", "Recipient@Example.COM", "Test Subject"),
			receivedTime: time.Now(),
			mailFrom:     "SMTP-Sender@Example.COM",
			rcptTo:       []string{"SMTP-Recipient@Example.COM"},
			clientAddr:   "192.168.1.100:12345",
			clientHost:   "client.example.com",
			tlsUsed:      true,
		}
		backend.sessions = []*smtpSession{session}

		// Test various case combinations
		testCases := []struct {
			field string
			email string
		}{
			{"from", "sender@example.com"},
			{"from", "SENDER@EXAMPLE.COM"},
			{"from", "SenDer@ExamPle.CoM"},
			{"from", "smtp-sender@example.com"},
			{"to", "recipient@example.com"},
			{"to", "RECIPIENT@EXAMPLE.COM"},
			{"to", "smtp-recipient@example.com"},
		}

		for _, tc := range testCases {
			results, err := backend.SearchByField(tc.field, tc.email)
			if err != nil {
				t.Errorf("SearchByField() case sensitivity test failed: %v", err)
			}
			if len(results) != 1 {
				t.Errorf("SearchByField() case sensitivity: field=%s, email=%s should find 1 result, got %d", tc.field, tc.email, len(results))
			}
		}
	})

	t.Run("unicode_and_special_characters", func(t *testing.T) {
		backend := &smtpBackend{}
		session := &smtpSession{
			data:         createTestEmailData("测试@example.com", "αβγ@example.com", "Test Subject with 中文"),
			receivedTime: time.Now(),
			mailFrom:     "测试@example.com",
			rcptTo:       []string{"αβγ@example.com", "مرحبا@example.com"},
			clientAddr:   "192.168.1.100:12345",
			clientHost:   "client.example.com",
			tlsUsed:      true,
		}
		backend.sessions = []*smtpSession{session}

		// Test unicode email addresses
		results, err := backend.SearchByField("from", "测试@example.com")
		if err != nil {
			t.Errorf("SearchByField() with unicode should not error, got: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("SearchByField() should handle unicode emails, got %d results", len(results))
		}

		// Test unicode in SMTP data
		results, err = backend.SearchByField("to", "مرحبا@example.com")
		if err != nil {
			t.Errorf("SearchByField() with unicode SMTP data should not error, got: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("SearchByField() should find unicode SMTP recipients, got %d results", len(results))
		}
	})

	t.Run("empty_email_addresses", func(t *testing.T) {
		backend := &smtpBackend{}
		session := &smtpSession{
			data:         "From: \r\nTo: \r\nSubject: Empty addresses\r\n\r\nTest",
			receivedTime: time.Now(),
			mailFrom:     "",           // Empty SMTP from
			rcptTo:       []string{""}, // Empty SMTP to
			clientAddr:   "192.168.1.100:12345",
			clientHost:   "client.example.com",
			tlsUsed:      false,
		}
		backend.sessions = []*smtpSession{session}

		// Searching for empty string should not crash and should find the empty SMTP from
		results, err := backend.SearchByField("from", "")
		if err != nil {
			t.Errorf("SearchByField() with empty email should not error, got: %v", err)
		}
		// Should find the session with empty SMTP from
		if len(results) != 1 {
			t.Errorf("SearchByField() with empty email should find session with empty SMTP from, got %d", len(results))
		}

		// Searching for a non-empty email should not find anything
		results, err = backend.SearchByField("from", "test@example.com")
		if err != nil {
			t.Errorf("SearchByField() should not error, got: %v", err)
		}
		if len(results) != 0 {
			t.Errorf("SearchByField() should not find non-empty email in empty session, got %d", len(results))
		}
	})

	t.Run("nil_and_empty_address_lists", func(t *testing.T) {
		// Test helper functions with nil and empty inputs
		if containsEmailInAddresses(nil, "test@example.com") {
			t.Error("containsEmailInAddresses() should return false for nil addresses")
		}

		if containsEmailInAddresses([]*mail.Address{}, "test@example.com") {
			t.Error("containsEmailInAddresses() should return false for empty addresses")
		}

		if containsEmailInStrings(nil, "test@example.com") {
			t.Error("containsEmailInStrings() should return false for nil strings")
		}

		if containsEmailInStrings([]string{}, "test@example.com") {
			t.Error("containsEmailInStrings() should return false for empty strings")
		}
	})

	t.Run("mixed_valid_invalid_addresses", func(t *testing.T) {
		addresses := []*mail.Address{
			{Name: "Valid User", Address: "valid@example.com"},
			{Name: "Invalid User", Address: ""}, // Empty address
			{Name: "", Address: "another@example.com"},
			nil, // Nil address (shouldn't happen but let's be safe)
		}

		// Should find valid addresses despite invalid ones
		if !containsEmailInAddresses(addresses, "valid@example.com") {
			t.Error("containsEmailInAddresses() should find valid address despite invalid ones")
		}

		if !containsEmailInAddresses(addresses, "another@example.com") {
			t.Error("containsEmailInAddresses() should find second valid address")
		}

		if containsEmailInAddresses(addresses, "nonexistent@example.com") {
			t.Error("containsEmailInAddresses() should not find nonexistent address")
		}
	})

	t.Run("very_long_email_addresses", func(t *testing.T) {
		// Test with very long email addresses
		longLocal := strings.Repeat("a", 64)           // Max local part length
		longDomain := strings.Repeat("b", 63) + ".com" // Long domain
		longEmail := longLocal + "@" + longDomain

		backend := &smtpBackend{}
		session := &smtpSession{
			data:         createTestEmailData(longEmail, "recipient@example.com", "Test Subject"),
			receivedTime: time.Now(),
			mailFrom:     longEmail,
			rcptTo:       []string{"recipient@example.com"},
			clientAddr:   "192.168.1.100:12345",
			clientHost:   "client.example.com",
			tlsUsed:      true,
		}
		backend.sessions = []*smtpSession{session}

		results, err := backend.SearchByField("from", longEmail)
		if err != nil {
			t.Errorf("SearchByField() with long email should not error, got: %v", err)
		}
		if len(results) != 1 {
			t.Errorf("SearchByField() should handle long emails, got %d results", len(results))
		}
	})

	t.Run("concurrent_access_safety", func(t *testing.T) {
		// Test that concurrent access doesn't cause race conditions
		backend := &smtpBackend{}
		setupTestData(backend)

		// Run multiple searches concurrently
		results := make(chan error, 10)

		for range 10 {
			go func() {
				_, err := backend.SearchByField("to", "recipient1@example.com")
				results <- err
			}()
		}

		// Check that all searches completed without error
		for range 10 {
			err := <-results
			if err != nil {
				t.Errorf("Concurrent search failed: %v", err)
			}
		}
	})

	t.Run("search_field_boundary_cases", func(t *testing.T) {
		backend := &smtpBackend{}

		// Test all invalid field names
		invalidFields := []string{
			"",
			"invalid",
			"TO",   // Wrong case
			"FROM", // Wrong case
			"email",
			"subject",
			"body",
			"sender",
			"recipient",
		}

		for _, field := range invalidFields {
			_, err := backend.SearchByField(field, "test@example.com")
			if err == nil {
				t.Errorf("SearchByField() should error for invalid field '%s'", field)

				continue
			}

			// Check that it returns the correct error type
			if !strings.Contains(err.Error(), "invalid search field") {
				t.Errorf("SearchByField() should return invalid search field error for '%s', got: %v", field, err)
			}
		}
	})
}

func TestDataIntegrityAfterSearch(t *testing.T) {
	// Ensure that search operations don't modify the original data
	backend := &smtpBackend{}
	originalSession := &smtpSession{
		data:         createTestEmailData("sender@example.com", "recipient@example.com", "Test Subject"),
		receivedTime: time.Now(),
		mailFrom:     "sender@example.com",
		rcptTo:       []string{"recipient@example.com", "recipient2@example.com"},
		clientAddr:   "192.168.1.100:12345",
		clientHost:   "client.example.com",
		tlsUsed:      true,
	}

	backend.sessions = []*smtpSession{originalSession}

	// Store original values
	originalData := originalSession.data
	originalMailFrom := originalSession.mailFrom
	originalRcptTo := make([]string, len(originalSession.rcptTo))
	copy(originalRcptTo, originalSession.rcptTo)

	// Perform search
	_, err := backend.SearchByField("to", "recipient@example.com")
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	// Verify data hasn't changed
	if originalSession.data != originalData {
		t.Error("Search operation modified session data")
	}
	if originalSession.mailFrom != originalMailFrom {
		t.Error("Search operation modified session mailFrom")
	}
	if len(originalSession.rcptTo) != len(originalRcptTo) {
		t.Error("Search operation modified session rcptTo length")
	}
	for i, addr := range originalSession.rcptTo {
		if addr != originalRcptTo[i] {
			t.Error("Search operation modified session rcptTo content")
		}
	}
}
