package fakesmtpserver

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"
	"github.com/sters/go-fake-smtp-server/config"
)

// ErrInvalidSearchField is returned when an invalid search field is specified.
var ErrInvalidSearchField = errors.New("invalid search field")

const (
	// Field names for search validation.
	FieldTo   = "to"
	FieldCC   = "cc"
	FieldBCC  = "bcc"
	FieldFrom = "from"
)

type (
	smtpView struct {
		// Email Content (parsed from data via enmime)
		Headers         []*smtpViewHeader `json:"headers"`
		FromAddressList []*mail.Address   `json:"from"` // From header
		ToAddressList   []*mail.Address   `json:"to"`   // To header
		CcAddressList   []*mail.Address   `json:"cc"`   // CC header
		BccAddressList  []*mail.Address   `json:"bcc"`  // BCC header
		Text            string            `json:"text"`
		HTML            string            `json:"html"`

		// SMTP Transaction Data (from session)
		SMTPFrom     string    `json:"smtpFrom"`     // MAIL FROM address
		SMTPTo       []string  `json:"smtpTo"`       // RCPT TO addresses
		ReceivedTime time.Time `json:"receivedTime"` // Session timestamp

		// Connection Metadata
		ClientAddr string `json:"clientAddr"` // Remote IP
		ClientHost string `json:"clientHost"` // HELO/EHLO hostname
		TLSUsed    bool   `json:"tlsUsed"`    // TLS connection

		// Authentication (if implemented)
		Authenticated bool   `json:"authenticated"` // Auth success
		AuthMechanism string `json:"authMechanism"` // PLAIN, LOGIN, etc.
	}

	smtpViewHeader struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
)

type smtpBackend struct {
	sessions []*smtpSession
	mux      sync.RWMutex
}

var sharedBackend = &smtpBackend{} //nolint:gochecknoglobals

func (b *smtpBackend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
	slog.Info("NewSession")

	_, tlsOK := conn.TLSConnectionState()
	s := &smtpSession{
		receivedTime: time.Now(),
		clientAddr:   conn.Conn().RemoteAddr().String(),
		clientHost:   conn.Hostname(),
		tlsUsed:      tlsOK,
		rcptTo:       make([]string, 0),
		rcptOpts:     make([]*smtp.RcptOptions, 0),
	}

	b.mux.Lock()
	b.sessions = append(b.sessions, s)
	b.mux.Unlock()

	return s, nil
}

func (b *smtpBackend) GetAllData() []smtpView {
	b.mux.RLock()
	sessions := make([]*smtpSession, len(b.sessions))
	copy(sessions, b.sessions)
	b.mux.RUnlock()

	result := make([]smtpView, len(sessions))
	for i, session := range sessions {
		view := smtpView{
			// SMTP transaction data
			SMTPFrom:      session.mailFrom,
			SMTPTo:        session.rcptTo,
			ReceivedTime:  session.receivedTime,
			ClientAddr:    session.clientAddr,
			ClientHost:    session.clientHost,
			TLSUsed:       session.tlsUsed,
			Authenticated: session.authenticated,
			AuthMechanism: session.authMechanism,
		}

		// Parse email content if available
		if session.data != "" {
			e, err := enmime.ReadEnvelope(strings.NewReader(session.data))
			if err != nil {
				slog.Info("failed to read envelope", "error", err)
				view.Text = "cannot parse this mail"
			} else {
				view.FromAddressList = getAddressList(e, "from")
				view.ToAddressList = getAddressList(e, "to")
				view.CcAddressList = getAddressList(e, "cc")
				view.BccAddressList = getAddressList(e, "bcc")
				view.Text = e.Text
				view.HTML = e.HTML

				// Parse headers (excluding address headers)
				keys := e.GetHeaderKeys()
				view.Headers = make([]*smtpViewHeader, 0, len(keys))
				for _, h := range keys {
					if !isAddressHeader(h) {
						view.Headers = append(view.Headers, &smtpViewHeader{
							Key:   h,
							Value: e.GetHeader(h),
						})
					}
				}
			}
		}

		result[i] = view
	}

	return result
}

func getAddressList(e *enmime.Envelope, key string) []*mail.Address {
	addrList, err := e.AddressList(key)
	if err != nil {
		return []*mail.Address{}
	}

	return addrList
}

