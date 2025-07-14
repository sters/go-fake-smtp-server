package fakesmtpserver

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"

	"go.uber.org/zap"
)

const VIEW_ADDR = "127.0.0.1:11080"

func StartViewServer() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r1 *http.Request) {
		buf := &bytes.Buffer{}
		enc := json.NewEncoder(buf)
		err := enc.Encode(smtpBackend.GetAllData())
		if err != nil {
			zap.L().Info("err", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		_, _ = w.Write(buf.Bytes())
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Handler: mux,
	}
	l, err := ln()
	if err != nil {
		return err
	}

	zap.L().Info("Starting HTTP server", zap.String("addr", l.Addr().String()))
	if err := server.Serve(l); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func ln() (net.Listener, error) {
	ln, err := net.Listen("tcp", VIEW_ADDR)
	return ln, err
}
