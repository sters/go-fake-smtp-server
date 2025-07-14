# ADR-001: Email Search Endpoints

## Status
Proposed

## Context
The current fake SMTP server provides a single endpoint `/` that returns all captured emails as JSON. Users need the ability to search for specific emails based on email addresses in different fields (To, CC, BCC, From). This would improve usability for testing scenarios where users need to find specific emails quickly.

## Decision
We will implement four new search endpoints that allow filtering emails by email addresses:

- `GET /search/to?email={email}` - Find emails where the email appears in To field
- `GET /search/cc?email={email}` - Find emails where the email appears in CC field  
- `GET /search/bcc?email={email}` - Find emails where the email appears in BCC field
- `GET /search/from?email={email}` - Find emails where the email appears in From field

## API Design

### Request Format
```
GET /search/{field}?email={email_address}
```

Where:
- `{field}` is one of: `to`, `cc`, `bcc`, `from`
- `{email_address}` is the email address to search for (URL encoded)

### Response Format
The response will maintain the same JSON structure as the existing `/` endpoint but only return matching emails:

```json
[
  {
    "headers": [
      {"key": "Subject", "value": "Test Email"},
      {"key": "From", "value": "sender@example.com"}
    ],
    "to": [{"Name": "", "Address": "recipient@example.com"}],
    "cc": [{"Name": "", "Address": "cc@example.com"}],
    "bcc": [],
    "text": "Email body text",
    "html": "<p>Email body HTML</p>"
  }
]
```

### Error Responses
- `400 Bad Request` - Missing or invalid email parameter
- `404 Not Found` - Invalid search field
- `500 Internal Server Error` - Server processing error

## Implementation Plan

### Phase 1: Enhanced Data Capture Infrastructure
1. **Upgrade smtpSession struct**
   - Add `mailFrom` and `mailOpts` fields for MAIL FROM data
   - Add `rcptTo` slice and `rcptOpts` slice for RCPT TO data
   - Add `clientAddr`, `clientHost`, `tlsUsed` for connection info
   - Add `authenticated`, `authMechanism` for auth state

2. **Update SMTP session methods**
   - Modify `Mail(from string, opts *smtp.MailOptions)` to store `from` and `opts`
   - Modify `Rcpt(to string, opts *smtp.RcptOptions)` to append `to` and `opts`
   - Capture connection info in `NewSession(conn *smtp.Conn)` using:
     - `conn.Conn().RemoteAddr()` for client IP
     - `conn.Hostname()` for HELO/EHLO hostname
     - `conn.TLSConnectionState()` for TLS usage

3. **Handle authentication (optional enhancement)**
   - Implement `AuthSession` interface if auth tracking needed
   - Use `AuthMechanisms()` and `Auth()` methods to track auth state
   - Store mechanism type but not credentials (privacy/security)

4. **Enhance smtpView struct**
   - Add SMTP transaction fields (`SMTPFrom`, `SMTPTo`, `MailOptions`, etc.)
   - Add connection metadata fields (`ClientAddr`, `ClientHost`, `TLSUsed`)
   - Add authentication fields (`Authenticated`, `AuthMechanism`)
   - Update JSON tags for API response

5. **Update GetAllData() method**
   - Populate both email header data (via enmime) and SMTP transaction data
   - Handle cases where SMTP envelope differs from email headers
   - Ensure backward compatibility with existing `/` endpoint
   - Add proper error handling for incomplete sessions

### Phase 2: Search Infrastructure
1. **Create unified search functions**
   - `SearchByField(field, email string) ([]smtpView, error)`
   - Helper functions for different data types
   - Case-insensitive email comparison

2. **Implement dual-source search logic**
   - Search email headers (From, To, CC, BCC)
   - Search SMTP transaction data (MAIL FROM, RCPT TO)
   - Handle discrepancies between header and SMTP data

### Phase 3: HTTP Endpoints
1. **Add new HTTP routes**
   - `GET /search/to`
   - `GET /search/cc`
   - `GET /search/bcc`
   - `GET /search/from`

2. **Add request validation**
   - Validate email parameter exists
   - Validate email format (basic validation)
   - Validate search field parameter

3. **Add error handling**
   - Proper HTTP status codes
   - JSON error responses
   - Logging for debugging

