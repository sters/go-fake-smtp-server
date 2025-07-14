package fakesmtpserver

import (
	"fmt"
	"io"
	"log/slog"
	"net/mail"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/jhillyerd/enmime"
)

const (
	SMTPAddr = "127.0.0.1:10025"
	HOSTNAME = "fakeserver"
)

type (
	smtpView struct {
		Headers        []*smtpViewHeader `json:"headers"`
		ToAddressList  []*mail.Address   `json:"to"`
		CcAddressList  []*mail.Address   `json:"cc"`
		BccAddressList []*mail.Address   `json:"bcc"`
		Text           string            `json:"text"`
		HTML           string            `json:"html"`
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

func (b *smtpBackend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	slog.Info("NewSession")

	s := &smtpSession{
		receivedTime: time.Now(),
	}

	b.mux.Lock()
	b.sessions = append(b.sessions, s)
	b.mux.Unlock()

	return s, nil
}

func (b *smtpBackend) GetAllData() []smtpView {
	b.mux.RLock()
	messages := make([]string, len(b.sessions))
	for i, s := range b.sessions {
		messages[i] = s.data
	}
	b.mux.RUnlock()

	result := make([]smtpView, len(messages))
	for i, s := range messages {
		e, err := enmime.ReadEnvelope(strings.NewReader(s))
		if err != nil {
			slog.Info("failed to read env", "error", err)
			result[i] = smtpView{
				Text: "cannot parse this mail",
			}

			continue
		}

		s := smtpView{
			ToAddressList:  getAddressList(e, "to"),
			CcAddressList:  getAddressList(e, "cc"),
			BccAddressList: getAddressList(e, "bcc"),
			Text:           e.Text,
			HTML:           e.HTML,
		}

		keys := e.GetHeaderKeys()
		s.Headers = make([]*smtpViewHeader, 0, len(keys))
		for _, h := range keys {
			// ignore to/cc/bcc
			switch strings.ToLower(h) {
			case "to", "cc", "bcc":

				continue
			}

			s.Headers = append(s.Headers, &smtpViewHeader{
				Key:   h,
				Value: e.GetHeader(h),
			})
		}

		result[i] = s
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

type smtpSession struct {
	data         string
	receivedTime time.Time
}

var _ smtp.Session = (*smtpSession)(nil)

func (s *smtpSession) AuthPlain(_, _ string) error {
	return nil
}

func (s *smtpSession) Mail(_ string, _ *smtp.MailOptions) error {
	return nil
}

func (s *smtpSession) Rcpt(string, *smtp.RcptOptions) error {
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

func StartSMTPServer() error {
	s := smtp.NewServer(sharedBackend)

	s.Addr = SMTPAddr
	s.Domain = HOSTNAME
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true
	// s.Debug = os.Stdout

	slog.Info("Starting SMTP fake server", "addr", s.Addr)
	if err := s.ListenAndServe(); err != nil {
		slog.Info("err", "error", err)

		return fmt.Errorf("smtp server error: %w", err)
	}

	return nil
}
