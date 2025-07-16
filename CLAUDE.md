# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a fake SMTP server written in Go that captures emails and provides a web interface to view them. The server consists of two main components:
- SMTP server (port 10025) that accepts emails
- HTTP server (port 11080) that serves captured emails as JSON

## Development Commands

Run the application:
```bash
make run
# or with arguments:
make run ARGS="--verbose"
```

Testing:
```bash
make test          # Run all tests with race detection
make cover         # Run tests with coverage

# Run specific test files or functions
go test -v ./fakesmtpserver -run TestSpecificFunction
go test -v ./...  # Run all tests in all packages
```

Linting:
```bash
make lint          # Run golangci-lint
make lint-fix      # Run golangci-lint with auto-fix
```

Other commands:
```bash
make tidy          # Clean up go.mod dependencies
```

## Architecture

The codebase is organized into two main packages:

### Main Package (`main.go`)
- Entry point that starts both SMTP and HTTP servers concurrently using errgroup
- Uses slog (standard library) for structured logging
- Handles graceful shutdown via signal handling
- Configuration loaded from environment variables via `config/config.go`

### FakeSmtpServer Package (`fakesmtpserver/`)

**SMTP Server (`smtp.go`)**:
- Implements `smtp.Backend` and `smtp.Session` interfaces from `github.com/emersion/go-smtp`
- `smtpBackend` manages all SMTP sessions with thread-safe access using `sync.RWMutex`
- `smtpSession` represents individual email sessions
- Stores email data in memory as raw strings
- Parses emails using `github.com/jhillyerd/enmime` for structured access

**HTTP Server (`http.go` and `handlers_*.go`)**:
- Main endpoint `/` returns all captured emails as JSON
- Search endpoints for filtering emails:
  - `/search/to?email=xxx` - Search by To field
  - `/search/cc?email=xxx` - Search by CC field
  - `/search/bcc?email=xxx` - Search by BCC field
  - `/search/from?email=xxx` - Search by From field
- Transforms raw email data into structured `smtpView` format
- Extracts headers, recipients (To/Cc/Bcc), text content, and HTML content

### Key Dependencies
- `github.com/emersion/go-smtp` - SMTP server implementation
- `github.com/jhillyerd/enmime` - Email parsing and MIME handling
- `github.com/caarlos0/env/v11` - Environment variable configuration
- `log/slog` - Structured logging (Go standard library)
- `golang.org/x/sync/errgroup` - Concurrent goroutine management

## Configuration

The server is configured via environment variables:

### SMTP Server Settings
- `SMTP_ADDR` - SMTP server address (default: `127.0.0.1:10025`)
- `SMTP_HOSTNAME` - SMTP server hostname (default: `fakeserver`)
- `SMTP_READ_TIMEOUT` - Read timeout (default: `10s`)
- `SMTP_WRITE_TIMEOUT` - Write timeout (default: `10s`)
- `SMTP_MAX_MESSAGE_BYTES` - Maximum message size (default: `1048576` / 1MB)
- `SMTP_MAX_RECIPIENTS` - Maximum recipients per message (default: `50`)
- `SMTP_ALLOW_INSECURE_AUTH` - Allow auth without TLS (default: `true`)

### HTTP Server Settings
- `VIEW_ADDR` - HTTP server address (default: `127.0.0.1:11080`)
- `VIEW_READ_HEADER_TIMEOUT` - HTTP read header timeout (default: `10s`)

## API Endpoints

All endpoints return JSON responses:

### List All Emails
```
GET /
```
Returns an array of all captured emails.

### Search Emails
```
GET /search/to?email=user@example.com    # Search by To field
GET /search/cc?email=user@example.com    # Search by CC field
GET /search/bcc?email=user@example.com   # Search by BCC field
GET /search/from?email=user@example.com  # Search by From field
```

### Response Format
```json
[
  {
    "headers": [
      {"key": "Subject", "value": "Test Email"},
      {"key": "Date", "value": "Mon, 01 Jan 2024 12:00:00 +0000"}
    ],
    "to": [{"Name": "John Doe", "Address": "john@example.com"}],
    "cc": [],
    "bcc": [],
    "from": [{"Name": "Jane Smith", "Address": "jane@example.com"}],
    "text": "Plain text content",
    "html": "<p>HTML content</p>"
  }
]
```

## Testing the Server

Send test emails using telnet or any SMTP client:
```bash
# Using telnet
telnet localhost 10025
HELO localhost
MAIL FROM: sender@example.com
RCPT TO: recipient@example.com
DATA
Subject: Test Email
From: sender@example.com
To: recipient@example.com

This is a test email.
.
QUIT
```

View captured emails:
```bash
curl http://localhost:11080/
curl "http://localhost:11080/search/to?email=recipient@example.com"
```

There's also commented debug code in `main.go:34-58` that demonstrates programmatic email sending for testing.