### Phase 4: Testing & Documentation
1. **Add tests**
   - Unit tests for search functions
   - Integration tests for HTTP endpoints
   - Test edge cases (invalid emails, no matches)

2. **Update documentation**
   - Update CLAUDE.md with new endpoints
   - Add usage examples
   - Document API specification

## Implementation Examples

### Enhanced Session Methods
```go
func (b *smtpBackend) NewSession(conn *smtp.Conn) (smtp.Session, error) {
    slog.Info("NewSession")
    
    s := &smtpSession{
        receivedTime: time.Now(),
        clientAddr:   conn.Conn().RemoteAddr().String(),
        clientHost:   conn.Hostname(),
        tlsUsed:      conn.TLSConnectionState() != nil,
        rcptTo:       make([]string, 0),
        rcptOpts:     make([]*smtp.RcptOptions, 0),
    }
    
    b.mux.Lock()
    b.sessions = append(b.sessions, s)
    b.mux.Unlock()
    
    return s, nil
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
```

### Enhanced GetAllData() Method
```go
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
            SMTPMailOpts:  session.mailOpts,
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
                
                // Parse headers (excluding to/cc/bcc)
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

func isAddressHeader(header string) bool {
    switch strings.ToLower(header) {
    case "to", "cc", "bcc", "from":
        return true
    default:
        return false
    }
}
```

## Technical Considerations

### Enhanced Data Capture
- **SMTP vs Email Headers**: SMTP transaction data (MAIL FROM, RCPT TO) can differ from email headers
- **BCC Handling**: BCC recipients are only visible in SMTP transaction, not email headers  
- **Multiple Recipients**: RCPT TO can have multiple addresses that may not all appear in headers
- **Envelope vs Content**: SMTP envelope addresses may differ from header addresses (forwarding, aliases)
- **Connection Timing**: Connection info available only during `NewSession()`, must be captured immediately
- **Incomplete Sessions**: Handle sessions where `Data()` hasn't been called yet (SMTP transaction in progress)

### Library-Specific Considerations
- **go-smtp Interface**: Current session interface doesn't include `AuthPlain()` - authentication requires `AuthSession`
- **TLS Detection**: `conn.TLSConnectionState()` returns `nil` for non-TLS connections
- **Address Parsing**: Use `net.ParseIP()` for proper IPv4/IPv6 handling of `RemoteAddr()`
- **Mail Options**: `smtp.MailOptions` contains extension data (SIZE, BODY type, UTF8, DSN parameters)
- **Thread Safety**: All session data modifications must be protected by session-level mutex if needed

### Performance
- **Memory usage**: Search operates on in-memory data, acceptable for fake server use case
- **Search complexity**: O(n*m) where n = number of emails, m = number of addresses per field
- **Optimization**: Could add indexing later if performance becomes an issue

### Data Structure Changes

#### Enhanced SMTP Session Data Capture
Currently, `smtpSession` only stores the final email data. Based on `github.com/emersion/go-smtp` library capabilities, we can capture:

```go
type smtpSession struct {
    // Existing fields
    data         string              // Final email data
    receivedTime time.Time          // When session was created
    
    // SMTP Transaction Data (from session methods)
    mailFrom     string              // From Mail(from, opts) method
    mailOpts     *smtp.MailOptions   // SIZE, BODY, UTF8 extensions
    rcptTo       []string           // From Rcpt(to, opts) calls
    rcptOpts     []*smtp.RcptOptions // DSN, DELIVERBY extensions
    
    // Connection Info (from smtp.Conn)
    clientAddr   string             // conn.Conn().RemoteAddr()
    clientHost   string             // conn.Hostname() - HELO/EHLO
    tlsUsed      bool               // conn.TLSConnectionState() != nil
    
    // Authentication (requires AuthSession interface)
    authenticated    bool           // Whether auth succeeded
    authMechanism   string          // PLAIN, LOGIN, etc.
    // Note: username/password not directly accessible in current interface
}
```

