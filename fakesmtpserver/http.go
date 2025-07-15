package fakesmtpserver

import (
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/sters/go-fake-smtp-server/config"
)

// StartViewServer starts the HTTP server that serves captured emails and search endpoints.
func StartViewServer(cfg *config.Config) error {
	mux := http.NewServeMux()

	// Register all handlers
	registerListHandlers(mux)
	registerSearchHandlers(mux)

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: cfg.ViewReadHeaderTimeout,
	}

	l, err := ln(cfg)
	if err != nil {
		return err
	}

	if err := server.Serve(l); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve error: %w", err)
	}

	return nil
}

func ln(cfg *config.Config) (net.Listener, error) {
	ln, err := net.Listen("tcp", cfg.ViewAddr)
	if err != nil {
		return nil, fmt.Errorf("listen error: %w", err)
	}

	return ln, nil
}
