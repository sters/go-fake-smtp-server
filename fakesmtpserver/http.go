package fakesmtpserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

const ViewAddr = "127.0.0.1:11080"

func StartViewServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err := enc.Encode(sharedBackend.GetAllData())
		if err != nil {
			slog.Info("err", "error", err)
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = w.Write(buf.Bytes())
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	l, err := ln()
	if err != nil {
		return err
	}

	slog.Info("Starting HTTP server", "addr", l.Addr().String())
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