#### Updated View Structure
```go
type smtpView struct {
    // Email Content (parsed from data via enmime)
    Headers         []*smtpViewHeader `json:"headers"`
    FromAddressList []*mail.Address   `json:"from"`          // From header
    ToAddressList   []*mail.Address   `json:"to"`            // To header  
    CcAddressList   []*mail.Address   `json:"cc"`            // CC header
    BccAddressList  []*mail.Address   `json:"bcc"`           // BCC header
    Text            string            `json:"text"`
    HTML            string            `json:"html"`
    
    // SMTP Transaction Data (from session)
    SMTPFrom        string               `json:"smtp_from"`      // MAIL FROM address
    SMTPTo          []string             `json:"smtp_to"`        // RCPT TO addresses
    SMTPMailOpts    *smtp.MailOptions    `json:"smtp_mail_opts"` // SIZE, BODY extensions
    ReceivedTime    time.Time            `json:"received_time"`  // Session timestamp
    
    // Connection Metadata
    ClientAddr      string            `json:"client_addr"`     // Remote IP
    ClientHost      string            `json:"client_host"`     // HELO/EHLO hostname
    TLSUsed         bool              `json:"tls_used"`        // TLS connection
    
    // Authentication (if implemented)
    Authenticated   bool              `json:"authenticated"`   // Auth success
    AuthMechanism   string            `json:"auth_mechanism"`  // PLAIN, LOGIN, etc.
}
```

### Search Logic
With enhanced data capture, we can search both email headers and SMTP transaction data:

```go
func (b *smtpBackend) SearchByField(field, email string) ([]smtpView, error) {
    allData := b.GetAllData()
    var results []smtpView
    
    for _, msg := range allData {
        var found bool
        searchEmail := strings.ToLower(email)
        
        switch field {
        case "to":
            // Search both To header and SMTP RCPT TO
            found = containsEmailInAddresses(msg.ToAddressList, searchEmail) ||
                   containsEmailInStrings(msg.SMTPTo, searchEmail)
        case "cc":
            found = containsEmailInAddresses(msg.CcAddressList, searchEmail)
        case "bcc":
            // BCC only visible in SMTP transaction, not in headers
            found = containsEmailInAddresses(msg.BccAddressList, searchEmail)
        case "from":
            // Search both From header and SMTP MAIL FROM
            found = containsEmailInAddresses(msg.FromAddressList, searchEmail) ||
                   strings.ToLower(msg.SMTPFrom) == searchEmail
        default:
            return nil, fmt.Errorf("invalid field: %s", field)
        }
        
        if found {
            results = append(results, msg)
        }
    }
    
    return results, nil
}

func containsEmailInAddresses(addresses []*mail.Address, searchEmail string) bool {
    for _, addr := range addresses {
        if strings.ToLower(addr.Address) == searchEmail {
            return true
        }
    }
    return false
}

func containsEmailInStrings(emails []string, searchEmail string) bool {
    for _, email := range emails {
        if strings.ToLower(email) == searchEmail {
            return true
        }
    }
    return false
}
```

## Alternatives Considered

1. **Single search endpoint with field parameter**
   - `GET /search?field=to&email=test@example.com`
   - **Rejected**: Less RESTful, more complex parameter validation

2. **POST endpoint with JSON body**
   - More flexible for complex queries
   - **Rejected**: Overkill for simple email filtering, GET is more appropriate

3. **Regex/wildcard search support**
   - Allow pattern matching in email addresses
   - **Deferred**: Can be added later if needed, YAGNI principle

## Consequences

### Positive
- **Improved usability**: Users can quickly find specific emails
- **Better testing workflows**: Easier to verify email delivery in tests
- **RESTful API**: Clean, predictable endpoint structure
- **Backward compatibility**: Existing `/` endpoint remains unchanged

### Negative
- **Increased complexity**: More code to maintain
- **Performance impact**: Additional processing for search operations
- **API surface growth**: More endpoints to document and test

## Implementation Timeline
- **Week 1**: Phase 1 - Enhanced data capture infrastructure
- **Week 2**: Phase 2 - Search infrastructure and logic
- **Week 3**: Phase 3 - HTTP endpoints and validation  
- **Week 4**: Phase 4 - Testing and documentation

## Success Criteria
1. All four search endpoints functional and tested
2. Performance acceptable for typical fake server usage (< 100ms response time)
3. Comprehensive test coverage (>90%)
4. Documentation updated and complete
5. Backward compatibility maintained