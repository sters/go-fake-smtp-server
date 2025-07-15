package fakesmtpserver

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

const ViewAddr = "127.0.0.1:11080"

// StartViewServer starts the HTTP server that serves captured emails and search endpoints.
func StartViewServer() error {
	mux := http.NewServeMux()

	// Register all handlers
	registerListHandlers(mux)
	registerSearchHandlers(mux)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	l, err := ln()
	if err != nil {
		return err
	}

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