func isAddressHeader(header string) bool {
	switch strings.ToLower(header) {
	case "to", "cc", "bcc", "from":
		return true
	default:
		return false
	}
}

// SearchByField searches for emails containing the specified email address in the given field.
func (b *smtpBackend) SearchByField(field, email string) ([]smtpView, error) {
	// Validate field parameter
	if field == "" {
		return nil, fmt.Errorf("%w: empty field", ErrInvalidSearchField)
	}

	// Validate field name before processing
	switch field {
	case FieldTo, FieldCC, FieldBCC, FieldFrom:
		// Valid field, continue
	default:
		return nil, fmt.Errorf("%w: %s", ErrInvalidSearchField, field)
	}

	allData := b.GetAllData()
	var results []smtpView

	searchEmail := strings.ToLower(email)

	for _, msg := range allData {
		var found bool

		switch field {
		case FieldTo:
			// Search both To header and SMTP RCPT TO
			found = containsEmailInAddresses(msg.ToAddressList, searchEmail) ||
				containsEmailInStrings(msg.SMTPTo, searchEmail)
		case FieldCC:
			found = containsEmailInAddresses(msg.CcAddressList, searchEmail)
		case FieldBCC:
			// BCC only visible in SMTP transaction, not in headers
			found = containsEmailInAddresses(msg.BccAddressList, searchEmail)
		case FieldFrom:
			// Search both From header and SMTP MAIL FROM
			found = containsEmailInAddresses(msg.FromAddressList, searchEmail) ||
				strings.ToLower(msg.SMTPFrom) == searchEmail
		}

		if found {
			results = append(results, msg)
		}
	}

	return results, nil
}

// containsEmailInAddresses checks if an email address is present in a slice of mail.Address.
func containsEmailInAddresses(addresses []*mail.Address, searchEmail string) bool {
	for _, addr := range addresses {
		if addr != nil && strings.ToLower(addr.Address) == searchEmail {
			return true
		}
	}

	return false
}

// containsEmailInStrings checks if an email address is present in a slice of strings.
func containsEmailInStrings(emails []string, searchEmail string) bool {
	for _, email := range emails {
		if strings.ToLower(email) == searchEmail {
			return true
		}
	}

	return false
}

type smtpSession struct {
	// Existing fields
	data         string
	receivedTime time.Time

	// SMTP Transaction Data
	mailFrom string              // MAIL FROM address
	mailOpts *smtp.MailOptions   // MAIL FROM options (SIZE, BODY, etc.)
	rcptTo   []string            // All RCPT TO addresses
	rcptOpts []*smtp.RcptOptions // RCPT TO options (DSN, etc.)

	// Connection Info
	clientAddr string // Client IP address
	clientHost string // HELO/EHLO hostname
	tlsUsed    bool   // Whether TLS was used

	// Authentication (for future enhancement)
	authenticated bool   // Whether auth succeeded
	authMechanism string // PLAIN, LOGIN, etc.
}

var _ smtp.Session = (*smtpSession)(nil)

func (s *smtpSession) AuthPlain(_, _ string) error {
	return nil
}

func (s *smtpSession) Mail(from string, opts *smtp.MailOptions) error {
	s.mailFrom = from
	s.mailOpts = opts

	return nil
}

func (s *smtpSession) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.rcptTo = append(s.rcptTo, to)
	s.rcptOpts = append(s.rcptOpts, opts)

	return nil
}

func (s *smtpSession) Data(r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read data error: %w", err)
	}

	s.data = string(b)
	// slog.Info("Received data", "data", s.data)

	return nil
}

func (s *smtpSession) Reset() {}

func (s *smtpSession) Logout() error {
	return nil
}

func StartSMTPServer(cfg *config.Config) error {
	s := smtp.NewServer(sharedBackend)

	s.Addr = cfg.SMTPAddr
	s.Domain = cfg.SMTPHostname
	s.ReadTimeout = cfg.SMTPReadTimeout
	s.WriteTimeout = cfg.SMTPWriteTimeout
	s.MaxMessageBytes = cfg.SMTPMaxMessageBytes
	s.MaxRecipients = cfg.SMTPMaxRecipients
	s.AllowInsecureAuth = cfg.SMTPAllowInsecureAuth
	// s.Debug = os.Stdout

	slog.Info("Starting SMTP fake server", "addr", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		slog.Info("err", "error", err)

		return fmt.Errorf("smtp server error: %w", err)
	}

	return nil
}
