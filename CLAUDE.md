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
- Uses Zap for structured logging
- Handles graceful shutdown via signal handling

### FakeSmtpServer Package (`fakesmtpserver/`)

**SMTP Server (`smtp.go`)**:
- Implements `smtp.Backend` and `smtp.Session` interfaces from `github.com/emersion/go-smtp`
- `smtpBackend` manages all SMTP sessions with thread-safe access
- `smtpSession` represents individual email sessions
- Stores email data in memory as raw strings
- Parses emails using `github.com/jhillyerd/enmime` for structured access

**HTTP Server (`http.go`)**:
- Provides a single endpoint `/` that returns all captured emails as JSON
- Transforms raw email data into structured `smtpView` format
- Extracts headers, recipients (To/Cc/Bcc), text content, and HTML content

### Key Dependencies
- `github.com/emersion/go-smtp` - SMTP server implementation
- `github.com/jhillyerd/enmime` - Email parsing and MIME handling
- `log/slog` - Structured logging (Go standard library)
- `golang.org/x/sync/errgroup` - Concurrent goroutine management

## Default Configuration

- SMTP server listens on `127.0.0.1:10025`
- HTTP server listens on `127.0.0.1:11080`
- Maximum message size: 1MB
- Maximum recipients: 50
- Allows insecure authentication

## Testing the Server

The server can be tested by sending emails to port 10025 and viewing them via HTTP on port 11080. There's commented debug code in `main.go:34-58` that demonstrates programmatic email sending for testing.